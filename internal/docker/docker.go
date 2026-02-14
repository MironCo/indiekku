package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultImageName       = "indiekku-server"
	DefaultContainerPrefix = "indiekku-"
)

// ContainerConfig holds all container run options
type ContainerConfig struct {
	Name      string   // Container name
	ImageName string   // Docker image name
	Port      string   // Port to expose (used for tracking, not necessarily passed to container)
	Command   string   // Override CMD (empty = use Dockerfile CMD)
	Args      []string // Override args (empty = use Dockerfile defaults)
}

// CheckDockerInstalled checks if Docker is installed and running
func CheckDockerInstalled() error {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker is not installed or not running. Please install Docker and start the Docker daemon")
	}
	return nil
}

// ImageExists checks if a Docker image exists locally
func ImageExists(imageName string) bool {
	cmd := exec.Command("docker", "images", "-q", imageName)
	output, err := cmd.Output()
	return err == nil && len(output) > 0
}

// BuildImage builds a Docker image using the active Dockerfile
func BuildImage(imageName string) error {
	content, err := GetActiveDockerfile()
	if err != nil {
		return fmt.Errorf("failed to get active Dockerfile: %w", err)
	}
	return BuildImageFromContent(imageName, content)
}

// BuildImageFromContent builds a Docker image from the provided Dockerfile content
func BuildImageFromContent(imageName, dockerfileContent string) error {
	tempDir := os.TempDir()
	dockerfilePath := filepath.Join(tempDir, "Dockerfile.indiekku")

	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}
	defer os.Remove(dockerfilePath)

	cmd := exec.Command("docker", "build", "-t", imageName, "-f", dockerfilePath, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunContainer starts a Docker container with the specified configuration
// Returns an error if the container fails to start or exits immediately
func RunContainer(cfg ContainerConfig) error {
	// Don't use --rm initially so we can get logs if it fails
	args := []string{"run", "-d", "--network", "host", "--name", cfg.Name, cfg.ImageName}

	// Add command and args if specified
	if cfg.Command != "" {
		args = append(args, cfg.Command)
		args = append(args, cfg.Args...)
	}

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start container: %w, output: %s", err, string(output))
	}

	containerID := strings.TrimSpace(string(output))

	// Wait a moment for the container to either stabilize or crash
	time.Sleep(2 * time.Second)

	// Verify container is actually running (not exited immediately)
	if running, logs := IsContainerRunning(cfg.Name); !running {
		// Clean up the stopped container
		cleanupCmd := exec.Command("docker", "rm", "-f", cfg.Name)
		cleanupCmd.Run() // Ignore errors

		if logs == "" {
			return fmt.Errorf("container exited immediately (ID: %s). No logs available", containerID)
		}
		return fmt.Errorf("container exited immediately. Logs:\n%s", logs)
	}

	return nil
}

// IsContainerRunning checks if a container is running and returns logs if not
func IsContainerRunning(containerName string) (bool, string) {
	// Check if container exists and is running
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", containerName)
	output, err := cmd.Output()
	if err != nil {
		// Container doesn't exist, try to get logs from exited container
		logs := getRecentLogs(containerName)
		return false, logs
	}

	running := string(output) == "true\n"
	if !running {
		logs := getRecentLogs(containerName)
		return false, logs
	}

	return true, ""
}

// getRecentLogs retrieves the last 50 lines of logs from a container (internal helper)
func getRecentLogs(containerName string) string {
	cmd := exec.Command("docker", "logs", "--tail", "50", containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("(could not retrieve logs: %v)", err)
	}
	return string(output)
}

// StopContainer stops and removes a Docker container
func StopContainer(containerName string) error {
	// Stop the container
	stopCmd := exec.Command("docker", "stop", containerName)
	if err := stopCmd.Run(); err != nil {
		return err
	}

	// Remove the container (since we no longer use --rm)
	rmCmd := exec.Command("docker", "rm", containerName)
	rmCmd.Run() // Ignore errors - container might already be removed

	return nil
}

// ListContainers lists all running containers with a given name prefix
func ListContainers(nameFilter string) ([]byte, error) {
	cmd := exec.Command("docker", "ps", "--filter", "name="+nameFilter, "--format", "{{.ID}}\t{{.Names}}\t{{.Status}}\t{{.Ports}}")
	return cmd.Output()
}

// RemoveImage removes a Docker image
func RemoveImage(imageName string) error {
	cmd := exec.Command("docker", "rmi", "-f", imageName)
	return cmd.Run()
}
