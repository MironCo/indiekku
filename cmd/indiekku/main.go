package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	defaultPort      = "7777"
	imageName        = "unity-server"
	containerPrefix  = "unity-server-"
	serverDir        = "server"
)

func main() {
	port := defaultPort
	containerName := containerPrefix + port

	// Find the server binary
	serverBinary, err := findServerBinary()
	if err != nil {
		fmt.Printf("Failed to find server binary: %v\n", err)
		fmt.Println("Please place your Unity server build in the 'server/' directory")
		os.Exit(1)
	}

	fmt.Printf("Found server binary: %s\n", serverBinary)

	// Check if Docker image exists, if not build it
	if !imageExists(imageName) {
		fmt.Println("Building Docker image...")
		if err := buildDockerImage(); err != nil {
			fmt.Printf("Failed to build Docker image: %v\n", err)
			os.Exit(1)
		}
	}

	// Run the container
	if err := runContainer(containerName, port, serverBinary); err != nil {
		fmt.Printf("Failed to start container: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Unity server container started: %s\n", containerName)
}

// findServerBinary scans the server directory for Unity server executables
func findServerBinary() (string, error) {
	entries, err := os.ReadDir(serverDir)
	if err != nil {
		return "", fmt.Errorf("could not read server directory: %w", err)
	}

	// Look for common Unity server binary patterns
	patterns := []string{".x86_64", ".exe"}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		for _, pattern := range patterns {
			if strings.HasSuffix(name, pattern) {
				return filepath.Join("/app", name), nil
			}
		}
	}

	return "", fmt.Errorf("no Unity server binary found in %s/ (looking for *.x86_64 or *.exe)", serverDir)
}

// runContainer starts a Docker container with the Unity server
func runContainer(containerName, port, serverBinary string) error {
	cmd := exec.Command("docker", "run", "--rm", "-d",
		"--network", "host", // This will make Unity bind to both IPv4 and IPv6
		"--name", containerName,
		imageName,
		serverBinary, "-port", port)

	fmt.Printf("Starting Unity server container on port %s...\n", port)

	if err := cmd.Start(); err != nil {
		return err
	}

	return nil
}

// imageExists checks if a Docker image exists locally
func imageExists(imageName string) bool {
	cmd := exec.Command("docker", "images", "-q", imageName)
	output, err := cmd.Output()
	return err == nil && len(output) > 0
}

// buildDockerImage builds the Docker image
func buildDockerImage() error {
	cmd := exec.Command("docker", "build", "-t", imageName, "-f", "Dockerfile", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}