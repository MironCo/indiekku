package api

import (
	"net/http"

	"indiekku/internal/security"

	"github.com/gin-gonic/gin"
)

// LoginRequest represents the login request body
type LoginRequest struct {
	APIKey string `json:"api_key" form:"api_key"`
}

// HandleLogin processes login requests and creates sessions
func (h *ApiHandler) HandleLogin(c *gin.Context) {
	var req LoginRequest

	// Support both JSON and form data
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Validate API key
	if req.APIKey != h.apiKey {
		// Check if htmx request
		if c.GetHeader("HX-Request") == "true" {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusUnauthorized, `<div class="message show error">Invalid API key</div>`)
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
		return
	}

	// Create session
	session, err := h.sessionStore.CreateSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// Set session cookie
	security.SetSessionCookie(c, session)

	// Check if htmx request - redirect to dashboard
	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/")
		c.Status(http.StatusOK)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
}

// HandleLogout clears the session and redirects to login
func (h *ApiHandler) HandleLogout(c *gin.Context) {
	// Get session from cookie
	if sessionID, err := c.Cookie(security.SessionCookieName); err == nil {
		h.sessionStore.DeleteSession(sessionID)
	}

	// Clear the cookie
	security.ClearSessionCookie(c)

	// Check if htmx request
	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", "/")
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/")
}

// CheckAuth returns whether the current request is authenticated
func (h *ApiHandler) CheckAuth(c *gin.Context) {
	// Check session cookie
	if sessionID, err := c.Cookie(security.SessionCookieName); err == nil {
		if h.sessionStore.ValidateSession(sessionID) {
			c.JSON(http.StatusOK, gin.H{"authenticated": true})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"authenticated": false})
}
