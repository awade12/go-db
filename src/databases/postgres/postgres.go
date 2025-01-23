package postgres

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

var (
	success  = color.New(color.FgGreen, color.Bold).SprintFunc()
	info     = color.New(color.FgCyan).SprintFunc()
	warn     = color.New(color.FgYellow).SprintFunc()
	errColor = color.New(color.FgRed, color.Bold).SprintFunc()
)

const (
	defaultPostgresVersion = "15"
	defaultPort            = "5432"
	defaultPassword        = "postgres"
	defaultContainerName   = "go-dbs-postgres"
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
	ContainerName string
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
	SSLMode       string            // SSL mode (disable, require, verify-ca, verify-full) -- will do automatically
	SSLCert       string            // path to SSL certificate ---- will do automatically
	SSLKey        string            // path to SSL key ---- will do automatically
	SSLRootCert   string            // path to SSL root certificate ---- will do automatically
	Timezone      string            // container timezone
	Locale        string            // database locale
}

// DefaultConfig returns a basic configuration
func DefaultConfig() *Config {
	return &Config{
		Version:       defaultPostgresVersion,
		Port:          defaultPort,
		Password:      defaultPassword,
		ContainerName: defaultContainerName,
		Username:      "postgres",
		Database:      "postgres",
		Environment:   make(map[string]string),
		SSLMode:       "disable",
		Timezone:      "UTC",
		Locale:        "en_US.utf8",
	}
}

// Create sets up a new PostgreSQL database instance using Docker with default settings
func Create() error {
	return CreateWithConfig(DefaultConfig())
}

// CreateWithConfig sets up a new PostgreSQL instance with custom configuration
func CreateWithConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("%s configuration cannot be nil", errColor("✘"))
	}

	fmt.Printf("%s Starting PostgreSQL setup...\n", info("ℹ"))

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
				cmd := exec.Command("docker", "pull", fmt.Sprintf("postgres:%s", cfg.Version))
				return cmd.Run()
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
	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		cmd := exec.Command("docker", "exec", cfg.ContainerName, "pg_isready")
		if err := cmd.Run(); err == nil {
			return nil
		}
		time.Sleep(1 * time.Second)
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
	fmt.Printf("\n%s Connection Details:\n", info("ℹ"))
	fmt.Printf("  %s Host: %s\n", info("→"), "0.0.0.0")
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
	fmt.Printf("  %s postgresql://%s:%s@0.0.0.0:%s/%s\n",
		info("→"), cfg.Username, cfg.Password, cfg.Port, cfg.Database)
}
