package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"indiekku/internal/api"
	"indiekku/internal/client"
	"indiekku/internal/docker"
	"indiekku/internal/history"
	"indiekku/internal/matchmaking"
	"indiekku/internal/security"
	"indiekku/internal/server"
	"indiekku/internal/state"

	"github.com/gin-gonic/gin"
)

var (
	version = "dev" // Set by ldflags during build
)

const (
	defaultPort    = "7777"
	defaultAPIPort = "3000"
	defaultGUIPort = "9090"
	defaultAPIURL  = "http://localhost:3000"
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
	case "dockerfiles":
		runDockerfiles()
	case "version":
		runVersion()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("indiekku - Container orchestration tool")
	fmt.Println("\nUsage:")
	fmt.Println("  indiekku serve [flags]     Start the API + matchmaking server (runs in background)")
	fmt.Println("    --public-ip <ip>         Public IP for matchmaking responses (auto-detected if omitted)")
	fmt.Println("    --match-port <port>      Matchmaking server port (default: 7070)")
	fmt.Println("    --token-secret <secret>  HMAC secret for join tokens (auto-generated if omitted)")
	fmt.Println("  indiekku shutdown          Stop the server")
	fmt.Println("  indiekku logs              View server logs")
	fmt.Println("  indiekku logs <server>     View logs for a specific container")
	fmt.Println("  indiekku start [port]      Start a container")
	fmt.Println("  indiekku stop <name>       Stop a container")
	fmt.Println("  indiekku ps                List running containers")
	fmt.Println("  indiekku dockerfiles       List available Dockerfile presets and active config")
	fmt.Println("  indiekku version           Show version information")
}

func runVersion() {
	fmt.Printf("indiekku %s\n", version)
}

func runDockerfiles() {
	fmt.Println("Dockerfile Configuration")
	fmt.Println("========================")
	fmt.Println()

	// Show available presets
	fmt.Println("Available Presets:")
	for _, name := range docker.ListPresets() {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Println()

	// Show current active Dockerfile
	activeName := docker.GetActiveDockerfileName()
	fmt.Printf("Active Dockerfile: %s\n", activeName)
	fmt.Println()

	// Show preview of active Dockerfile
	content, err := docker.GetActiveDockerfile()
	if err == nil {
		lines := strings.Split(content, "\n")
		fmt.Println("Preview (first 10 lines):")
		for i, line := range lines {
			if i >= 10 {
				fmt.Println("  ...")
				break
			}
			fmt.Printf("  %s\n", line)
		}
	}
	fmt.Println()
	fmt.Println("Use the web UI or API to change the active Dockerfile.")
}

// detectPublicIP asks ipify for the machine's public IP.
func detectPublicIP() (string, error) {
	c := &http.Client{Timeout: 5 * time.Second}
	resp, err := c.Get("https://api.ipify.org")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
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
		fmt.Printf("         http://localhost:3000/api/v1/servers\n\n")
		fmt.Println(strings.Repeat("=", 70))
		fmt.Println()
	}

	// Separate __daemon__ sentinel from real flags before parsing
	isDaemonProcess := false
	var filteredArgs []string
	for _, arg := range os.Args[2:] {
		if arg == "__daemon__" {
			isDaemonProcess = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}

	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	apiPort := fs.String("api-port", defaultAPIPort, "")
	publicIP := fs.String("public-ip", "", "")
	matchPort := fs.String("match-port", "7070", "")
	tokenSecret := fs.String("token-secret", "", "")
	if err := fs.Parse(filteredArgs); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// If this is NOT the daemon process, fork and exit
	if !isDaemonProcess {
		// Check if server is already running
		if _, err := os.Stat(pidFile); err == nil {
			fmt.Println("Error: server is already running")
			fmt.Println("Use 'indiekku shutdown' to stop it first")
			os.Exit(1)
		}

		execPath, err := os.Executable()
		if err != nil {
			fmt.Printf("Failed to get executable path: %v\n", err)
			os.Exit(1)
		}

		// Pass all flags through to the daemon
		args := []string{"serve", "__daemon__",
			"--api-port", *apiPort,
			"--match-port", *matchPort,
		}
		if *publicIP != "" {
			args = append(args, "--public-ip", *publicIP)
		}
		if *tokenSecret != "" {
			args = append(args, "--token-secret", *tokenSecret)
		}

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

		fmt.Printf("✓ indiekku started\n")
		fmt.Printf("  PID:       %d\n", cmd.Process.Pid)
		fmt.Printf("  API:       127.0.0.1:%s  (localhost only)\n", *apiPort)
		fmt.Printf("  Web UI:    https://0.0.0.0:%s  (HTTPS, self-signed cert)\n", defaultGUIPort)
		fmt.Printf("  Match:     0.0.0.0:%s     (matchmaking)\n", *matchPort)
		fmt.Printf("\nUse 'indiekku logs' to view logs\n")
		fmt.Printf("Use 'indiekku shutdown' to stop\n")
		return
	}

	// ---- Daemon process ----
	fmt.Println("Starting indiekku...")

	pid := os.Getpid()
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644); err != nil {
		fmt.Printf("Warning: Failed to write PID file: %v\n", err)
	}
	defer os.Remove(pidFile)

	// Resolve public IP
	resolvedIP := *publicIP
	if resolvedIP == "" {
		detected, err := detectPublicIP()
		if err != nil {
			fmt.Printf("Warning: could not auto-detect public IP: %v\n", err)
			fmt.Println("Use --public-ip to set it explicitly.")
		} else {
			resolvedIP = detected
			fmt.Printf("✓ Public IP detected: %s\n", resolvedIP)
		}
	} else {
		fmt.Printf("✓ Public IP: %s (from flag)\n", resolvedIP)
	}

	// Generate token secret if not provided
	secret := *tokenSecret
	if secret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			fmt.Printf("Failed to generate token secret: %v\n", err)
			os.Exit(1)
		}
		secret = hex.EncodeToString(b)
		fmt.Println("Warning: --token-secret not set, using random secret (join tokens invalid after restart)")
	}

	// Load API key for authentication
	loadedAPIKey, err := security.LoadAPIKey()
	if err != nil {
		fmt.Printf("Failed to load API key: %v\n", err)
		os.Exit(1)
	}

	// Initialize state handler
	stateHandler := state.NewStateHandler()

	// Initialize history manager
	historyManager, err := history.NewHistoryManager("indiekku.db")
	if err != nil {
		fmt.Printf("Warning: Failed to initialize history tracking: %v\n", err)
		fmt.Println("History tracking will be disabled, but the server will continue to run.")
		historyManager = nil
	} else {
		defer historyManager.Close()
		fmt.Println("✓ History tracking enabled (indiekku.db)")
	}

	// Create API handler
	apiHandler := api.NewAPIHandler(
		stateHandler,
		historyManager,
		server.DefaultServerDir,
		docker.DefaultImageName,
		loadedAPIKey,
	)

	// Store matchmaking config for the web UI
	tokenSecretStatus := "configured"
	if *tokenSecret == "" {
		tokenSecretStatus = "auto-generated"
	}
	apiHandler.SetMatchConfig(api.MatchConfig{
		PublicIP:          resolvedIP,
		MatchPort:         *matchPort,
		TokenSecretStatus: tokenSecretStatus,
	})

	// Start GUI server with self-signed TLS
	guiRouter := apiHandler.SetupGUIRouter("127.0.0.1:"+*apiPort, "127.0.0.1:"+*matchPort)
	go func() {
		cert, err := security.GenerateSelfSignedCert(resolvedIP)
		if err != nil {
			fmt.Printf("Warning: Failed to generate TLS cert, falling back to HTTP: %v\n", err)
			fmt.Printf("Web UI listening on 0.0.0.0:%s (HTTP)\n", defaultGUIPort)
			if err := guiRouter.Run(":" + defaultGUIPort); err != nil {
				fmt.Printf("Failed to start GUI server: %v\n", err)
				os.Exit(1)
			}
			return
		}
		tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
		ln, err := tls.Listen("tcp", ":"+defaultGUIPort, tlsCfg)
		if err != nil {
			fmt.Printf("Failed to start GUI TLS listener: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Web UI listening on 0.0.0.0:%s (HTTPS)\n", defaultGUIPort)
		srv := &http.Server{Handler: guiRouter}
		if err := srv.Serve(ln); err != nil {
			fmt.Printf("Failed to start GUI server: %v\n", err)
			os.Exit(1)
		}
	}()

	// Start matchmaking server
	indiekkuClient, err := client.NewClient("http://127.0.0.1:" + *apiPort)
	if err != nil {
		fmt.Printf("Failed to create internal client for matchmaking: %v\n", err)
		os.Exit(1)
	}
	matchHandler := matchmaking.NewHandler(indiekkuClient, resolvedIP, secret)

	gin.SetMode(gin.ReleaseMode)
	matchRouter := gin.New()
	matchRouter.Use(gin.Recovery())
	matchRouter.GET("/health", matchHandler.Health)
	matchRouter.GET("/servers", matchHandler.ListServers)
	matchRouter.POST("/match", matchHandler.Match)
	matchRouter.POST("/join/:name", matchHandler.Join)

	go func() {
		fmt.Printf("Matchmaking server listening on 0.0.0.0:%s\n", *matchPort)
		if err := matchRouter.Run(":" + *matchPort); err != nil {
			fmt.Printf("Failed to start matchmaking server: %v\n", err)
			os.Exit(1)
		}
	}()

	// Start API server on localhost only
	router := apiHandler.SetupRouter()
	fmt.Printf("API server listening on 127.0.0.1:%s\n", *apiPort)
	fmt.Printf("PID: %d (saved to %s)\n", pid, pidFile)
	if err := router.Run("127.0.0.1:" + *apiPort); err != nil {
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
		if err == nil && resp != nil && resp.Count > 0 {
			for _, server := range resp.Servers {
				fmt.Printf("  Stopping %s...\n", server.ContainerName)
				if err := apiClient.StopServer(server.ContainerName); err != nil {
					fmt.Printf("    Warning: Failed to stop %s: %v\n", server.ContainerName, err)
				} else {
					fmt.Printf("    ✓ Stopped %s\n", server.ContainerName)
				}
			}
		} else if err == nil && resp != nil && resp.Count == 0 {
			fmt.Println("  No running servers to stop")
		} else if err != nil {
			fmt.Printf("  Warning: Could not list servers: %v\n", err)
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
		// Process already dead - clean up the stale PID file
		os.Remove(pidFile)
		fmt.Printf("Process already stopped. Cleaned up stale PID file.\n")
		return
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
	// Check if a server name was provided
	if len(os.Args) > 2 {
		// View logs for a specific game server
		serverName := os.Args[2]
		runServerLogs(serverName)
		return
	}

	// View API server logs
	runAPILogs()
}

func runAPILogs() {
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

func runServerLogs(serverName string) {
	fmt.Printf("Fetching logs for %s...\n\n", serverName)

	// Get and stream Docker logs
	cmd, err := docker.GetContainerLogs(serverName, true, "100")
	if err != nil {
		fmt.Printf("Failed to get logs: %v\n", err)
		os.Exit(1)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("\nFailed to stream logs: %v\n", err)
		fmt.Println("\nTip: Use 'indiekku ps' to see available servers")
		os.Exit(1)
	}
}
