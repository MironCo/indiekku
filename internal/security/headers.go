package security

import "github.com/gin-gonic/gin"

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking attacks
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS protection (for older browsers)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		// Allow 'self' for scripts/styles, and unpkg.com for HTMX
		// 'unsafe-inline' needed for inline styles in the brutalist UI
		// Note: For a local dev tool, this is acceptable. For production, use nonces/hashes.
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' https://unpkg.com; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")

		// Permissions policy (disable unnecessary features)
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}
