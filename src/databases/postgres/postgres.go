package postgres

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	defaultPostgresVersion = "15"
	defaultPort            = "5432"
	defaultPassword        = "postgres"
	defaultContainerName   = "go-dbs-postgres"
)

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
		return fmt.Errorf("configuration cannot be nil")
	}

	// Check if Docker is installed
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker is not installed: %v", err)
	}

	// Check if container already exists
	if exists, _ := containerExists(cfg.ContainerName); exists {
		return fmt.Errorf("container %s already exists. use 'docker rm %s' to remove it first", cfg.ContainerName, cfg.ContainerName)
	}

	// Pull the PostgreSQL image
	pullCmd := exec.Command("docker", "pull", fmt.Sprintf("postgres:%s", cfg.Version))
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("failed to pull PostgreSQL image: %v", err)
	}

	// Build docker run command with all options
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

	// Create and start the PostgreSQL container
	createCmd := exec.Command("docker", args...)
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create PostgreSQL container: %v", err)
	}

	fmt.Printf("PostgreSQL container created successfully!\n")
	printConnectionDetails(cfg)

	return nil
}

func Stop(containerName string) error {
	if exists, running := containerExists(containerName); !exists {
		return fmt.Errorf("container %s does not exist", containerName)
	} else if !running {
		return fmt.Errorf("container %s is already stopped", containerName)
	}

	cmd := exec.Command("docker", "stop", containerName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop container: %v", err)
	}

	fmt.Printf("Container %s stopped successfully\n", containerName)
	return nil
}

func Start(containerName string) error {
	if exists, running := containerExists(containerName); !exists {
		return fmt.Errorf("container %s does not exist", containerName)
	} else if running {
		return fmt.Errorf("container %s is already running", containerName)
	}

	cmd := exec.Command("docker", "start", containerName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	fmt.Printf("Container %s started successfully\n", containerName)
	return nil
}

func Remove(containerName string, force bool) error {
	if exists, _ := containerExists(containerName); !exists {
		return fmt.Errorf("container %s does not exist", containerName)
	}

	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, containerName)

	cmd := exec.Command("docker", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove container: %v", err)
	}

	fmt.Printf("Container %s removed successfully\n", containerName)
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

// ? will print the conn details for the containers
func printConnectionDetails(cfg *Config) {
	fmt.Printf("Connection details:\n")
	fmt.Printf("  Host: localhost\n")
	fmt.Printf("  Port: %s\n", cfg.Port)
	fmt.Printf("  User: %s\n", cfg.Username)
	fmt.Printf("  Password: %s\n", cfg.Password)
	fmt.Printf("  Database: %s\n", cfg.Database)
	if cfg.Volume != "" {
		fmt.Printf("  Data Volume: %s\n", cfg.Volume)
	}
	if cfg.SSLMode != "disable" {
		fmt.Printf("  SSL Mode: %s\n", cfg.SSLMode)
	}
	fmt.Printf("\nManagement Commands:\n")
	fmt.Printf("  Stop:    docker stop %s\n", cfg.ContainerName)
	fmt.Printf("  Start:   docker start %s\n", cfg.ContainerName)
	fmt.Printf("  Remove:  docker rm %s\n", cfg.ContainerName)
	fmt.Printf("  Logs:    docker logs %s\n", cfg.ContainerName)
}
