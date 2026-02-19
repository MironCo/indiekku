package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	"indiekku/internal/docker"
	"indiekku/internal/history"
	"indiekku/internal/namegen"
	"indiekku/internal/security"
	"indiekku/internal/server"
	"indiekku/internal/state"
	"indiekku/internal/validation"

	"github.com/gin-gonic/gin"
)

// MatchConfig holds the matchmaking configuration shown in the web UI.
type MatchConfig struct {
	PublicIP          string `json:"public_ip"`
	MatchPort         string `json:"match_port"`
	MaxPlayers        int    `json:"max_players"`
	TokenSecretStatus string `json:"token_secret_status"`
}

// Handler handles HTTP requests for the game server API
type ApiHandler struct {
	stateManager   *state.StateHandler
	historyManager *history.HistoryManager
	sessionStore   *security.SessionStore
	csrfManager    *security.CSRFManager
	serverDir      string
	imageName      string
	apiKey         string
	matchConfig    *MatchConfig
	pollMu         sync.Mutex
	pollCancels    map[string]context.CancelFunc
}

// NewHandler creates a new API handler
func NewAPIHandler(stateManager *state.StateHandler, historyManager *history.HistoryManager, serverDir, imageName, apiKey string) *ApiHandler {
	return &ApiHandler{
		stateManager:   stateManager,
		historyManager: historyManager,
		sessionStore:   security.NewSessionStore(apiKey),
		csrfManager:    security.NewCSRFManager(),
		serverDir:      serverDir,
		imageName:      imageName,
		apiKey:         apiKey,
		pollCancels:    make(map[string]context.CancelFunc),
	}
}

// SetMatchConfig stores the matchmaking config so the web UI can read it.
func (h *ApiHandler) SetMatchConfig(cfg MatchConfig) {
	h.matchConfig = &cfg
}

// GetMatchConfig handles GET /api/v1/matchmaking/config
func (h *ApiHandler) GetMatchConfig(c *gin.Context) {
	if h.matchConfig == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "matchmaking not configured"})
		return
	}
	c.JSON(http.StatusOK, h.matchConfig)
}

// GetCSRFToken generates and returns a new CSRF token
func (h *ApiHandler) GetCSRFToken(c *gin.Context) {
	token, err := h.csrfManager.GenerateToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate CSRF token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"csrf_token": token})
}

// StartServerRequest represents the request body for starting a server
type StartServerRequest struct {
	Port    string   `json:"port,omitempty"`
	Command string   `json:"command,omitempty"` // Custom command to run (overrides auto-detected binary)
	Args    []string `json:"args,omitempty"`    // Custom args (overrides default -port behavior)
}

// StartServerResponse represents the response for starting a server
type StartServerResponse struct {
	ContainerName string `json:"container_name"`
	Port          string `json:"port"`
	Message       string `json:"message"`
}

// HeartbeatRequest represents the request body for server heartbeats
type HeartbeatRequest struct {
	ContainerName string `json:"container_name"`
	PlayerCount   int    `json:"player_count"`
}

// StartServer handles POST /servers/start
func (h *ApiHandler) StartServer(c *gin.Context) {
	var req StartServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate port
	if result := validation.ValidatePort(req.Port); !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Message})
		return
	}

	// Validate command if provided
	if result := validation.ValidateCommand(req.Command); !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Message})
		return
	}

	// Validate args if provided
	if result := validation.ValidateArgs(req.Args); !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Message})
		return
	}

	// Determine port
	port := req.Port
	if port == "" {
		port = h.stateManager.GetNextAvailablePort(docker.GetDefaultPort())
	} else if h.stateManager.IsPortInUse(port) {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("Port %s is already in use", port),
		})
		return
	}

	// Determine command and args
	command := req.Command
	args := req.Args

	// If no custom command provided, try to find server binary (legacy behavior)
	if command == "" {
		serverBinary, err := server.FindBinary(h.serverDir)
		if err != nil {
			// No binary found and no command specified - that's okay if Dockerfile has CMD
			fmt.Printf("No server binary found, using Dockerfile CMD: %v\n", err)
		} else {
			command = serverBinary
			if len(args) == 0 {
				args = []string{"-port", port}
			}
		}
	}

	// Generate a unique container name with video game theme
	var containerName string
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		name, err := namegen.Generate()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to generate server name: %v", err),
			})
			return
		}
		containerName = name

		// Check if this name is already in use
		if _, err := h.stateManager.GetServer(containerName); err != nil {
			// Server not found, this name is available
			break
		}

		// Name collision, try again
		if i == maxRetries-1 {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate unique server name after multiple attempts",
			})
			return
		}
	}

	// Check if Docker image exists, if not build it
	if !docker.ImageExists(h.imageName) {
		if err := docker.BuildImage(h.imageName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to build Docker image: %v", err),
			})
			return
		}
	}

	// Run the container with config
	// ContainerPort is the internal port the app listens on (from config)
	// Port is the external port we expose
	containerPort := fmt.Sprintf("%d", docker.GetDefaultPort())
	cfg := docker.ContainerConfig{
		Name:          containerName,
		ImageName:     h.imageName,
		Port:          port,
		ContainerPort: containerPort,
		Command:       command,
		Args:          args,
	}
	if err := docker.RunContainer(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to start container: %v", err),
		})
		return
	}

	// Register server in state
	h.stateManager.AddServer(&state.ServerInfo{
		ContainerName: containerName,
		Port:          port,
		Command:       command,
		Args:          args,
		PlayerCount:   0,
		StartedAt:     time.Now(),
	})

	// Start polling the container's SDK status endpoint (if the Unity SDK is installed).
	// Errors are silently ignored so servers without the SDK still work.
	ctx, cancel := context.WithCancel(context.Background())
	h.pollMu.Lock()
	h.pollCancels[containerName] = cancel
	h.pollMu.Unlock()
	go docker.StartPolling(ctx, containerName, func(playerCount, maxPlayers int) {
		h.stateManager.UpdateServerStatus(containerName, playerCount, maxPlayers)
	})

	// Record server start in history
	if h.historyManager != nil {
		if err := h.historyManager.RecordServerStart(containerName, port); err != nil {
			fmt.Printf("Warning: Failed to record server start: %v\n", err)
		}
	}

	c.JSON(http.StatusCreated, StartServerResponse{
		ContainerName: containerName,
		Port:          port,
		Message:       "Container started successfully",
	})
}

// StopServer handles DELETE /servers/:name
func (h *ApiHandler) StopServer(c *gin.Context) {
	containerName := c.Param("name")

	// Validate container name
	if result := validation.ValidateContainerName(containerName); !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Message})
		return
	}

	// Check if server exists in state
	serverInfo, err := h.stateManager.GetServer(containerName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Server not found: %s", containerName),
		})
		return
	}

	// Stop the container
	if err := docker.StopContainer(containerName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to stop container: %v", err),
		})
		return
	}

	// Record server stop in history
	if h.historyManager != nil {
		if err := h.historyManager.RecordServerStop(containerName, serverInfo.Port, serverInfo.StartedAt); err != nil {
			fmt.Printf("Warning: Failed to record server stop: %v\n", err)
		}
	}

	// Cancel the SDK status poller for this container.
	h.pollMu.Lock()
	if cancel, ok := h.pollCancels[containerName]; ok {
		cancel()
		delete(h.pollCancels, containerName)
	}
	h.pollMu.Unlock()

	// Remove from state
	h.stateManager.RemoveServer(containerName)

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Server %s stopped successfully", containerName),
	})
}

// ListServers handles GET /servers
func (h *ApiHandler) ListServers(c *gin.Context) {
	servers := h.stateManager.ListServers()
	c.JSON(http.StatusOK, gin.H{
		"servers": servers,
		"count":   len(servers),
	})
}

// GetServer handles GET /servers/:name
func (h *ApiHandler) GetServer(c *gin.Context) {
	containerName := c.Param("name")

	// Validate container name
	if result := validation.ValidateContainerName(containerName); !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Message})
		return
	}

	server, err := h.stateManager.GetServer(containerName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Server not found: %s", containerName),
		})
		return
	}

	c.JSON(http.StatusOK, server)
}

// Heartbeat handles POST /heartbeat
func (h *ApiHandler) Heartbeat(c *gin.Context) {
	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate container name
	if result := validation.ValidateContainerName(req.ContainerName); !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Message})
		return
	}

	// Validate player count
	if result := validation.ValidatePlayerCount(req.PlayerCount); !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Message})
		return
	}

	if err := h.stateManager.UpdatePlayerCount(req.ContainerName, req.PlayerCount); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("Server not found: %s", req.ContainerName),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Heartbeat received",
	})
}

// GetServerHistory handles GET /history/servers
func (h *ApiHandler) GetServerHistory(c *gin.Context) {
	containerName := c.Query("container_name")
	limit := 100 // default limit

	// Validate container name if provided
	if containerName != "" {
		if result := validation.ValidateContainerName(containerName); !result.Valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": result.Message})
			return
		}
	}

	if h.historyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "History tracking not enabled",
		})
		return
	}

	events, err := h.historyManager.GetServerEvents(containerName, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to fetch server history: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"count":  len(events),
	})
}

// GetUploadHistory handles GET /history/uploads
func (h *ApiHandler) GetUploadHistory(c *gin.Context) {
	limit := 100 // default limit

	if h.historyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "History tracking not enabled",
		})
		return
	}

	uploads, err := h.historyManager.GetUploadHistory(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to fetch upload history: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"uploads": uploads,
		"count":   len(uploads),
	})
}

// GetServerLogs retrieves logs for a specific server
func (h *ApiHandler) GetServerLogs(c *gin.Context) {
	containerName := c.Param("name")
	if containerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Container name is required"})
		return
	}

	// Validate container name
	if result := validation.ValidateContainerName(containerName); !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Message})
		return
	}

	// Get logs from Docker (last 100 lines)
	logs, err := docker.GetContainerLogsSince(containerName, "5m")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get logs: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"container_name": containerName,
		"logs":           string(logs),
	})
}

// SetupGUIRouter creates a router that serves the web UI and proxies API calls
// to the API server running on apiAddr (e.g. "127.0.0.1:3000") and matchmaking
// calls to matchAddr (e.g. "127.0.0.1:7070").
// This router is intended to be bound to 0.0.0.0 so the dashboard is
// reachable externally while the API stays localhost-only.
func (h *ApiHandler) SetupGUIRouter(apiAddr, matchAddr string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(security.SecurityHeadersMiddleware())

	// Web UI pages
	r.GET("/", h.ServeWebUI)
	r.GET("/history", h.ServeHistoryUI)
	r.GET("/logs", h.ServeLogsUI)
	r.GET("/deploy", h.ServeDeployUI)
	r.GET("/match", h.ServeMatchUI)
	r.GET("/styles.css", h.ServeStyles)
	r.GET("/favicon.svg", h.ServeFavicon)

	// Proxy all API and health traffic to the localhost API server.
	apiTarget, _ := url.Parse("http://" + apiAddr)
	apiProxy := httputil.NewSingleHostReverseProxy(apiTarget)
	r.GET("/health", func(c *gin.Context) { apiProxy.ServeHTTP(c.Writer, c.Request) })
	r.Any("/api/v1/*path", func(c *gin.Context) { apiProxy.ServeHTTP(c.Writer, c.Request) })

	// Proxy /match-proxy/* to the matchmaking server, stripping the prefix.
	matchTarget, _ := url.Parse("http://" + matchAddr)
	matchProxy := httputil.NewSingleHostReverseProxy(matchTarget)
	r.Any("/match-proxy/*path", func(c *gin.Context) {
		c.Request.URL.Path = c.Param("path")
		matchProxy.ServeHTTP(c.Writer, c.Request)
	})

	return r
}

// SetupRouter configures all API routes
func (h *ApiHandler) SetupRouter() *gin.Engine {
	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)

	// Configure Gin to log to stdout (which gets redirected to indiekku.log by the daemon)
	gin.DefaultWriter = os.Stdout

	r := gin.New()
	r.Use(gin.Logger())   // Add logger middleware
	r.Use(gin.Recovery()) // Add recovery middleware

	// Apply security headers to all routes
	r.Use(security.SecurityHeadersMiddleware())

	// Health check (no auth required)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Web UI (no auth required - auth handled in the UI itself)
	r.GET("/", h.ServeWebUI)
	r.GET("/history", h.ServeHistoryUI)
	r.GET("/logs", h.ServeLogsUI)
	r.GET("/deploy", h.ServeDeployUI)
	r.GET("/styles.css", h.ServeStyles)
	r.GET("/favicon.svg", h.ServeFavicon)

	// API routes (auth required)
	api := r.Group("/api/v1")
	api.Use(security.AuthMiddleware(h.apiKey))
	{
		// CSRF token endpoint (GET, no CSRF check needed)
		api.GET("/csrf-token", h.GetCSRFToken)

		// Read-only endpoints (no CSRF protection needed)
		api.GET("/matchmaking/config", h.GetMatchConfig)
		api.GET("/servers", h.ListServers)
		api.GET("/servers/:name", h.GetServer)
		api.GET("/servers/:name/logs", h.GetServerLogs)
		api.GET("/history/servers", h.GetServerHistory)
		api.GET("/history/uploads", h.GetUploadHistory)
		api.GET("/dockerfiles/presets", h.ListDockerfilePresets)
		api.GET("/dockerfiles/active", h.GetActiveDockerfile)
		api.GET("/dockerfiles/history", h.GetDockerfileHistory)

		// State-changing endpoints (CSRF protection required)
		csrfProtected := api.Group("")
		csrfProtected.Use(security.CSRFMiddleware(h.csrfManager))
		{
			csrfProtected.POST("/servers/start", h.StartServer)
			csrfProtected.DELETE("/servers/:name", h.StopServer)
			csrfProtected.POST("/heartbeat", h.Heartbeat)
			csrfProtected.POST("/upload", h.UploadRelease)
			csrfProtected.POST("/dockerfiles/active", h.SetActiveDockerfile)
		}
	}

	return r
}
