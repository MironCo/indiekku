package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"indiekku/internal/api"
	"indiekku/internal/client"
	"indiekku/internal/docker"
	"indiekku/internal/security"
	"indiekku/internal/server"
	"indiekku/internal/state"
)

const (
	defaultPort    = "7777"
	defaultAPIPort = "8080"
	defaultAPIURL  = "http://localhost:8080"
	pidFile        = "indiekku.pid"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "serve":
		runServe()
	case "shutdown":
		runShutdown()
	case "logs":
		runLogs()
	case "start":
		runStart()
	case "stop":
		runStop()
	case "ps":
		runPs()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("indiekku - Game server orchestration tool")
	fmt.Println("\nUsage:")
	fmt.Println("  indiekku serve        Start the API server (runs in background)")
	fmt.Println("  indiekku shutdown     Stop the API server")
	fmt.Println("  indiekku logs         View API server logs")
	fmt.Println("  indiekku start [port] Start a game server container")
	fmt.Println("  indiekku stop <name>  Stop a game server container")
	fmt.Println("  indiekku ps           List running game server containers")
}

func runServe() {
	// Check if Docker is installed
	if err := docker.CheckDockerInstalled(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Ensure API key exists (generate if first run)
	apiKey, isNew, err := security.EnsureAPIKey()
	if err != nil {
		fmt.Printf("Failed to ensure API key: %v\n", err)
		os.Exit(1)
	}

	if isNew {
		fmt.Println()
		fmt.Println(strings.Repeat("=", 70))
		fmt.Println("  NEW API KEY GENERATED")
		fmt.Println(strings.Repeat("=", 70))
		fmt.Println()
		fmt.Printf("  Your API Key: %s\n\n", apiKey)
		fmt.Printf("  This key has been saved to: .indiekku_apikey\n")
		fmt.Printf("  Keep this key secure - you'll need it to authenticate API requests.\n\n")
		fmt.Printf("  Example usage:\n")
		fmt.Printf("    curl -H \"Authorization: Bearer %s\" \\\n", apiKey)
		fmt.Printf("         http://localhost:8080/api/v1/servers\n\n")
		fmt.Println(strings.Repeat("=", 70))
		fmt.Println()
	}

	// Check if this is the forked daemon process (internal flag)
	isDaemonProcess := false
	apiPort := defaultAPIPort

	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "__daemon__" {
			isDaemonProcess = true
		} else {
			apiPort = os.Args[i]
		}
	}

	// If this is NOT the daemon process, fork and exit
	if !isDaemonProcess {
		// Check if server is already running
		if _, err := os.Stat(pidFile); err == nil {
			fmt.Println("Error: API server is already running")
			fmt.Println("Use 'indiekku shutdown' to stop it first")
			os.Exit(1)
		}

		// Get the actual executable path (not from os.Args)
		execPath, err := os.Executable()
		if err != nil {
			fmt.Printf("Failed to get executable path: %v\n", err)
			os.Exit(1)
		}

		// Build args using the resolved executable path
		args := []string{"serve", "__daemon__"}
		if apiPort != defaultAPIPort {
			args = append(args, apiPort)
		}

		// Create log file for daemon output
		logFile, err := os.OpenFile("indiekku.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("Failed to create log file: %v\n", err)
			os.Exit(1)
		}

		cmd := exec.Command(execPath, args...)
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		cmd.Stdin = nil

		if err := cmd.Start(); err != nil {
			fmt.Printf("Failed to start daemon: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ indiekku API server started\n")
		fmt.Printf("  PID: %d\n", cmd.Process.Pid)
		fmt.Printf("  Port: %s\n", apiPort)
		fmt.Printf("\nUse 'indiekku logs' to view logs\n")
		fmt.Printf("Use 'indiekku shutdown' to stop\n")
		return
	}

	// This is the daemon process, start the server
	fmt.Println("Starting indiekku API server...")

	// Save PID to file
	pid := os.Getpid()
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644); err != nil {
		fmt.Printf("Warning: Failed to write PID file: %v\n", err)
	}
	defer os.Remove(pidFile)

	// Load API key for authentication
	loadedAPIKey, err := security.LoadAPIKey()
	if err != nil {
		fmt.Printf("Failed to load API key: %v\n", err)
		os.Exit(1)
	}

	// Initialize state handler
	stateHandler := state.NewStateHandler()

	// Create API handler
	apiHandler := api.NewAPIHandler(
		stateHandler,
		server.DefaultServerDir,
		docker.DefaultImageName,
		loadedAPIKey,
	)

	// Setup and run the router
	router := apiHandler.SetupRouter()

	fmt.Printf("API server listening on port %s\n", apiPort)
	fmt.Printf("PID: %d (saved to %s)\n", pid, pidFile)
	if err := router.Run(":" + apiPort); err != nil {
		fmt.Printf("Failed to start API server: %v\n", err)
		os.Exit(1)
	}
}

func runShutdown() {
	// Read PID file
	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		fmt.Printf("Error: API server is not running (no PID file found)\n")
		os.Exit(1)
	}

	// First, try to stop all running game servers via the API
	apiClient, err := client.NewClient(defaultAPIURL)
	if err == nil {
		fmt.Println("Stopping all running game servers...")
		resp, err := apiClient.ListServers()
		if err == nil && resp.Count > 0 {
			for _, server := range resp.Servers {
				fmt.Printf("  Stopping %s...\n", server.ContainerName)
				if err := apiClient.StopServer(server.ContainerName); err != nil {
					fmt.Printf("    Warning: Failed to stop %s: %v\n", server.ContainerName, err)
				} else {
					fmt.Printf("    ✓ Stopped %s\n", server.ContainerName)
				}
			}
		} else if resp.Count == 0 {
			fmt.Println("  No running servers to stop")
		}
	}

	pid := string(pidBytes)
	fmt.Printf("\nShutting down indiekku API server (PID: %s)...\n", pid)

	// Kill the process
	// Use syscall.Kill on Unix, or just use os.FindProcess
	var pidInt int
	fmt.Sscanf(pid, "%d", &pidInt)

	process, err := os.FindProcess(pidInt)
	if err != nil {
		fmt.Printf("Error: Failed to find process: %v\n", err)
		os.Exit(1)
	}

	if err := process.Kill(); err != nil {
		fmt.Printf("Error: Failed to kill process: %v\n", err)
		os.Exit(1)
	}

	// Remove PID file
	os.Remove(pidFile)

	fmt.Printf("✓ API server stopped successfully\n")
}

func runStart() {
	port := ""
	if len(os.Args) > 2 {
		port = os.Args[2]
	}

	// Create API client
	apiClient, err := client.NewClient(defaultAPIURL)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Check if API is running
	if err := apiClient.HealthCheck(); err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nMake sure the indiekku API server is running:")
		fmt.Println("  ./indiekku serve")
		os.Exit(1)
	}

	// Start the server via API
	fmt.Printf("Starting game server...\n")
	resp, err := apiClient.StartServer(port)
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ %s\n", resp.Message)
	fmt.Printf("  Container: %s\n", resp.ContainerName)
	fmt.Printf("  Port: %s\n", resp.Port)
}

func runStop() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: indiekku stop <container_name>")
		os.Exit(1)
	}

	containerName := os.Args[2]

	// Create API client
	apiClient, err := client.NewClient(defaultAPIURL)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Check if API is running
	if err := apiClient.HealthCheck(); err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nMake sure the indiekku API server is running:")
		fmt.Println("  ./indiekku serve")
		os.Exit(1)
	}

	// Stop the server via API
	fmt.Printf("Stopping container: %s\n", containerName)
	if err := apiClient.StopServer(containerName); err != nil {
		fmt.Printf("Failed to stop server: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Container %s stopped successfully\n", containerName)
}

func runPs() {
	// Create API client
	apiClient, err := client.NewClient(defaultAPIURL)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Check if API is running
	if err := apiClient.HealthCheck(); err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nMake sure the indiekku API server is running:")
		fmt.Println("  ./indiekku serve")
		os.Exit(1)
	}

	// List servers via API
	resp, err := apiClient.ListServers()
	if err != nil {
		fmt.Printf("Failed to list servers: %v\n", err)
		os.Exit(1)
	}

	if resp.Count == 0 {
		fmt.Println("No game servers running")
		return
	}

	fmt.Printf("Running game servers (%d):\n\n", resp.Count)
	fmt.Printf("%-25s %-10s %-10s %-20s\n", "CONTAINER", "PORT", "PLAYERS", "STARTED")
	fmt.Println("--------------------------------------------------------------------------------")
	for _, s := range resp.Servers {
		fmt.Printf("%-25s %-10s %-10d %-20s\n",
			s.ContainerName,
			s.Port,
			s.PlayerCount,
			s.StartedAt.Format("2006-01-02 15:04:05"),
		)
	}
}

func runLogs() {
	// Check if log file exists
	if _, err := os.Stat("indiekku.log"); os.IsNotExist(err) {
		fmt.Println("No log file found. Has the server been started?")
		os.Exit(1)
	}

	// Open and follow the file in Go (cross-platform)
	file, err := os.Open("indiekku.log")
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Read existing content
	content, _ := io.ReadAll(file)
	fmt.Print(string(content))

	fmt.Println("\nFollowing logs (Ctrl+C to exit)...")

	// Tail the file
	for {
		line := make([]byte, 1024)
		n, err := file.Read(line)
		if n > 0 {
			fmt.Print(string(line[:n]))
		}
		if err == io.EOF {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if err != nil {
			break
		}
	}
}
