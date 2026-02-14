package security

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// SessionCookieName is the name of the session cookie
	SessionCookieName = "indiekku_session"
	// SessionDuration is how long sessions last
	SessionDuration = 24 * time.Hour
	// SessionIDLength is the length of session IDs in bytes
	SessionIDLength = 32
)

// Session represents an authenticated session
type Session struct {
	ID        string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// SessionStore manages active sessions
type SessionStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	apiKey   string
}

// NewSessionStore creates a new session store
func NewSessionStore(apiKey string) *SessionStore {
	store := &SessionStore{
		sessions: make(map[string]*Session),
		apiKey:   apiKey,
	}
	// Start cleanup goroutine
	go store.cleanupExpired()
	return store
}

// generateSessionID creates a cryptographically secure session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, SessionIDLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateSession creates a new session and returns it
func (s *SessionStore) CreateSession() (*Session, error) {
	id, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        id,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(SessionDuration),
	}

	s.mu.Lock()
	s.sessions[id] = session
	s.mu.Unlock()

	return session, nil
}

// ValidateSession checks if a session ID is valid
func (s *SessionStore) ValidateSession(id string) bool {
	s.mu.RLock()
	session, exists := s.sessions[id]
	s.mu.RUnlock()

	if !exists {
		return false
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		s.DeleteSession(id)
		return false
	}

	return true
}

// DeleteSession removes a session
func (s *SessionStore) DeleteSession(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// ValidateAPIKey validates an API key against the stored key
func (s *SessionStore) ValidateAPIKey(key string) bool {
	return key != "" && s.apiKey != "" && key == s.apiKey
}

// cleanupExpired periodically removes expired sessions
func (s *SessionStore) cleanupExpired() {
	ticker := time.NewTicker(time.Hour)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, session := range s.sessions {
			if now.After(session.ExpiresAt) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

// SessionAuthMiddleware returns middleware that validates session cookies
// Falls back to Bearer token auth for API compatibility
func SessionAuthMiddleware(store *SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First try session cookie
		if cookie, err := c.Cookie(SessionCookieName); err == nil {
			if store.ValidateSession(cookie) {
				c.Next()
				return
			}
		}

		// Fall back to Bearer token for API clients
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Extract Bearer token
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token := authHeader[7:]
				if store.ValidateAPIKey(token) {
					c.Next()
					return
				}
			}
		}

		// Check if this is an htmx request
		if c.GetHeader("HX-Request") == "true" {
			// Return 401 with HX-Redirect header to trigger client-side redirect
			c.Header("HX-Redirect", "/")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		c.Abort()
	}
}

// SetSessionCookie sets the session cookie on the response
func SetSessionCookie(c *gin.Context, session *Session) {
	// Calculate max age in seconds
	maxAge := int(time.Until(session.ExpiresAt).Seconds())

	c.SetCookie(
		SessionCookieName,
		session.ID,
		maxAge,
		"/",
		"",    // domain (empty = current host)
		false, // secure (false for localhost, set true in production)
		true,  // httpOnly
	)
}

// ClearSessionCookie clears the session cookie
func ClearSessionCookie(c *gin.Context) {
	c.SetCookie(
		SessionCookieName,
		"",
		-1, // max age -1 = delete
		"/",
		"",
		false,
		true,
	)
}
