package api

import (
	"fmt"
	"net/http"
	"time"

	"indiekku/internal/docker"
	"indiekku/internal/server"
	"indiekku/internal/state"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for the game server API
type ApiHandler struct {
	stateManager *state.StateHandler
	serverDir    string
	imageName    string
}

// NewHandler creates a new API handler
func NewAPIHandler(stateManager *state.StateHandler, serverDir, imageName string) *ApiHandler {
	return &ApiHandler{
		stateManager: stateManager,
		serverDir:    serverDir,
		imageName:    imageName,
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

	containerName := docker.DefaultContainerPrefix + port

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
	_, err := h.stateManager.GetServer(containerName)
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

// SetupRouter configures all API routes
func (h *ApiHandler) SetupRouter() *gin.Engine {
	r := gin.Default()

	// API routes
	api := r.Group("/api/v1")
	{
		api.POST("/servers/start", h.StartServer)
		api.DELETE("/servers/:name", h.StopServer)
		api.GET("/servers", h.ListServers)
		api.POST("/heartbeat", h.Heartbeat)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return r
}
