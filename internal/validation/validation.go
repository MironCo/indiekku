package validation

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Common validation errors
var (
	ErrInvalidPort           = errors.New("invalid port number")
	ErrPortOutOfRange        = errors.New("port must be between 1 and 65535")
	ErrInvalidContainerName  = errors.New("invalid container name")
	ErrContainerNameTooLong  = errors.New("container name must be 63 characters or less")
	ErrInvalidPlayerCount    = errors.New("player count must be non-negative")
	ErrPlayerCountTooHigh    = errors.New("player count exceeds maximum allowed")
	ErrFileTooLarge          = errors.New("file exceeds maximum allowed size")
	ErrTooManyFiles          = errors.New("archive contains too many files")
	ErrFileSizeMismatch      = errors.New("extracted file size exceeds compressed size ratio")
	ErrInvalidCommand        = errors.New("command contains invalid characters")
	ErrInvalidArgument       = errors.New("argument contains invalid characters")
	ErrEmptyDockerfile       = errors.New("dockerfile content cannot be empty")
	ErrDockerfileTooLarge    = errors.New("dockerfile exceeds maximum allowed size")
	ErrInvalidPresetName     = errors.New("invalid preset name")
)

// Limits for validation
const (
	MaxPort              = 65535
	MinPort              = 1
	MaxContainerNameLen  = 63
	MaxPlayerCount       = 10000 // Reasonable max for a game server
	MaxUploadSize        = 500 * 1024 * 1024 // 500 MB
	MaxFilesInArchive    = 10000
	MaxCompressionRatio  = 100 // Max ratio of uncompressed to compressed size
	MaxDockerfileSize    = 1024 * 1024 // 1 MB
	MaxCommandLen        = 256
	MaxArgLen            = 1024
	MaxArgs              = 50
)

// containerNameRegex validates container names (Docker naming convention)
// Must start with alphanumeric, can contain alphanumeric, hyphens, and underscores
var containerNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// presetNameRegex validates preset names (alphanumeric, hyphens, underscores, dots)
var presetNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// dangerousCharsRegex matches shell metacharacters and control sequences
var dangerousCharsRegex = regexp.MustCompile(`[;&|$` + "`" + `\\<>(){}\[\]!#*?~]`)

// ValidationResult contains the result of a validation
type ValidationResult struct {
	Valid   bool
	Error   error
	Message string
}

// OK returns a successful validation result
func OK() ValidationResult {
	return ValidationResult{Valid: true}
}

// Fail returns a failed validation result
func Fail(err error, message string) ValidationResult {
	return ValidationResult{Valid: false, Error: err, Message: message}
}

// ValidatePort validates a port string
func ValidatePort(port string) ValidationResult {
	if port == "" {
		return OK() // Empty port is valid (auto-assign)
	}

	portNum, err := strconv.Atoi(port)
	if err != nil {
		return Fail(ErrInvalidPort, fmt.Sprintf("port must be a number, got: %s", port))
	}

	if portNum < MinPort || portNum > MaxPort {
		return Fail(ErrPortOutOfRange, fmt.Sprintf("port must be between %d and %d, got: %d", MinPort, MaxPort, portNum))
	}

	return OK()
}

// ValidateContainerName validates a container name
func ValidateContainerName(name string) ValidationResult {
	if name == "" {
		return Fail(ErrInvalidContainerName, "container name cannot be empty")
	}

	if len(name) > MaxContainerNameLen {
		return Fail(ErrContainerNameTooLong, fmt.Sprintf("container name must be %d characters or less, got: %d", MaxContainerNameLen, len(name)))
	}

	if !containerNameRegex.MatchString(name) {
		return Fail(ErrInvalidContainerName, "container name must start with alphanumeric and contain only alphanumeric, hyphens, or underscores")
	}

	return OK()
}

// ValidatePlayerCount validates a player count
func ValidatePlayerCount(count int) ValidationResult {
	if count < 0 {
		return Fail(ErrInvalidPlayerCount, fmt.Sprintf("player count cannot be negative, got: %d", count))
	}

	if count > MaxPlayerCount {
		return Fail(ErrPlayerCountTooHigh, fmt.Sprintf("player count cannot exceed %d, got: %d", MaxPlayerCount, count))
	}

	return OK()
}

// ValidateFileSize validates an upload file size
func ValidateFileSize(size int64) ValidationResult {
	if size > MaxUploadSize {
		return Fail(ErrFileTooLarge, fmt.Sprintf("file size %d bytes exceeds maximum %d bytes", size, MaxUploadSize))
	}

	return OK()
}

// ValidateCommand validates a command string for potential injection
func ValidateCommand(cmd string) ValidationResult {
	if cmd == "" {
		return OK() // Empty command is valid (use default)
	}

	if len(cmd) > MaxCommandLen {
		return Fail(ErrInvalidCommand, fmt.Sprintf("command length %d exceeds maximum %d", len(cmd), MaxCommandLen))
	}

	if dangerousCharsRegex.MatchString(cmd) {
		return Fail(ErrInvalidCommand, "command contains potentially dangerous characters")
	}

	// Check for path traversal attempts
	if strings.Contains(cmd, "..") {
		return Fail(ErrInvalidCommand, "command cannot contain path traversal sequences")
	}

	return OK()
}

// ValidateArgs validates command arguments
func ValidateArgs(args []string) ValidationResult {
	if len(args) == 0 {
		return OK()
	}

	if len(args) > MaxArgs {
		return Fail(ErrInvalidArgument, fmt.Sprintf("too many arguments: %d, maximum: %d", len(args), MaxArgs))
	}

	for i, arg := range args {
		if len(arg) > MaxArgLen {
			return Fail(ErrInvalidArgument, fmt.Sprintf("argument %d length %d exceeds maximum %d", i, len(arg), MaxArgLen))
		}

		// Allow some shell-like chars in args but block the most dangerous ones
		if strings.ContainsAny(arg, ";&|`\\") {
			return Fail(ErrInvalidArgument, fmt.Sprintf("argument %d contains potentially dangerous characters", i))
		}
	}

	return OK()
}

// ValidateDockerfile validates dockerfile content
func ValidateDockerfile(content string) ValidationResult {
	content = strings.TrimSpace(content)

	if content == "" {
		return Fail(ErrEmptyDockerfile, "dockerfile content cannot be empty")
	}

	if len(content) > MaxDockerfileSize {
		return Fail(ErrDockerfileTooLarge, fmt.Sprintf("dockerfile size %d exceeds maximum %d", len(content), MaxDockerfileSize))
	}

	// Basic validation: must contain FROM instruction
	lines := strings.Split(strings.ToUpper(content), "\n")
	hasFrom := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "FROM ") || line == "FROM" {
			hasFrom = true
			break
		}
	}

	if !hasFrom {
		return Fail(errors.New("invalid dockerfile"), "dockerfile must contain a FROM instruction")
	}

	return OK()
}

// ValidatePresetName validates a preset name
func ValidatePresetName(name string) ValidationResult {
	if name == "" {
		return Fail(ErrInvalidPresetName, "preset name cannot be empty")
	}

	if len(name) > 64 {
		return Fail(ErrInvalidPresetName, "preset name must be 64 characters or less")
	}

	if !presetNameRegex.MatchString(name) {
		return Fail(ErrInvalidPresetName, "preset name must start with alphanumeric and contain only alphanumeric, hyphens, underscores, or dots")
	}

	return OK()
}

// ZipFileValidator provides validation for ZIP archive extraction
type ZipFileValidator struct {
	TotalFiles       int
	TotalSize        int64
	CompressedSize   int64
	MaxFiles         int
	MaxTotalSize     int64
	MaxRatio         int64
}

// NewZipFileValidator creates a new validator with default limits
func NewZipFileValidator() *ZipFileValidator {
	return &ZipFileValidator{
		MaxFiles:     MaxFilesInArchive,
		MaxTotalSize: MaxUploadSize * MaxCompressionRatio,
		MaxRatio:     MaxCompressionRatio,
	}
}

// ValidateFileEntry validates a single file entry from a ZIP
func (v *ZipFileValidator) ValidateFileEntry(uncompressedSize, compressedSize uint64) ValidationResult {
	v.TotalFiles++
	v.TotalSize += int64(uncompressedSize)
	v.CompressedSize += int64(compressedSize)

	if v.TotalFiles > v.MaxFiles {
		return Fail(ErrTooManyFiles, fmt.Sprintf("archive contains more than %d files", v.MaxFiles))
	}

	if v.TotalSize > v.MaxTotalSize {
		return Fail(ErrFileTooLarge, fmt.Sprintf("total extracted size %d exceeds maximum %d", v.TotalSize, v.MaxTotalSize))
	}

	// Check compression ratio for potential zip bombs
	if v.CompressedSize > 0 {
		ratio := v.TotalSize / v.CompressedSize
		if ratio > v.MaxRatio {
			return Fail(ErrFileSizeMismatch, fmt.Sprintf("compression ratio %d exceeds maximum %d (potential zip bomb)", ratio, v.MaxRatio))
		}
	}

	return OK()
}

// SanitizeFilename removes or replaces unsafe characters from a filename
func SanitizeFilename(name string) string {
	// Remove path separators
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")

	// Remove null bytes and other control characters
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)

	// Truncate if too long
	if len(name) > 255 {
		name = name[:255]
	}

	return name
}