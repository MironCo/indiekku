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
// It looks for executable files, skipping known non-executable types
// It recursively searches subdirectories, skipping __MACOSX folders
func FindBinary(serverDir string) (string, error) {
	var candidates []string

	// Known non-executable extensions to skip
	skipExtensions := map[string]bool{
		".txt": true, ".md": true, ".json": true, ".yaml": true, ".yml": true,
		".xml": true, ".cfg": true, ".ini": true, ".log": true, ".env": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true,
		".wav": true, ".mp3": true, ".ogg": true, ".zip": true, ".tar": true,
		".gz": true, ".dll": true, ".so": true, ".dylib": true, ".pdb": true,
	}

	// Known non-executable files to skip
	skipFiles := map[string]bool{
		".gitkeep": true, ".gitignore": true, ".DS_Store": true,
		"README": true, "LICENSE": true, "Makefile": true,
	}

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

		name := info.Name()

		// Skip hidden files (except we want to check them for executability)
		if strings.HasPrefix(name, ".") {
			return nil
		}

		// Skip known non-executable files
		if skipFiles[name] {
			return nil
		}

		// Skip known non-executable extensions
		ext := strings.ToLower(filepath.Ext(name))
		if skipExtensions[ext] {
			return nil
		}

		// Check if file looks like an executable:
		// 1. Has execute permission bit set
		// 2. Has known executable extension (.x86_64, .exe)
		// 3. Contains "linux" or "server" in name (common patterns)
		isExec := info.Mode()&0111 != 0
		hasExecExt := strings.HasSuffix(name, ".x86_64") || strings.HasSuffix(name, ".exe")
		looksLikeBinary := strings.Contains(strings.ToLower(name), "linux") ||
			strings.Contains(strings.ToLower(name), "server") ||
			strings.Contains(strings.ToLower(name), "arm64") ||
			strings.Contains(strings.ToLower(name), "amd64")

		if isExec || hasExecExt || looksLikeBinary {
			relPath, err := filepath.Rel(serverDir, path)
			if err != nil {
				return err
			}
			candidates = append(candidates, filepath.Join("/app", relPath))
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("could not scan server directory: %w", err)
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no executable found in %s", serverDir)
	}

	// Return the first candidate (could be smarter about prioritization)
	return candidates[0], nil
}
