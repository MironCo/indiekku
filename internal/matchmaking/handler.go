package matchmaking

import (
	"fmt"
	"net/http"
	"time"

	"indiekku/internal/client"

	"github.com/gin-gonic/gin"
)

const joinTokenTTL = 60 * time.Second

// Handler handles matchmaking HTTP requests
type Handler struct {
	indiekku   *client.Client
	publicIP   string
	maxPlayers int
	secret     string
}

// NewHandler creates a new matchmaking handler
func NewHandler(indiekku *client.Client, publicIP string, maxPlayers int, secret string) *Handler {
	return &Handler{
		indiekku:   indiekku,
		publicIP:   publicIP,
		maxPlayers: maxPlayers,
		secret:     secret,
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

// findOpenServer returns the first running server with available player slots,
// or nil if all servers are full (or none are running).
func (h *Handler) findOpenServer() (*client.ServerInfo, error) {
	resp, err := h.indiekku.ListServers()
	if err != nil {
		return nil, err
	}

	for _, s := range resp.Servers {
		if s.PlayerCount < h.maxPlayers {
			return s, nil
		}
	}

	return nil, nil
}
