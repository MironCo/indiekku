package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

const (
	// StatusPort is the internal port the indiekku Unity SDK listens on.
	// This port is never exposed externally; indiekku reaches it via the
	// container's bridge IP.
	StatusPort = 9999

	statusPollInterval = 30 * time.Second
	statusPollTimeout  = 5 * time.Second
)

// containerStatus is the JSON shape returned by the Unity SDK's /status endpoint.
type containerStatus struct {
	PlayerCount int `json:"player_count"`
	MaxPlayers  int `json:"max_players"`
}

// GetContainerIP returns the bridge network IP of a running container.
func GetContainerIP(containerName string) (string, error) {
	cmd := exec.Command("docker", "inspect", "-f", "{{.NetworkSettings.IPAddress}}", containerName)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker inspect failed: %w", err)
	}
	ip := strings.TrimSpace(string(out))
	if ip == "" {
		return "", fmt.Errorf("container %s has no bridge IP (may be using a custom network)", containerName)
	}
	return ip, nil
}

// PollContainerStatus fetches the current status from the Unity SDK endpoint.
// Returns playerCount, maxPlayers, and an error.
func PollContainerStatus(containerName string) (playerCount, maxPlayers int, err error) {
	ip, err := GetContainerIP(containerName)
	if err != nil {
		return 0, 0, err
	}

	url := fmt.Sprintf("http://%s:%d/status", ip, StatusPort)
	client := &http.Client{Timeout: statusPollTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return 0, 0, fmt.Errorf("status poll failed: %w", err)
	}
	defer resp.Body.Close()

	var s containerStatus
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return 0, 0, fmt.Errorf("failed to decode status response: %w", err)
	}
	return s.PlayerCount, s.MaxPlayers, nil
}

// StartPolling polls the container's SDK status endpoint on a fixed interval,
// calling updateFn with each result. It stops when ctx is cancelled.
// Errors (e.g. SDK not present) are silently skipped so non-SDK servers still work.
func StartPolling(ctx context.Context, containerName string, updateFn func(playerCount, maxPlayers int)) {
	ticker := time.NewTicker(statusPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pc, mp, err := PollContainerStatus(containerName)
			if err != nil {
				// SDK not installed or server not ready yet — skip silently.
				continue
			}
			updateFn(pc, mp)
		}
	}
}
