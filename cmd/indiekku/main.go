package main

import (
	"fmt"
	"os"

	"indiekku/internal/docker"
	"indiekku/internal/server"
)

const (
	defaultPort = "7777"
)

func main() {
	port := defaultPort
	containerName := docker.DefaultContainerPrefix + port
	imageName := docker.DefaultImageName

	// Find the server binary
	serverBinary, err := server.FindBinary(server.DefaultServerDir)
	if err != nil {
		fmt.Printf("Failed to find server binary: %v\n", err)
		fmt.Println("Please place your game server build in the 'server/' directory")
		os.Exit(1)
	}

	fmt.Printf("Found server binary: %s\n", serverBinary)

	// Check if Docker image exists, if not build it
	if !docker.ImageExists(imageName) {
		fmt.Println("Building Docker image...")
		if err := docker.BuildImage(imageName); err != nil {
			fmt.Printf("Failed to build Docker image: %v\n", err)
			os.Exit(1)
		}
	}

	// Run the container
	fmt.Printf("Starting game server container on port %s...\n", port)
	if err := docker.RunContainer(containerName, imageName, port, serverBinary); err != nil {
		fmt.Printf("Failed to start container: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Game server container started: %s\n", containerName)
	fmt.Printf("Server running on port %s\n", port)
}