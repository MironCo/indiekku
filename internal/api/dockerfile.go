package api

import (
	"fmt"
	"io"
	"net/http"

	"github.com/MironCo/indiekku/internal/docker"
	"github.com/MironCo/indiekku/internal/validation"

	"github.com/gin-gonic/gin"
)

// ListDockerfilePresets handles GET /dockerfiles/presets
func (h *ApiHandler) ListDockerfilePresets(c *gin.Context) {
	presets := docker.ListPresets()
	presetsWithContent := make([]map[string]string, 0, len(presets))

	for _, name := range presets {
		content, _ := docker.GetPreset(name)
		presetsWithContent = append(presetsWithContent, map[string]string{
			"name":    name,
			"content": content,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"presets": presetsWithContent,
	})
}

// GetActiveDockerfile handles GET /dockerfiles/active
func (h *ApiHandler) GetActiveDockerfile(c *gin.Context) {
	content, err := docker.GetActiveDockerfile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get active Dockerfile: %v", err),
		})
		return
	}

	name := docker.GetActiveDockerfileName()

	c.JSON(http.StatusOK, gin.H{
		"name":    name,
		"content": content,
	})
}

// SetActiveDockerfileRequest represents the request for setting active Dockerfile
type SetActiveDockerfileRequest struct {
	Preset string `json:"preset,omitempty"` // Use a preset by name
}

// SetActiveDockerfile handles POST /dockerfiles/active
// Accepts either JSON with preset name, or multipart form with dockerfile file
func (h *ApiHandler) SetActiveDockerfile(c *gin.Context) {
	contentType := c.GetHeader("Content-Type")

	// Handle multipart form (custom Dockerfile upload)
	if contentType == "multipart/form-data" || len(contentType) > 19 && contentType[:19] == "multipart/form-data" {
		file, header, err := c.Request.FormFile("dockerfile")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Failed to read dockerfile from form",
			})
			return
		}
		defer file.Close()

		content, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to read dockerfile content",
			})
			return
		}

		// Validate dockerfile content
		if result := validation.ValidateDockerfile(string(content)); !result.Valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": result.Message})
			return
		}

		if err := docker.SetActiveDockerfile(string(content)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to set active Dockerfile: %v", err),
			})
			return
		}

		// Record in history
		if h.historyManager != nil {
			h.historyManager.RecordDockerfileChange(header.Filename, "custom", "Uploaded via API")
		}

		// Remove old image to force rebuild
		docker.RemoveImage(h.imageName)

		c.JSON(http.StatusOK, gin.H{
			"message": "Active Dockerfile set from upload",
			"name":    header.Filename,
		})
		return
	}

	// Handle JSON (preset selection)
	var req SetActiveDockerfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	if req.Preset == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Preset name is required",
		})
		return
	}

	// Validate preset name
	if result := validation.ValidatePresetName(req.Preset); !result.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Message})
		return
	}

	if err := docker.SetActiveFromPreset(req.Preset); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to set preset: %v", err),
		})
		return
	}

	// Record in history
	if h.historyManager != nil {
		h.historyManager.RecordDockerfileChange(req.Preset, "preset:"+req.Preset, "Set via API")
	}

	// Remove old image to force rebuild
	docker.RemoveImage(h.imageName)

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Active Dockerfile set to preset: %s", req.Preset),
		"name":    req.Preset,
	})
}

// GetDockerfileHistory handles GET /dockerfiles/history
func (h *ApiHandler) GetDockerfileHistory(c *gin.Context) {
	if h.historyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "History tracking not enabled",
		})
		return
	}

	history, err := h.historyManager.GetDockerfileHistory(100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get Dockerfile history: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"count":   len(history),
	})
}
