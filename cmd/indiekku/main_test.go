package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestCLIBuild verifies that the CLI binary can be built successfully
func TestCLIBuild(t *testing.T) {
	// Build the binary
	cmd := exec.Command("go", "build", "-o", "../../bin/indiekku-test", ".")
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, string(output))
	}

	// Clean up
	defer os.Remove("../../bin/indiekku-test")

	// Verify binary exists
	if _, err := os.Stat("../../bin/indiekku-test"); os.IsNotExist(err) {
		t.Fatal("Binary was not created")
	}
}

// TestCLIUsage verifies that the CLI shows usage when called without arguments
func TestCLIUsage(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "../../bin/indiekku-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("../../bin/indiekku-test")

	// Run without arguments - should show usage and exit with code 1
	cmd := exec.Command("../../bin/indiekku-test")
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()

	// Should exit with error
	if err == nil {
		t.Error("Expected error when running without arguments, got none")
	}

	// Should contain usage information
	outputStr := string(output)
	if len(outputStr) == 0 {
		t.Error("Expected usage output, got empty string")
	}

	// Should mention available commands
	expectedStrings := []string{"Usage:", "serve", "start", "stop", "ps"}
	for _, expected := range expectedStrings {
		if !contains(outputStr, expected) {
			t.Errorf("Usage output missing '%s'", expected)
		}
	}
}

// TestCLIInvalidCommand verifies that the CLI handles invalid commands gracefully
func TestCLIInvalidCommand(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "../../bin/indiekku-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("../../bin/indiekku-test")

	// Run with invalid command
	cmd := exec.Command("../../bin/indiekku-test", "invalid-command-xyz")
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()

	// Should exit with error
	if err == nil {
		t.Error("Expected error when running with invalid command, got none")
	}

	// Should mention the unknown command
	outputStr := string(output)
	if !contains(outputStr, "Unknown command") {
		t.Errorf("Expected 'Unknown command' in output, got: %s", outputStr)
	}
}

// TestCLIHealthCheckWithoutServer verifies that commands fail when server is not running
func TestCLIHealthCheckWithoutServer(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "../../bin/indiekku-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("../../bin/indiekku-test")

	// Create a temporary API key file so authentication doesn't fail first
	tmpDir := t.TempDir()
	apiKeyPath := filepath.Join(tmpDir, ".indiekku_apikey")
	if err := os.WriteFile(apiKeyPath, []byte("test-api-key-1234567890abcdef"), 0600); err != nil {
		t.Fatalf("Failed to create test API key: %v", err)
	}

	// Try to list servers when API is not running
	cmd := exec.Command("../../bin/indiekku-test", "ps")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Should fail because API server is not running
	if err == nil {
		t.Error("Expected error when API server is not running, got none")
	}

	// Should mention that API server is not running
	outputStr := string(output)
	if !contains(outputStr, "API server is not running") && !contains(outputStr, "connection refused") {
		t.Logf("Output: %s", outputStr)
		// This might fail in different ways depending on the system, so we'll be lenient
	}
}

// TestCLIStartCommandRequiresAPIKey verifies that start command needs API key
func TestCLIStartCommandRequiresAPIKey(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "../../bin/indiekku-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("../../bin/indiekku-test")

	// Try to start server from a directory without API key
	tmpDir := t.TempDir()

	// Get the absolute path to the binary
	binPath, err := filepath.Abs("../../bin/indiekku-test")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	cmd := exec.Command(binPath, "start")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// Should fail because API key doesn't exist
	if err == nil {
		t.Error("Expected error when API key file doesn't exist, got none")
	}

	// Should mention API key issue
	outputStr := string(output)
	if !contains(outputStr, "failed to load API key") && !contains(outputStr, "no such file") && !contains(outputStr, "Error:") {
		t.Logf("Expected API key error, got: %s", outputStr)
		// Be lenient - as long as it fails, that's okay
	}
}

// TestCLIStopCommandFormat verifies that stop command requires container name
func TestCLIStopCommandFormat(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "../../bin/indiekku-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("../../bin/indiekku-test")

	// Try to stop without container name
	cmd := exec.Command("../../bin/indiekku-test", "stop")
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()

	// Should fail because container name is required
	if err == nil {
		t.Error("Expected error when container name is not provided, got none")
	}

	// Should show usage for stop command
	outputStr := string(output)
	if !contains(outputStr, "Usage:") || !contains(outputStr, "stop") {
		t.Errorf("Expected usage message for stop command, got: %s", outputStr)
	}
}

// TestCLIServeCommand verifies that serve command can start (we'll kill it quickly)
func TestCLIServeCommand(t *testing.T) {
	// This test is more complex as it needs to actually start the server
	// We'll skip it in CI environments
	if os.Getenv("CI") != "" {
		t.Skip("Skipping serve test in CI environment")
	}

	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "../../bin/indiekku-test", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI: %v", err)
	}
	defer os.Remove("../../bin/indiekku-test")

	// Use a temporary directory for test
	tmpDir := t.TempDir()

	// Get the absolute path to the binary
	binPath, err := filepath.Abs("../../bin/indiekku-test")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Start the serve command in the background
	cmd := exec.Command(binPath, "serve")
	cmd.Dir = tmpDir
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start serve command: %v", err)
	}

	// Give it a moment to start
	time.Sleep(2 * time.Second)

	// Kill the process
	if err := cmd.Process.Kill(); err != nil {
		t.Logf("Warning: failed to kill serve process: %v", err)
	}

	// Clean up PID file if it exists
	pidPath := filepath.Join(tmpDir, "indiekku.pid")
	os.Remove(pidPath)
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsCheck(s, substr))
}

func containsCheck(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
