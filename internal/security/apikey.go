package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// APIKeyLength is the length of the generated API key in bytes (32 bytes = 64 hex characters)
	APIKeyLength = 32
	// APIKeyFile is the filename where the API key is stored
	APIKeyFile = ".indiekku_apikey"
)

// GenerateAPIKey generates a cryptographically secure random API key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, APIKeyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// SaveAPIKey saves the API key to a file in the current directory
func SaveAPIKey(apiKey string) error {
	keyPath := filepath.Join(".", APIKeyFile)

	// Write with restricted permissions (owner read/write only)
	if err := os.WriteFile(keyPath, []byte(apiKey), 0600); err != nil {
		return fmt.Errorf("failed to write API key file: %w", err)
	}

	return nil
}

// LoadAPIKey loads the API key from the file
func LoadAPIKey() (string, error) {
	keyPath := filepath.Join(".", APIKeyFile)

	data, err := os.ReadFile(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("API key file not found")
		}
		return "", fmt.Errorf("failed to read API key file: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// APIKeyExists checks if an API key file already exists
func APIKeyExists() bool {
	keyPath := filepath.Join(".", APIKeyFile)
	_, err := os.Stat(keyPath)
	return err == nil
}

// EnsureAPIKey ensures an API key exists, generating one if necessary
// Returns the API key and a boolean indicating if it was newly generated
func EnsureAPIKey() (string, bool, error) {
	if APIKeyExists() {
		apiKey, err := LoadAPIKey()
		if err != nil {
			return "", false, err
		}
		return apiKey, false, nil
	}

	// Generate new API key
	apiKey, err := GenerateAPIKey()
	if err != nil {
		return "", false, err
	}

	// Save it
	if err := SaveAPIKey(apiKey); err != nil {
		return "", false, err
	}

	return apiKey, true, nil
}
