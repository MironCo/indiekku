package docker

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	DefaultImageName       = "unity-server"
	DefaultContainerPrefix = "unity-server-"
)

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

// BuildImage builds a Docker image from the Dockerfile
func BuildImage(imageName string) error {
	cmd := exec.Command("docker", "build", "-t", imageName, "-f", "Dockerfile", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunContainer starts a Docker container with the specified configuration
func RunContainer(containerName, imageName, port, serverBinary string) error {
	cmd := exec.Command("docker", "run", "--rm", "-d",
		"--network", "host", // This will make Unity bind to both IPv4 and IPv6
		"--name", containerName,
		imageName,
		serverBinary, "-port", port)

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