package state

import (
	"fmt"
	"sync"
	"time"
)

// ServerInfo represents a running game server container
type ServerInfo struct {
	ContainerID   string    `json:"container_id"`
	ContainerName string    `json:"container_name"`
	Port          string    `json:"port"`
	PlayerCount   int       `json:"player_count"`
	StartedAt     time.Time `json:"started_at"`
}

// StateHandler handles in-memory state for running game servers
type StateHandler struct {
	mu      sync.RWMutex
	servers map[string]*ServerInfo // key: container name
	ports   map[string]bool        // track used ports
}

// NewStateHandler creates a new state handler
func NewStateHandler() *StateHandler {
	return &StateHandler{
		servers: make(map[string]*ServerInfo),
		ports:   make(map[string]bool),
	}
}

// AddServer registers a new running server
func (h *StateHandler) AddServer(info *ServerInfo) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.servers[info.ContainerName] = info
	h.ports[info.Port] = true
}

// RemoveServer unregisters a server
func (h *StateHandler) RemoveServer(containerName string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if server, exists := h.servers[containerName]; exists {
		delete(h.ports, server.Port)
		delete(h.servers, containerName)
	}
}

// GetServer retrieves a server by container name
func (h *StateHandler) GetServer(containerName string) (*ServerInfo, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	server, exists := h.servers[containerName]
	if !exists {
		return nil, fmt.Errorf("server not found: %s", containerName)
	}
	return server, nil
}

// ListServers returns all running servers
func (h *StateHandler) ListServers() []*ServerInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	servers := make([]*ServerInfo, 0, len(h.servers))
	for _, server := range h.servers {
		servers = append(servers, server)
	}
	return servers
}

// UpdatePlayerCount updates the player count for a server
func (h *StateHandler) UpdatePlayerCount(containerName string, count int) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	server, exists := h.servers[containerName]
	if !exists {
		return fmt.Errorf("server not found: %s", containerName)
	}
	server.PlayerCount = count
	return nil
}

// IsPortInUse checks if a port is already in use
func (h *StateHandler) IsPortInUse(port string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.ports[port]
}

// GetNextAvailablePort finds the next available port starting from base port
func (h *StateHandler) GetNextAvailablePort(basePort int) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for port := basePort; port < basePort+1000; port++ {
		portStr := fmt.Sprintf("%d", port)
		if !h.ports[portStr] {
			return portStr
		}
	}
	return fmt.Sprintf("%d", basePort) // fallback
}
