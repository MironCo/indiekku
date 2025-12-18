package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	key1, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}

	// Should be 64 hex characters (32 bytes)
	if len(key1) != 64 {
		t.Errorf("Expected key length 64, got %d", len(key1))
	}

	// Should only contain hex characters
	for _, c := range key1 {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("Key contains non-hex character: %c", c)
		}
	}

	// Generate another key - should be different (randomness check)
	key2, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("Failed to generate second API key: %v", err)
	}

	if key1 == key2 {
		t.Error("Generated keys should be different (randomness test)")
	}
}

func TestSaveAndLoadAPIKey(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Test save
	testKey := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	err = SaveAPIKey(testKey)
	if err != nil {
		t.Fatalf("Failed to save API key: %v", err)
	}

	// Check file exists
	if _, err := os.Stat(APIKeyFile); os.IsNotExist(err) {
		t.Fatal("API key file was not created")
	}

	// Check file permissions (should be 0600)
	info, err := os.Stat(APIKeyFile)
	if err != nil {
		t.Fatalf("Failed to stat API key file: %v", err)
	}
	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("Expected file permissions 0600, got %o", mode)
	}

	// Test load
	loadedKey, err := LoadAPIKey()
	if err != nil {
		t.Fatalf("Failed to load API key: %v", err)
	}

	if loadedKey != testKey {
		t.Errorf("Loaded key %s does not match saved key %s", loadedKey, testKey)
	}
}

func TestLoadAPIKey_FileDoesNotExist(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Try to load non-existent key
	_, err = LoadAPIKey()
	if err == nil {
		t.Error("Expected error when loading non-existent API key")
	}
}

func TestEnsureAPIKey(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// First call should generate new key
	key1, isNew1, err := EnsureAPIKey()
	if err != nil {
		t.Fatalf("Failed to ensure API key: %v", err)
	}

	if !isNew1 {
		t.Error("Expected isNew=true on first call")
	}

	if len(key1) != 64 {
		t.Errorf("Expected key length 64, got %d", len(key1))
	}

	// Second call should load existing key
	key2, isNew2, err := EnsureAPIKey()
	if err != nil {
		t.Fatalf("Failed to ensure API key on second call: %v", err)
	}

	if isNew2 {
		t.Error("Expected isNew=false on second call")
	}

	if key1 != key2 {
		t.Error("Second call should return same key as first call")
	}
}

func TestSaveAPIKey_InvalidDirectory(t *testing.T) {
	// Try to save to a directory that doesn't exist and can't be created
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Create temp directory with a file where we'd need a directory
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create a file with the same name as our key file
	if err := os.WriteFile(APIKeyFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create conflicting file: %v", err)
	}

	// Make it read-only
	if err := os.Chmod(APIKeyFile, 0444); err != nil {
		t.Fatalf("Failed to chmod file: %v", err)
	}

	// Try to save should fail
	err = SaveAPIKey("testkey")
	if err == nil {
		t.Error("Expected error when saving to read-only file")
	}
}

func TestLoadAPIKey_WithWhitespace(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Save key with whitespace
	testKey := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	keyWithWhitespace := "  " + testKey + "\n\t"

	err = os.WriteFile(APIKeyFile, []byte(keyWithWhitespace), 0600)
	if err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Load should trim whitespace
	loadedKey, err := LoadAPIKey()
	if err != nil {
		t.Fatalf("Failed to load API key: %v", err)
	}

	if loadedKey != testKey {
		t.Errorf("Expected trimmed key %s, got %s", testKey, loadedKey)
	}
}

func TestSaveAPIKey_CreatesDirectoryIfNeeded(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Change to subdirectory
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("Failed to change to subdirectory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Save should work even though we're in a different directory
	testKey := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	err = SaveAPIKey(testKey)
	if err != nil {
		t.Fatalf("Failed to save API key: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(APIKeyFile); os.IsNotExist(err) {
		t.Fatal("API key file was not created")
	}
}
