package main

import (
	"fmt"
	"os"
	"os/exec"

	"indiekku/internal/api"
	"indiekku/internal/client"
	"indiekku/internal/docker"
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
	fmt.Println("  indiekku serve        Start the API server")
	fmt.Println("  indiekku shutdown     Stop the API server")
	fmt.Println("  indiekku start [port] Start a game server container")
	fmt.Println("  indiekku stop <name>  Stop a game server container")
	fmt.Println("  indiekku ps           List running game server containers")
}

func runServe() {
	_, err := server.FindBinary(server.DefaultServerDir)
	if err != nil {
		fmt.Printf("Failed to find server binary: %v\n", err)
		os.Exit(1)
	}

	// Check if --daemon or -d flag is present
	daemon := false
	apiPort := defaultAPIPort

	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "--daemon" || os.Args[i] == "-d" {
			daemon = true
		} else {
			apiPort = os.Args[i]
		}
	}

	// If daemon mode, fork and exit parent
	if daemon {
		// Re-run ourselves without the daemon flag
		args := []string{os.Args[0], "serve"}
		if apiPort != defaultAPIPort {
			args = append(args, apiPort)
		}

		// Create log file for daemon output
		logFile, err := os.OpenFile("indiekku.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("Failed to create log file: %v\n", err)
			os.Exit(1)
		}

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		cmd.Stdin = nil

		if err := cmd.Start(); err != nil {
			fmt.Printf("Failed to start daemon: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ indiekku API server started in background\n")
		fmt.Printf("  PID: %d\n", cmd.Process.Pid)
		fmt.Printf("  Port: %s\n", apiPort)
		fmt.Printf("  Logs: indiekku.log\n")
		fmt.Printf("\nTo stop: ./indiekku shutdown\n")
		return
	}

	fmt.Println("Starting indiekku API server...")

	// Save PID to file
	pid := os.Getpid()
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644); err != nil {
		fmt.Printf("Warning: Failed to write PID file: %v\n", err)
	}
	defer os.Remove(pidFile)

	// Initialize state handler
	stateHandler := state.NewStateHandler()

	// Create API handler
	apiHandler := api.NewAPIHandler(
		stateHandler,
		server.DefaultServerDir,
		docker.DefaultImageName,
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

	pid := string(pidBytes)
	fmt.Printf("Shutting down indiekku API server (PID: %s)...\n", pid)

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
	apiClient := client.NewClient(defaultAPIURL)

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
	apiClient := client.NewClient(defaultAPIURL)

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
	apiClient := client.NewClient(defaultAPIURL)

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
