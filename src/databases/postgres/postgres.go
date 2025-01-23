package postgres

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/awade12/go-db/src/utils"
	"github.com/schollz/progressbar/v3"
)

var (
	success  = utils.Success
	info     = utils.Info
	warn     = utils.Warn
	errColor = utils.ErrColor
)

const (
	defaultPostgresVersion = "15"
	defaultPort            = "5432"
)

// findAvailablePort finds an available port starting from the given port
func findAvailablePort(startPort int) (int, error) {
	for port := startPort; port < startPort+100; port++ {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found in range %d-%d", startPort, startPort+100)
}

// Config holds PostgreSQL configuration options
type Config struct {
	Version       string
	Port          string
	Password      string
	ContainerName string // required: name of the container
	Username      string
	Database      string
	Volume        string            // for persistent storage
	Memory        string            // memory limit
	CPU           string            // CPU limit
	Replicas      int               // number of replicas for HA
	InitScripts   []string          // paths to initialization SQL scripts
	Environment   map[string]string // additional environment variables
	Networks      []string          // docker networks to join
	ExtraMounts   []string          // additional volume mounts
	SSLMode       string            // SSL mode (disable, require, verify-ca, verify-full)
	SSLCert       string            // path to SSL certificate
	SSLKey        string            // path to SSL key
	SSLRootCert   string            // path to SSL root certificate
	Timezone      string            // container timezone
	Locale        string            // database locale
}

func DefaultConfig(name string) *Config {
	if name == "" {
		name = "postgres-" + time.Now().Format("20060102-150405")
	}
	return &Config{
		Version:       defaultPostgresVersion,
		Port:          defaultPort,
		Password:      utils.GenerateSecurePassword(),
		ContainerName: name,
		Username:      "postgres",
		Database:      name, // Use the container name as the default database name
		Environment:   make(map[string]string),
		SSLMode:       "disable",
		Timezone:      "UTC",
		Locale:        "en_US.utf8",
	}
}

// Create sets up a new PostgreSQL database instance using Docker with default settings
func Create(name string) error {
	return CreateWithConfig(DefaultConfig(name))
}

// CreateWithConfig sets up a new PostgreSQL instance with custom configuration
func CreateWithConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("%s configuration cannot be nil", errColor("✘"))
	}

	if cfg.ContainerName == "" {
		return fmt.Errorf("%s container name is required", errColor("✘"))
	}

	fmt.Printf("%s Starting PostgreSQL setup for %s...\n", info("ℹ"), cfg.ContainerName)

	// Check if Docker is installed
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("%s Docker is not installed: %v", errColor("✘"), err)
	}

	// Check if container already exists
	if exists, _ := containerExists(cfg.ContainerName); exists {
		return fmt.Errorf("%s Container %s already exists. Use 'go-db remove %s' to remove it first",
			errColor("✘"), cfg.ContainerName, cfg.ContainerName)
	}

	// Find available port if default is taken
	if cfg.Port == defaultPort {
		port, err := findAvailablePort(5432)
		if err != nil {
			return fmt.Errorf("%s Failed to find available port: %v", errColor("✘"), err)
		}
		cfg.Port = fmt.Sprintf("%d", port)
		if cfg.Port != defaultPort {
			fmt.Printf("%s Port %s was taken, using port %s instead\n", info("ℹ"), defaultPort, cfg.Port)
		}
	}

	steps := []struct {
		name string
		fn   func() error
	}{
		{
			name: "Pulling PostgreSQL image",
			fn: func() error {
				// Only pull if image doesn't exist
				if out, _ := exec.Command("docker", "images", "-q", fmt.Sprintf("postgres:%s", cfg.Version)).Output(); len(out) == 0 {
					cmd := exec.Command("docker", "pull", fmt.Sprintf("postgres:%s", cfg.Version))
					return cmd.Run()
				}
				return nil
			},
		},
		{
			name: "Creating container",
			fn: func() error {
				args := buildDockerArgs(cfg)
				cmd := exec.Command("docker", args...)
				return cmd.Run()
			},
		},
		{
			name: "Waiting for container to be ready",
			fn: func() error {
				return waitForPostgres(cfg)
			},
		},
	}

	bar := progressbar.NewOptions(len(steps),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(30),
		progressbar.OptionSetDescription("[cyan]Setting up PostgreSQL[reset]"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	for _, step := range steps {
		bar.Describe(fmt.Sprintf("[cyan]%s[reset]", step.name))
		if err := step.fn(); err != nil {
			fmt.Printf("\n%s %s failed: %v\n", errColor("✘"), step.name, err)
			return fmt.Errorf("failed during %s: %v", step.name, err)
		}
		bar.Add(1)
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("\n%s PostgreSQL container created successfully!\n", success("✔"))
	printConnectionDetails(cfg)

	return nil
}

func buildDockerArgs(cfg *Config) []string {
	args := []string{
		"run",
		"--name", cfg.ContainerName,
		"-e", fmt.Sprintf("POSTGRES_PASSWORD=%s", cfg.Password),
		"-e", fmt.Sprintf("POSTGRES_USER=%s", cfg.Username),
		"-e", fmt.Sprintf("POSTGRES_DB=%s", cfg.Database),
		"-e", fmt.Sprintf("TZ=%s", cfg.Timezone),
		"-e", fmt.Sprintf("LANG=%s", cfg.Locale),
		"-p", fmt.Sprintf("%s:5432", cfg.Port),
		"-d",
	}

	// Add environment variables
	for k, v := range cfg.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add optional configurations
	if cfg.Volume != "" {
		args = append(args, "-v", fmt.Sprintf("%s:/var/lib/postgresql/data", cfg.Volume))
	}
	if cfg.Memory != "" {
		args = append(args, "--memory", cfg.Memory)
	}
	if cfg.CPU != "" {
		args = append(args, "--cpus", cfg.CPU)
	}

	// Add networks
	for _, network := range cfg.Networks {
		args = append(args, "--network", network)
	}

	// Add extra mounts
	for _, mount := range cfg.ExtraMounts {
		args = append(args, "-v", mount)
	}

	// Handle SSL configuration
	if cfg.SSLMode != "disable" {
		if cfg.SSLCert != "" && cfg.SSLKey != "" {
			args = append(args, "-v", fmt.Sprintf("%s:/var/lib/postgresql/server.crt", cfg.SSLCert))
			args = append(args, "-v", fmt.Sprintf("%s:/var/lib/postgresql/server.key", cfg.SSLKey))
			if cfg.SSLRootCert != "" {
				args = append(args, "-v", fmt.Sprintf("%s:/var/lib/postgresql/root.crt", cfg.SSLRootCert))
			}
		}
	}

	// Handle initialization scripts
	if len(cfg.InitScripts) > 0 {
		for i, script := range cfg.InitScripts {
			args = append(args, "-v", fmt.Sprintf("%s:/docker-entrypoint-initdb.d/init_%d.sql:ro", script, i))
		}
	}

	// Add image name
	args = append(args, fmt.Sprintf("postgres:%s", cfg.Version))

	return args
}

func waitForPostgres(cfg *Config) error {
	maxAttempts := 10 // Reduced from 30
	for i := 0; i < maxAttempts; i++ {
		cmd := exec.Command("docker", "exec", cfg.ContainerName, "pg_isready")
		if err := cmd.Run(); err == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond) // Reduced from 1 second
	}
	return fmt.Errorf("timeout waiting for PostgreSQL to be ready")
}

func Stop(containerName string) error {
	if exists, running := containerExists(containerName); !exists {
		return fmt.Errorf("%s Container %s does not exist", errColor("✘"), containerName)
	} else if !running {
		return fmt.Errorf("%s Container %s is already stopped", warn("⚠"), containerName)
	}

	fmt.Printf("%s Stopping container %s...\n", info("ℹ"), containerName)
	cmd := exec.Command("docker", "stop", containerName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s Failed to stop container: %v", errColor("✘"), err)
	}

	fmt.Printf("%s Container %s stopped successfully\n", success("✔"), containerName)
	return nil
}

func Start(containerName string) error {
	if exists, running := containerExists(containerName); !exists {
		return fmt.Errorf("%s Container %s does not exist", errColor("✘"), containerName)
	} else if running {
		return fmt.Errorf("%s Container %s is already running", warn("⚠"), containerName)
	}

	fmt.Printf("%s Starting container %s...\n", info("ℹ"), containerName)
	cmd := exec.Command("docker", "start", containerName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s Failed to start container: %v", errColor("✘"), err)
	}

	fmt.Printf("%s Container %s started successfully\n", success("✔"), containerName)
	return nil
}

func Remove(containerName string, force bool) error {
	if exists, _ := containerExists(containerName); !exists {
		return fmt.Errorf("%s Container %s does not exist", errColor("✘"), containerName)
	}

	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, containerName)

	fmt.Printf("%s Removing container %s...\n", info("ℹ"), containerName)
	cmd := exec.Command("docker", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s Failed to remove container: %v", errColor("✘"), err)
	}

	fmt.Printf("%s Container %s removed successfully\n", success("✔"), containerName)
	return nil
}

func containerExists(name string) (exists bool, running bool) {
	out, err := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Status}}").Output()
	if err != nil {
		return false, false
	}

	status := strings.TrimSpace(string(out))
	if status == "" {
		return false, false
	}

	return true, strings.HasPrefix(status, "Up")
}

func printConnectionDetails(cfg *Config) {
	// Get server IP
	serverIP, err := utils.GetOutboundIP()
	if err != nil {
		serverIP = "localhost" // Fallback to localhost if IP detection fails
		fmt.Printf("%s Warning: Could not detect server IP, using localhost\n", warn("⚠"))
	}

	fmt.Printf("\n%s Connection Details:\n", info("ℹ"))
	fmt.Printf("  %s Host: %s\n", info("→"), serverIP)
	fmt.Printf("  %s Port: %s\n", info("→"), cfg.Port)
	fmt.Printf("  %s User: %s\n", info("→"), cfg.Username)
	fmt.Printf("  %s Password: %s\n", info("→"), cfg.Password)
	fmt.Printf("  %s Database: %s\n", info("→"), cfg.Database)
	if cfg.Volume != "" {
		fmt.Printf("  %s Data Volume: %s\n", info("→"), cfg.Volume)
	}
	if cfg.SSLMode != "disable" {
		fmt.Printf("  %s SSL Mode: %s\n", info("→"), cfg.SSLMode)
	}

	fmt.Printf("\n%s Management Commands:\n", info("ℹ"))
	fmt.Printf("  %s Stop:    go-db stop %s\n", info("→"), cfg.ContainerName)
	fmt.Printf("  %s Start:   go-db start %s\n", info("→"), cfg.ContainerName)
	fmt.Printf("  %s Remove:  go-db remove %s\n", info("→"), cfg.ContainerName)
	fmt.Printf("  %s Logs:    docker logs %s\n", info("→"), cfg.ContainerName)

	fmt.Printf("\n%s Connection String:\n", info("ℹ"))
	fmt.Printf("  %s postgresql://%s:%s@%s:%s/%s\n",
		info("→"), cfg.Username, cfg.Password, serverIP, cfg.Port, cfg.Database)

	// Try to get public IP for external access
	publicIP, err := utils.GetPublicIP()
	if err == nil && publicIP != serverIP {
		fmt.Printf("\n%s External Connection String:\n", info("ℹ"))
		fmt.Printf("  %s postgresql://%s:%s@%s:%s/%s\n",
			info("→"), cfg.Username, cfg.Password, publicIP, cfg.Port, cfg.Database)
	}
}

// List displays all PostgreSQL containers (both running and stopped)
func List() error {
	fmt.Printf("\n%s PostgreSQL Containers\n", info("📦"))

	cmd := exec.Command("docker", "ps", "-a", "--filter", "ancestor=postgres:15", "--format", "{{.Names}}\t{{.Status}}\t{{.Ports}}\t{{.ID}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("%s Failed to list containers: %v", errColor("✘"), err)
	}

	if len(output) == 0 {
		// Try again with a more general filter if no containers found
		cmd = exec.Command("docker", "ps", "-a", "--filter", "ancestor=postgres", "--format", "{{.Names}}\t{{.Status}}\t{{.Ports}}\t{{.ID}}")
		output, err = cmd.Output()
		if err != nil {
			return fmt.Errorf("%s Failed to list containers: %v", errColor("✘"), err)
		}
	}

	if len(output) == 0 {
		fmt.Printf("\n  %s No PostgreSQL containers found\n\n", warn("⚠"))
		return nil
	}

	// Print header with custom formatting
	fmt.Printf("\n  %-20s %-15s %-15s %s\n", "NAME", "STATUS", "PORT", "CONTAINER ID")
	fmt.Printf("  %s\n", strings.Repeat("─", 80))

	containers := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, container := range containers {
		fields := strings.Split(container, "\t")
		if len(fields) >= 3 {
			name := fields[0]
			status := fields[1]
			ports := fields[2]
			id := ""
			if len(fields) > 3 {
				id = fields[3][:12] // Show first 12 chars of container ID
			}

			// Extract just the host port for cleaner display
			port := "N/A"
			if portMatch := strings.Split(ports, ":"); len(portMatch) > 1 {
				port = strings.Split(portMatch[1], "-")[0]
			}

			// Status formatting
			statusColor := warn
			statusSymbol := "🔴" // Red circle for stopped
			if strings.HasPrefix(status, "Up") {
				statusColor = success
				statusSymbol = "🟢" // Green circle for running
			}

			// Format the status to be more concise
			shortStatus := "Stopped ⏹️"
			if strings.HasPrefix(status, "Up") {
				upTime := strings.TrimPrefix(status, "Up ")
				shortStatus = "Running ⏵️ " + upTime
			}

			fmt.Printf("  %-20s %s  %-25s%s %-15s %s\n",
				info(name),
				statusSymbol,
				statusColor(shortStatus),
				utils.ResetColor(),
				port,
				id)
		}
	}
	fmt.Println()
	return nil
}

// ShowConnectionDetails displays connection information for a specific container
func ShowConnectionDetails(containerName string) error {
	if exists, _ := containerExists(containerName); !exists {
		return fmt.Errorf("%s Container %s does not exist", errColor("✘"), containerName)
	}

	// Get container details using docker inspect
	cmd := exec.Command("docker", "inspect",
		"--format",
		"{{range $k, $v := .Config.Env}}{{$v}}{{println}}{{end}}",
		containerName)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("%s Failed to get container details: %v", errColor("✘"), err)
	}

	// Parse environment variables
	env := make(map[string]string)
	for _, line := range strings.Split(string(output), "\n") {
		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}

	// Get port mapping
	cmd = exec.Command("docker", "inspect",
		"--format",
		"{{range $p, $conf := .NetworkSettings.Ports}}{{if eq $p \"5432/tcp\"}}{{range $conf}}{{.HostPort}}{{end}}{{end}}{{end}}",
		containerName)
	portBytes, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("%s Failed to get port mapping: %v", errColor("✘"), err)
	}
	port := strings.TrimSpace(string(portBytes))

	// Create a temporary config to reuse the existing printConnectionDetails function
	cfg := &Config{
		ContainerName: containerName,
		Port:          port,
		Username:      strings.TrimPrefix(env["POSTGRES_USER"], "POSTGRES_USER="),
		Password:      strings.TrimPrefix(env["POSTGRES_PASSWORD"], "POSTGRES_PASSWORD="),
		Database:      strings.TrimPrefix(env["POSTGRES_DB"], "POSTGRES_DB="),
	}

	if cfg.Username == "" {
		cfg.Username = "postgres" // default username if not set
	}
	if cfg.Database == "" {
		cfg.Database = cfg.Username // default database if not set
	}

	printConnectionDetails(cfg)
	return nil
}
