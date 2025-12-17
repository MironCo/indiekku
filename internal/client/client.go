package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultTimeout = 5 * time.Minute
)

// Client handles HTTP requests to the indiekku API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// HealthCheck checks if the API server is running
func (c *Client) HealthCheck() error {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("API server is not running: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API server returned status %d", resp.StatusCode)
	}

	return nil
}

// StartServerRequest represents the request to start a server
type StartServerRequest struct {
	Port string `json:"port,omitempty"`
}

// StartServerResponse represents the response from starting a server
type StartServerResponse struct {
	ContainerName string `json:"container_name"`
	Port          string `json:"port"`
	Message       string `json:"message"`
}

// StartServer starts a new game server
func (c *Client) StartServer(port string) (*StartServerResponse, error) {
	req := StartServerRequest{Port: port}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/v1/servers/start",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result StartServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// StopServer stops a running game server
func (c *Client) StopServer(containerName string) error {
	req, err := http.NewRequest(
		"DELETE",
		c.baseURL+"/api/v1/servers/"+containerName,
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// ServerInfo represents information about a running server
type ServerInfo struct {
	ContainerID   string    `json:"container_id"`
	ContainerName string    `json:"container_name"`
	Port          string    `json:"port"`
	PlayerCount   int       `json:"player_count"`
	StartedAt     time.Time `json:"started_at"`
}

// ListServersResponse represents the response from listing servers
type ListServersResponse struct {
	Servers []*ServerInfo `json:"servers"`
	Count   int           `json:"count"`
}

// ListServers lists all running game servers
func (c *Client) ListServers() (*ListServersResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/servers")
	if err != nil {
		return nil, fmt.Errorf("failed to list servers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result ListServersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
