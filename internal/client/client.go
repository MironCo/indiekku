package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"indiekku/internal/security"
)

const (
	defaultTimeout = 5 * time.Minute
)

// Client handles HTTP requests to the indiekku API
type Client struct {
	baseURL    string
	apiKey     string
	csrfToken  string
	httpClient *http.Client
}

// NewClient creates a new API client
// It automatically loads the API key from the standard location
func NewClient(baseURL string) (*Client, error) {
	apiKey, err := security.LoadAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to load API key: %w (make sure 'indiekku serve' has been run)", err)
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}, nil
}

// addAuthHeader adds the Bearer token to the request
func (c *Client) addAuthHeader(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
}

// fetchCSRFToken fetches a CSRF token from the server for state-changing requests
func (c *Client) fetchCSRFToken() error {
	req, err := http.NewRequest("GET", c.baseURL+"/api/v1/csrf-token", nil)
	if err != nil {
		return err
	}
	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch CSRF token: status %d", resp.StatusCode)
	}

	var result struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	c.csrfToken = result.CSRFToken
	return nil
}

// addCSRFHeader adds the CSRF token header (fetches token if needed)
func (c *Client) addCSRFHeader(req *http.Request) error {
	if c.csrfToken == "" {
		if err := c.fetchCSRFToken(); err != nil {
			return err
		}
	}
	req.Header.Set("X-CSRF-Token", c.csrfToken)
	return nil
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
	reqBody := StartServerRequest{Port: port}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		"POST",
		c.baseURL+"/api/v1/servers/start",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	c.addAuthHeader(req)
	if err := c.addCSRFHeader(req); err != nil {
		return nil, fmt.Errorf("failed to get CSRF token: %w", err)
	}

	resp, err := c.httpClient.Do(req)
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

	c.addAuthHeader(req)
	if err := c.addCSRFHeader(req); err != nil {
		return fmt.Errorf("failed to get CSRF token: %w", err)
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
	req, err := http.NewRequest("GET", c.baseURL+"/api/v1/servers", nil)
	if err != nil {
		return nil, err
	}

	c.addAuthHeader(req)

	resp, err := c.httpClient.Do(req)
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
