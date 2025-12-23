package api

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"indiekku/internal/docker"
	"indiekku/internal/server"

	"github.com/gin-gonic/gin"
)

// UploadRelease handles the upload of a new server build
func (h *ApiHandler) UploadRelease(c *gin.Context) {
	// Get the uploaded file
	file, err := c.FormFile("server_build")
	if err != nil {
		fmt.Printf("Error getting form file: %v\n", err)
		// Record failed upload
		if h.historyManager != nil {
			h.historyManager.RecordUpload("", 0, false, fmt.Sprintf("No file uploaded: %v", err))
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("No file uploaded: %v", err),
		})
		return
	}

	fmt.Printf("Received file: %s (%d bytes)\n", file.Filename, file.Size)

	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".zip") {
		fmt.Printf("Invalid file extension: %s\n", file.Filename)
		// Record failed upload
		if h.historyManager != nil {
			h.historyManager.RecordUpload(file.Filename, file.Size, false, "File must be a ZIP archive")
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File must be a ZIP archive",
		})
		return
	}

	// Create temporary file to save upload
	tempFile, err := os.CreateTemp("", "indiekku-upload-*.zip")
	if err != nil {
		fmt.Printf("Error creating temp file: %v\n", err)
		// Record failed upload
		if h.historyManager != nil {
			h.historyManager.RecordUpload(file.Filename, file.Size, false, fmt.Sprintf("Failed to create temp file: %v", err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create temp file: %v", err),
		})
		return
	}
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)

	fmt.Printf("Saving to temp file: %s\n", tempFilePath)

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		tempFile.Close()
		fmt.Printf("Error opening uploaded file: %v\n", err)
		// Record failed upload
		if h.historyManager != nil {
			h.historyManager.RecordUpload(file.Filename, file.Size, false, fmt.Sprintf("Failed to open uploaded file: %v", err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to open uploaded file: %v", err),
		})
		return
	}
	defer src.Close()

	// Copy the uploaded file to temp file
	if _, err := io.Copy(tempFile, src); err != nil {
		tempFile.Close()
		fmt.Printf("Error copying uploaded file: %v\n", err)
		// Record failed upload
		if h.historyManager != nil {
			h.historyManager.RecordUpload(file.Filename, file.Size, false, fmt.Sprintf("Failed to save uploaded file: %v", err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to save uploaded file: %v", err),
		})
		return
	}
	tempFile.Close()

	fmt.Printf("Extracting to: %s\n", server.DefaultServerDir)

	// Extract the zip file
	if err := extractZipToServerDir(tempFilePath, server.DefaultServerDir); err != nil {
		fmt.Printf("Error extracting ZIP: %v\n", err)
		// Record failed upload
		if h.historyManager != nil {
			h.historyManager.RecordUpload(file.Filename, file.Size, false, fmt.Sprintf("Failed to extract ZIP file: %v", err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to extract ZIP file: %v", err),
		})
		return
	}

	fmt.Printf("Upload successful: %s\n", file.Filename)
	fmt.Printf("Rebuilding Docker image...\n")

	// Rebuild Docker image with new server files
	if err := docker.BuildImage(docker.DefaultImageName); err != nil {
		fmt.Printf("Error rebuilding Docker image: %v\n", err)
		// Record failed upload
		if h.historyManager != nil {
			h.historyManager.RecordUpload(file.Filename, file.Size, false, fmt.Sprintf("Failed to rebuild Docker image: %v", err))
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to rebuild Docker image: %v", err),
		})
		return
	}

	fmt.Printf("Docker image rebuilt successfully\n")

	// Record successful upload
	if h.historyManager != nil {
		if err := h.historyManager.RecordUpload(file.Filename, file.Size, true, "Upload and Docker rebuild successful"); err != nil {
			fmt.Printf("Warning: Failed to record upload history: %v\n", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Release uploaded successfully",
		"file":    file.Filename,
		"size":    file.Size,
	})
}

// extractZipToServerDir extracts a zip file to the server directory
func extractZipToServerDir(zipPath, destDir string) error {
	// Open the zip file
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer r.Close()

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Clear existing files in the destination directory
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return fmt.Errorf("failed to read destination directory: %w", err)
	}

	for _, entry := range entries {
		// Skip .gitkeep files
		if entry.Name() == ".gitkeep" {
			continue
		}

		path := filepath.Join(destDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove existing file %s: %w", path, err)
		}
	}

	// Extract all files from zip
	for _, f := range r.File {
		if err := extractFile(f, destDir); err != nil {
			return fmt.Errorf("failed to extract %s: %w", f.Name, err)
		}
	}

	return nil
}

// extractFile extracts a single file from a zip archive
func extractFile(f *zip.File, destDir string) error {
	// Prevent path traversal attacks
	destPath := filepath.Join(destDir, f.Name)
	if !strings.HasPrefix(destPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", f.Name)
	}

	if f.FileInfo().IsDir() {
		// Create directory
		return os.MkdirAll(destPath, 0755)
	}

	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// Create destination file
	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Open source file from zip
	srcFile, err := f.Open()
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Copy contents
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return err
	}

	return nil
}
