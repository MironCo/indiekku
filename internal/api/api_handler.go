package api

import (
	"fmt"
	"net/http"
	"time"

	"indiekku/internal/docker"
	"indiekku/internal/history"
	"indiekku/internal/namegen"
	"indiekku/internal/security"
	"indiekku/internal/server"
	"indiekku/internal/state"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for the game server API
type ApiHandler struct {
	stateManager   *state.StateHandler
	historyManager *history.HistoryManager
	serverDir      string
	imageName      string
	apiKey         string
}

// NewHandler creates a new API handler
func NewAPIHandler(stateManager *state.StateHandler, historyManager *history.HistoryManager, serverDir, imageName, apiKey string) *ApiHandler {
	return &ApiHandler{
		stateManager:   stateManager,
		historyManager: historyManager,
		serverDir:      serverDir,
		imageName:      imageName,
		apiKey:         apiKey,
	}
}

// StartServerRequest represents the request body for starting a server
type StartServerRequest struct {
	Port string `json:"port,omitempty"`
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

	// Find server binary
	serverBinary, err := server.FindBinary(h.serverDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to find server binary: %v", err),
		})
		return
	}

	// Determine port
	port := req.Port
	if port == "" {
		port = h.stateManager.GetNextAvailablePort(7777)
	} else if h.stateManager.IsPortInUse(port) {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("Port %s is already in use", port),
		})
		return
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

	// Run the container
	if err := docker.RunContainer(containerName, h.imageName, port, serverBinary); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to start container: %v", err),
		})
		return
	}

	// Register server in state
	h.stateManager.AddServer(&state.ServerInfo{
		ContainerName: containerName,
		Port:          port,
		PlayerCount:   0,
		StartedAt:     time.Now(),
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
		Message:       "Game server started successfully",
	})
}

// StopServer handles DELETE /servers/:name
func (h *ApiHandler) StopServer(c *gin.Context) {
	containerName := c.Param("name")

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

// Heartbeat handles POST /heartbeat
func (h *ApiHandler) Heartbeat(c *gin.Context) {
	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
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

// SetupRouter configures all API routes
func (h *ApiHandler) SetupRouter() *gin.Engine {
	r := gin.Default()

	// Apply security headers to all routes
	r.Use(security.SecurityHeadersMiddleware())

	// Health check (no auth required)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Web UI (no auth required - auth handled in the UI itself)
	r.GET("/", h.ServeWebUI)
	r.GET("/history", h.ServeHistoryUI)

	// API routes (auth required)
	api := r.Group("/api/v1")
	api.Use(security.AuthMiddleware(h.apiKey))
	{
		api.POST("/servers/start", h.StartServer)
		api.DELETE("/servers/:name", h.StopServer)
		api.GET("/servers", h.ListServers)
		api.POST("/heartbeat", h.Heartbeat)
		api.POST("/upload", h.UploadRelease)
		api.GET("/history/servers", h.GetServerHistory)
		api.GET("/history/uploads", h.GetUploadHistory)
	}

	return r
}
