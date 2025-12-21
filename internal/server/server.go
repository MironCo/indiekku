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
// It recursively searches subdirectories, skipping __MACOSX folders
func FindBinary(serverDir string) (string, error) {
	// Look for common game server binary patterns
	patterns := []string{".x86_64", ".exe"}

	var foundPath string

	// Walk through the directory tree
	err := filepath.Walk(serverDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip __MACOSX directories (created by macOS zip)
		if info.IsDir() && strings.Contains(path, "__MACOSX") {
			return filepath.SkipDir
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file matches any pattern
		name := info.Name()
		for _, pattern := range patterns {
			if strings.HasSuffix(name, pattern) {
				// Get relative path from serverDir
				relPath, err := filepath.Rel(serverDir, path)
				if err != nil {
					return err
				}
				// Return the path as it will be in the Docker container
				foundPath = filepath.Join("/app", relPath)
				return filepath.SkipAll // Stop searching once found
			}
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return "", fmt.Errorf("could not scan server directory: %w", err)
	}

	if foundPath == "" {
		return "", fmt.Errorf("no game server binary found in %s: (looking for *.x86_64 or *.exe)", serverDir)
	}

	return foundPath, nil
}
