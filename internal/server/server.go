package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultServerDir = "game_server"
)

// FindBinary scans the server directory for game server executables
// It looks for common patterns like .x86_64 (Linux) and .exe (Windows)
func FindBinary(serverDir string) (string, error) {
	entries, err := os.ReadDir(serverDir)
	if err != nil {
		return "", fmt.Errorf("could not read server directory: %w", err)
	}

	// Look for common game server binary patterns
	patterns := []string{".x86_64", ".exe"}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		for _, pattern := range patterns {
			if strings.HasSuffix(name, pattern) {
				// Return the path as it will be in the Docker container
				return filepath.Join("/app", name), nil
			}
		}
	}
	return "", fmt.Errorf("no game server binary found in %s: (looking for *.x86_64 or *.exe)", serverDir)
}
