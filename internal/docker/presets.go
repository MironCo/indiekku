package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DockerfilesDir = "dockerfiles"

// Preset Dockerfiles
const unityDockerfile = `FROM --platform=linux/amd64 debian:13-slim

RUN apt-get update && apt-get install -y \
    libxss1 \
    libgtk-3-0 \
    libxrandr2 \
    libasound2 \
    libpangocairo-1.0-0 \
    libatk1.0-0 \
    libcairo-gobject2 \
    libgdk-pixbuf-xlib-2.0-0 \
    libnss3 \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -m -u 10001 appuser

# Copy files and fix ownership
COPY game_server/ /app/
RUN chown -R appuser:appuser /app && \
    find /app -type f \( -name "*.x86_64" -o -name "*.exe" \) -exec chmod +x {} \;

WORKDIR /app
USER appuser

EXPOSE 7777
`

const genericDockerfile = `FROM --platform=linux/amd64 debian:13-slim

# Create non-root user
RUN useradd -m -u 10001 appuser

# Copy files and fix ownership
COPY game_server/ /app/
RUN chown -R appuser:appuser /app && \
    find /app -type f -exec chmod +x {} \;

WORKDIR /app
USER appuser

EXPOSE 7777
`

// Presets maps preset names to their Dockerfile content
var Presets = map[string]string{
	"unity":   unityDockerfile,
	"generic": genericDockerfile,
}

// GetPreset returns the Dockerfile content for a preset name
func GetPreset(name string) (string, bool) {
	content, ok := Presets[name]
	return content, ok
}

// ListPresets returns all available preset names
func ListPresets() []string {
	names := make([]string, 0, len(Presets))
	for name := range Presets {
		names = append(names, name)
	}
	return names
}

// EnsureDockerfilesDir creates the dockerfiles directory and writes presets if missing
func EnsureDockerfilesDir() error {
	if err := os.MkdirAll(DockerfilesDir, 0755); err != nil {
		return fmt.Errorf("failed to create dockerfiles directory: %w", err)
	}

	// Write preset files if they don't exist
	for name, content := range Presets {
		path := filepath.Join(DockerfilesDir, name+".Dockerfile")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write preset %s: %w", name, err)
			}
		}
	}

	return nil
}

// GetActiveDockerfile returns the content of the currently active Dockerfile
// Falls back to unity preset if no active Dockerfile exists
func GetActiveDockerfile() (string, error) {
	activePath := filepath.Join(DockerfilesDir, "active.Dockerfile")

	content, err := os.ReadFile(activePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Fall back to unity preset
			return unityDockerfile, nil
		}
		return "", fmt.Errorf("failed to read active Dockerfile: %w", err)
	}

	return string(content), nil
}

// ValidateDockerfile checks if a Dockerfile has valid syntax
func ValidateDockerfile(content string) error {
	// Basic validation - check for required FROM instruction
	lines := strings.Split(content, "\n")
	hasFrom := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip comments and empty lines
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(strings.ToUpper(trimmed), "FROM ") {
			hasFrom = true
			break
		}
	}
	if !hasFrom {
		return fmt.Errorf("invalid Dockerfile: missing FROM instruction")
	}
	return nil
}

// SetActiveDockerfile saves the given content as the active Dockerfile
func SetActiveDockerfile(content string) error {
	// Validate before saving
	if err := ValidateDockerfile(content); err != nil {
		return err
	}

	if err := EnsureDockerfilesDir(); err != nil {
		return err
	}

	activePath := filepath.Join(DockerfilesDir, "active.Dockerfile")
	if err := os.WriteFile(activePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write active Dockerfile: %w", err)
	}

	return nil
}

// SetActiveFromPreset sets the active Dockerfile from a preset name
func SetActiveFromPreset(presetName string) error {
	content, ok := GetPreset(presetName)
	if !ok {
		return fmt.Errorf("unknown preset: %s", presetName)
	}
	return SetActiveDockerfile(content)
}

// SaveCustomDockerfile saves a custom Dockerfile with a given name
func SaveCustomDockerfile(name, content string) (string, error) {
	if err := EnsureDockerfilesDir(); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%s.Dockerfile", name)
	path := filepath.Join(DockerfilesDir, filename)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to save custom Dockerfile: %w", err)
	}

	return path, nil
}

// GetActiveDockerfileName returns the name/source of the active Dockerfile
func GetActiveDockerfileName() string {
	activePath := filepath.Join(DockerfilesDir, "active.Dockerfile")

	content, err := os.ReadFile(activePath)
	if err != nil {
		return "unity (default)"
	}

	// Check if it matches a preset
	contentStr := string(content)
	for name, preset := range Presets {
		if contentStr == preset {
			return name
		}
	}

	return "custom"
}
