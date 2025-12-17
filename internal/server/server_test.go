package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindBinary(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		files       []string
		wantErr     bool
		wantBinary  string
	}{
		{
			name:       "finds .x86_64 binary",
			files:      []string{"test-server.x86_64"},
			wantErr:    false,
			wantBinary: "/app/test-server.x86_64",
		},
		{
			name:       "finds .exe binary",
			files:      []string{"test-server.exe"},
			wantErr:    false,
			wantBinary: "/app/test-server.exe",
		},
		{
			name:    "no binary found",
			files:   []string{"readme.txt", "config.json"},
			wantErr: true,
		},
		{
			name:       "multiple binaries, returns first",
			files:      []string{"server1.x86_64", "server2.x86_64"},
			wantErr:    false,
			wantBinary: "/app/server1.x86_64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test files
			for _, file := range tt.files {
				f, err := os.Create(filepath.Join(tempDir, file))
				if err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				f.Close()
			}

			// Test FindBinary
			binary, err := FindBinary(tempDir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if binary != tt.wantBinary {
					t.Errorf("got binary %q, want %q", binary, tt.wantBinary)
				}
			}

			// Cleanup for next test
			for _, file := range tt.files {
				os.Remove(filepath.Join(tempDir, file))
			}
		})
	}
}

func TestFindBinary_DirectoryDoesNotExist(t *testing.T) {
	_, err := FindBinary("/nonexistent/directory")
	if err == nil {
		t.Error("expected error for nonexistent directory, got nil")
	}
}