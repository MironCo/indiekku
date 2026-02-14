package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
func RunContainer(cfg ContainerConfig) error {
	args := []string{"run", "--rm", "-d", "--network", "host", "--name", cfg.Name, cfg.ImageName}

	// Add command and args if specified
	if cfg.Command != "" {
		args = append(args, cfg.Command)
		args = append(args, cfg.Args...)
	}

	cmd := exec.Command("docker", args...)
	return cmd.Start()
}

// StopContainer stops a running Docker container
func StopContainer(containerName string) error {
	cmd := exec.Command("docker", "stop", containerName)
	return cmd.Run()
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
