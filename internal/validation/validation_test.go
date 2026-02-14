package validation

import (
	"strings"
	"testing"
)

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		valid   bool
	}{
		{"empty port is valid", "", true},
		{"valid port", "8080", true},
		{"min port", "1", true},
		{"max port", "65535", true},
		{"port zero invalid", "0", false},
		{"port too high", "65536", false},
		{"negative port", "-1", false},
		{"non-numeric", "abc", false},
		{"float port", "80.5", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePort(tt.port)
			if result.Valid != tt.valid {
				t.Errorf("ValidatePort(%q) = %v, want %v", tt.port, result.Valid, tt.valid)
			}
		})
	}
}

func TestValidateContainerName(t *testing.T) {
	tests := []struct {
		name      string
		container string
		valid     bool
	}{
		{"valid name", "my-server", true},
		{"alphanumeric", "server1", true},
		{"with underscore", "server_1", true},
		{"starts with number", "1server", true},
		{"empty name", "", false},
		{"starts with hyphen", "-server", false},
		{"starts with underscore", "_server", false},
		{"has spaces", "my server", false},
		{"has dots", "my.server", false},
		{"special chars", "my@server", false},
		{"too long", strings.Repeat("a", 64), false},
		{"max length", strings.Repeat("a", 63), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateContainerName(tt.container)
			if result.Valid != tt.valid {
				t.Errorf("ValidateContainerName(%q) = %v, want %v", tt.container, result.Valid, tt.valid)
			}
		})
	}
}

func TestValidatePlayerCount(t *testing.T) {
	tests := []struct {
		name  string
		count int
		valid bool
	}{
		{"zero", 0, true},
		{"positive", 50, true},
		{"max allowed", MaxPlayerCount, true},
		{"negative", -1, false},
		{"too high", MaxPlayerCount + 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePlayerCount(tt.count)
			if result.Valid != tt.valid {
				t.Errorf("ValidatePlayerCount(%d) = %v, want %v", tt.count, result.Valid, tt.valid)
			}
		})
	}
}

func TestValidateFileSize(t *testing.T) {
	tests := []struct {
		name  string
		size  int64
		valid bool
	}{
		{"small file", 1024, true},
		{"max size", MaxUploadSize, true},
		{"too large", MaxUploadSize + 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateFileSize(tt.size)
			if result.Valid != tt.valid {
				t.Errorf("ValidateFileSize(%d) = %v, want %v", tt.size, result.Valid, tt.valid)
			}
		})
	}
}

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		valid   bool
	}{
		{"empty command", "", true},
		{"simple command", "server", true},
		{"with path", "/usr/bin/server", true},
		{"shell injection semicolon", "cmd; rm -rf /", false},
		{"shell injection pipe", "cmd | rm", false},
		{"shell injection ampersand", "cmd && rm", false},
		{"backtick", "cmd `whoami`", false},
		{"path traversal", "../../../etc/passwd", false},
		{"dollar sign", "cmd $HOME", false},
		{"too long", strings.Repeat("a", MaxCommandLen+1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateCommand(tt.command)
			if result.Valid != tt.valid {
				t.Errorf("ValidateCommand(%q) = %v, want %v", tt.command, result.Valid, tt.valid)
			}
		})
	}
}

func TestValidateArgs(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		valid bool
	}{
		{"empty args", []string{}, true},
		{"simple args", []string{"-port", "8080"}, true},
		{"with equals", []string{"--config=path"}, true},
		{"with semicolon", []string{"arg; rm"}, false},
		{"with pipe", []string{"arg | cat"}, false},
		{"with backtick", []string{"arg `whoami`"}, false},
		{"too many args", make([]string, MaxArgs+1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateArgs(tt.args)
			if result.Valid != tt.valid {
				t.Errorf("ValidateArgs(%v) = %v, want %v", tt.args, result.Valid, tt.valid)
			}
		})
	}
}

func TestValidateDockerfile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{"valid dockerfile", "FROM ubuntu:latest\nRUN apt-get update", true},
		{"from only", "FROM alpine", true},
		{"empty", "", false},
		{"whitespace only", "   \n\t  ", false},
		{"no FROM", "RUN apt-get update", false},
		{"too large", strings.Repeat("FROM ubuntu\n", MaxDockerfileSize), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateDockerfile(tt.content)
			if result.Valid != tt.valid {
				t.Errorf("ValidateDockerfile() = %v, want %v", result.Valid, tt.valid)
			}
		})
	}
}

func TestValidatePresetName(t *testing.T) {
	tests := []struct {
		name   string
		preset string
		valid  bool
	}{
		{"valid name", "unity-linux", true},
		{"with dots", "unity.linux", true},
		{"alphanumeric", "preset1", true},
		{"empty", "", false},
		{"starts with hyphen", "-preset", false},
		{"special chars", "preset@name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePresetName(tt.preset)
			if result.Valid != tt.valid {
				t.Errorf("ValidatePresetName(%q) = %v, want %v", tt.preset, result.Valid, tt.valid)
			}
		})
	}
}

func TestZipFileValidator(t *testing.T) {
	t.Run("validates file count", func(t *testing.T) {
		v := NewZipFileValidator()
		v.MaxFiles = 3

		// First 3 files should be OK
		for i := 0; i < 3; i++ {
			result := v.ValidateFileEntry(100, 50)
			if !result.Valid {
				t.Errorf("File %d should be valid", i+1)
			}
		}

		// 4th file should fail
		result := v.ValidateFileEntry(100, 50)
		if result.Valid {
			t.Error("4th file should fail validation")
		}
	})

	t.Run("validates compression ratio", func(t *testing.T) {
		v := NewZipFileValidator()
		v.MaxRatio = 10

		// High compression ratio should fail (zip bomb protection)
		result := v.ValidateFileEntry(10000, 10)
		if result.Valid {
			t.Error("High compression ratio should fail")
		}
	})
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal name", "file.txt", "file.txt"},
		{"with forward slash", "path/file.txt", "path_file.txt"},
		{"with backslash", "path\\file.txt", "path_file.txt"},
		{"with null byte", "file\x00name.txt", "filename.txt"},
		{"with control chars", "file\x01\x02.txt", "file.txt"},
		{"long name", strings.Repeat("a", 300), strings.Repeat("a", 255)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
