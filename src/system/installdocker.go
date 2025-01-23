package system

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
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

// InstallDocker installs Docker on the current system
func InstallDocker() error {
	fmt.Printf("%s Docker installation started\n", info("ℹ"))

	switch runtime.GOOS {
	case "linux":
		return installDockerLinux()
	case "darwin":
		return installDockerDarwin()
	default:
		return fmt.Errorf("%s unsupported operating system: %s", errColor("✘"), runtime.GOOS)
	}
}

func installDockerLinux() error {
	// Check if Docker is already installed
	if _, err := exec.LookPath("docker"); err == nil {
		fmt.Printf("%s Docker is already installed\n", success("✔"))
		return nil
	}

	fmt.Printf("%s Detecting Linux distribution...\n", info("ℹ"))

	// Detect the Linux distribution
	if _, err := os.Stat("/etc/debian_version"); err == nil {
		fmt.Printf("%s Debian/Ubuntu detected\n", success("✔"))
		return installDockerDebian()
	} else if _, err := os.Stat("/etc/redhat-release"); err == nil {
		fmt.Printf("%s RHEL/CentOS/Fedora detected\n", success("✔"))
		return installDockerRHEL()
	}

	return fmt.Errorf("%s unsupported Linux distribution", errColor("✘"))
}

func installDockerDebian() error {
	steps := []struct {
		name    string
		command []string
	}{
		{"Updating package list", []string{"apt-get", "update"}},
		{"Installing prerequisites", []string{"apt-get", "install", "-y", "ca-certificates", "curl", "gnupg"}},
		{"Creating keyring directory", []string{"install", "-m", "0755", "-d", "/etc/apt/keyrings"}},
		{"Downloading Docker GPG key", []string{"curl", "-fsSL", "https://download.docker.com/linux/ubuntu/gpg", "-o", "/etc/apt/keyrings/docker.gpg"}},
		{"Setting up GPG key permissions", []string{"chmod", "a+r", "/etc/apt/keyrings/docker.gpg"}},
		{"Adding Docker repository", []string{"echo", "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable", ">", "/etc/apt/sources.list.d/docker.list"}},
		{"Updating package list", []string{"apt-get", "update"}},
		{"Installing Docker", []string{"apt-get", "install", "-y", "docker-ce", "docker-ce-cli", "containerd.io", "docker-buildx-plugin", "docker-compose-plugin"}},
	}

	return executeSteps(steps)
}

func installDockerRHEL() error {
	steps := []struct {
		name    string
		command []string
	}{
		{"Installing DNF plugins", []string{"dnf", "install", "-y", "dnf-plugins-core"}},
		{"Adding Docker repository", []string{"dnf", "config-manager", "--add-repo", "https://download.docker.com/linux/fedora/docker-ce.repo"}},
		{"Installing Docker", []string{"dnf", "install", "-y", "docker-ce", "docker-ce-cli", "containerd.io", "docker-buildx-plugin", "docker-compose-plugin"}},
	}

	return executeSteps(steps)
}

func installDockerDarwin() error {
	// Check if Docker is already installed
	if _, err := exec.LookPath("docker"); err == nil {
		fmt.Printf("%s Docker is already installed\n", success("✔"))
		return nil
	}

	fmt.Printf("\n%s For macOS, please install Docker Desktop manually:\n", info("ℹ"))
	fmt.Printf("%s 1. Visit %s\n", info("→"), "https://www.docker.com/products/docker-desktop")
	fmt.Printf("%s 2. Download and install Docker Desktop for Mac\n", info("→"))
	fmt.Printf("%s 3. Follow the installation instructions\n", info("→"))
	return fmt.Errorf("%s manual installation required for macOS", warn("⚠"))
}

func executeSteps(steps []struct {
	name    string
	command []string
}) error {
	totalSteps := len(steps)
	bar := progressbar.NewOptions(totalSteps,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(30),
		progressbar.OptionSetDescription("[cyan]Installing Docker[reset]"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	for i, step := range steps {
		bar.Describe(fmt.Sprintf("[cyan]%s[reset]", step.name))

		command := exec.Command("sudo", step.command...)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr

		if err := command.Run(); err != nil {
			fmt.Printf("\n%s Failed to %s: %v\n", errColor("✘"), step.name, err)
			return fmt.Errorf("failed to execute step %d: %v", i+1, err)
		}

		bar.Add(1)
		time.Sleep(100 * time.Millisecond) // Small delay for visual feedback
	}

	fmt.Printf("\n%s Docker installation completed successfully!\n", success("✔"))
	fmt.Printf("%s Starting Docker service...\n", info("ℹ"))

	// Start Docker service
	startCmd := exec.Command("sudo", "systemctl", "start", "docker")
	if err := startCmd.Run(); err != nil {
		return fmt.Errorf("%s failed to start Docker service: %v", errColor("✘"), err)
	}

	fmt.Printf("%s Docker service started\n", success("✔"))
	fmt.Printf("%s Adding current user to docker group...\n", info("ℹ"))

	// Add user to docker group
	username := os.Getenv("USER")
	groupCmd := exec.Command("sudo", "usermod", "-aG", "docker", username)
	if err := groupCmd.Run(); err != nil {
		fmt.Printf("%s Warning: Could not add user to docker group: %v\n", warn("⚠"), err)
		fmt.Printf("%s You may need to use 'sudo' with docker commands\n", warn("⚠"))
	} else {
		fmt.Printf("%s User added to docker group\n", success("✔"))
		fmt.Printf("%s Please log out and back in for this to take effect\n", info("ℹ"))
	}

	return nil
}
