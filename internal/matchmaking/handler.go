package matchmaking

import (
	"fmt"
	"net/http"
	"time"

	"indiekku/internal/client"

	"github.com/gin-gonic/gin"
)

const (
	joinTokenTTL      = 60 * time.Second
	defaultMaxPlayers = 4 // fallback when the Unity SDK hasn't reported max_players
)

// Handler handles matchmaking HTTP requests
type Handler struct {
	indiekku *client.Client
	publicIP string
	secret   string
}

// NewHandler creates a new matchmaking handler
func NewHandler(indiekku *client.Client, publicIP string, secret string) *Handler {
	return &Handler{
		indiekku: indiekku,
		publicIP: publicIP,
		secret:   secret,
	}
}

// Health handles GET /health
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// MatchResponse is returned to the game client
type MatchResponse struct {
	IP            string `json:"ip"`
	Port          string `json:"port"`
	ContainerName string `json:"container_name"`
	JoinToken     string `json:"join_token"`
}

// Match handles POST /match — finds an open server or starts a new one,
// then returns the address and a short-lived join token.
func (h *Handler) Match(c *gin.Context) {
	// Find a server with open slots
	server, err := h.findOpenServer()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to find server: %v", err)})
		return
	}

	// No open server found — start a new one
	if server == nil {
		resp, err := h.indiekku.StartServer("")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to start server: %v", err)})
			return
		}
		server = &client.ServerInfo{
			ContainerName: resp.ContainerName,
			Port:          resp.Port,
		}
	}

	// Issue a join token valid for 60 seconds
	token, err := GenerateJoinToken(h.secret, server.ContainerName, server.Port, joinTokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate join token"})
		return
	}

	c.JSON(http.StatusOK, MatchResponse{
		IP:            h.publicIP,
		Port:          server.Port,
		ContainerName: server.ContainerName,
		JoinToken:     token,
	})
}

// ServerListEntry is a server as seen by game clients
type ServerListEntry struct {
	ContainerName string `json:"container_name"`
	Port          string `json:"port"`
	PlayerCount   int    `json:"player_count"`
	MaxPlayers    int    `json:"max_players"`
	Full          bool   `json:"full"`
}

// ListServers handles GET /servers — returns all running servers and their occupancy.
func (h *Handler) ListServers(c *gin.Context) {
	resp, err := h.indiekku.ListServers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to list servers: %v", err)})
		return
	}

	entries := make([]ServerListEntry, 0, len(resp.Servers))
	for _, s := range resp.Servers {
		max := effectiveMax(s)
		entries = append(entries, ServerListEntry{
			ContainerName: s.ContainerName,
			Port:          s.Port,
			PlayerCount:   s.PlayerCount,
			MaxPlayers:    max,
			Full:          s.PlayerCount >= max,
		})
	}

	c.JSON(http.StatusOK, gin.H{"servers": entries, "count": len(entries)})
}

// Join handles POST /join/:name — manually joins a specific server if it has open
// slots, returning the same join token response as /match.
func (h *Handler) Join(c *gin.Context) {
	containerName := c.Param("name")

	resp, err := h.indiekku.ListServers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to list servers: %v", err)})
		return
	}

	var target *client.ServerInfo
	for _, s := range resp.Servers {
		if s.ContainerName == containerName {
			target = s
			break
		}
	}

	if target == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("server %q not found", containerName)})
		return
	}

	max := effectiveMax(target)
	if target.PlayerCount >= max {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("server %q is full (%d/%d)", containerName, target.PlayerCount, max)})
		return
	}

	token, err := GenerateJoinToken(h.secret, target.ContainerName, target.Port, joinTokenTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate join token"})
		return
	}

	c.JSON(http.StatusOK, MatchResponse{
		IP:            h.publicIP,
		Port:          target.Port,
		ContainerName: target.ContainerName,
		JoinToken:     token,
	})
}

// findOpenServer returns the first running server with available player slots,
// or nil if all servers are full (or none are running).
func (h *Handler) findOpenServer() (*client.ServerInfo, error) {
	resp, err := h.indiekku.ListServers()
	if err != nil {
		return nil, err
	}

	for _, s := range resp.Servers {
		if s.PlayerCount < effectiveMax(s) {
			return s, nil
		}
	}

	return nil, nil
}

// effectiveMax returns the server's reported max players, or defaultMaxPlayers
// if the Unity SDK hasn't reported one yet.
func effectiveMax(s *client.ServerInfo) int {
	if s.MaxPlayers > 0 {
		return s.MaxPlayers
	}
	return defaultMaxPlayers
}
