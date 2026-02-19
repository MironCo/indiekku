package security

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware returns a Gin middleware that validates the API key
func AuthMiddleware(validAPIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")

		// Check for Bearer token format
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
			})
			c.Abort()
			return
		}

		// Extract the token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format, expected 'Bearer <token>'",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate the token (reject empty tokens as a security measure)
		if token == "" || validAPIKey == "" || subtle.ConstantTimeCompare([]byte(token), []byte(validAPIKey)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid API key",
			})
			c.Abort()
			return
		}

		// Token is valid, continue to the next handler
		c.Next()
	}
}
