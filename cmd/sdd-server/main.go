package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	port := "7777"
	containerName := "unity-server-" + port

	// Check if Docker image exists, if not build it
	if !imageExists("unity-server") {
		fmt.Println("Building Docker image...")
		if err := buildDockerImage(); err != nil {
			fmt.Printf("Failed to build Docker image: %v\n", err)
			os.Exit(1)
		}
	}

	cmd := exec.Command("docker", "run", "--rm", "-d",
		"--network", "host", // This will make Unity bind to both IPv4 and IPv6
		"--name", containerName,
		"unity-server",
		"./sdd-server-build.x86_64", "-port", "7777")

	// Start the container
	fmt.Printf("Starting Unity server container on port %s...\n", port)
	if err := cmd.Start(); err != nil { // Use Run() instead of Start()
		fmt.Printf("Failed to start container: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Unity server container started: %s\n", containerName)
}

func imageExists(imageName string) bool {
	cmd := exec.Command("docker", "images", "-q", imageName)
	output, err := cmd.Output()
	return err == nil && len(output) > 0
}

func buildDockerImage() error {
	cmd := exec.Command("docker", "build", "-t", "unity-server", "-f", "Dockerfile", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
