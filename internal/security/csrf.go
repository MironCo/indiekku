package security

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// CSRFManager handles CSRF token generation and validation
type CSRFManager struct {
	tokens map[string]bool
	mu     sync.RWMutex
}

// NewCSRFManager creates a new CSRF manager
func NewCSRFManager() *CSRFManager {
	return &CSRFManager{
		tokens: make(map[string]bool),
	}
}

// GenerateToken generates a new CSRF token
func (m *CSRFManager) GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(b)

	m.mu.Lock()
	m.tokens[token] = true
	m.mu.Unlock()

	return token, nil
}

// ValidateToken validates a CSRF token
func (m *CSRFManager) ValidateToken(token string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tokens[token]
}

// InvalidateToken removes a token after use
func (m *CSRFManager) InvalidateToken(token string) {
	m.mu.Lock()
	delete(m.tokens, token)
	m.mu.Unlock()
}

// CSRFMiddleware validates CSRF tokens for state-changing requests
// Note: For a local development tool with API key authentication, CSRF protection
// is less critical since requests come from localhost. However, this provides
// defense-in-depth security.
func CSRFMiddleware(manager *CSRFManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only check CSRF for state-changing methods
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "DELETE" || c.Request.Method == "PATCH" {
			token := c.GetHeader("X-CSRF-Token")
			if token == "" {
				// Also check form value
				token = c.PostForm("csrf_token")
			}

			if token == "" || !manager.ValidateToken(token) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "Invalid or missing CSRF token",
				})
				c.Abort()
				return
			}

			// Invalidate after use so tokens are single-use
			manager.InvalidateToken(token)
		}

		c.Next()
	}
}
