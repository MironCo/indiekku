package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validAPIKey := "test-api-key-12345"
	middleware := AuthMiddleware(validAPIKey)

	// Create test router
	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Create request with valid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+validAPIKey)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validAPIKey := "test-api-key-12345"
	middleware := AuthMiddleware(validAPIKey)

	// Create test router
	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Create request with invalid token
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_MissingAuthHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validAPIKey := "test-api-key-12345"
	middleware := AuthMiddleware(validAPIKey)

	// Create test router
	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Create request without Authorization header
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidAuthFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validAPIKey := "test-api-key-12345"
	middleware := AuthMiddleware(validAPIKey)

	// Create test router
	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	testCases := []struct {
		name   string
		header string
	}{
		{"missing Bearer prefix", "test-api-key-12345"},
		{"wrong prefix", "Basic test-api-key-12345"},
		{"only Bearer", "Bearer"},
		{"empty Bearer", "Bearer "},
		{"extra spaces", "Bearer  test-api-key-12345"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tc.header)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Test '%s': Expected status 401, got %d", tc.name, w.Code)
			}
		})
	}
}

func TestAuthMiddleware_CaseSensitive(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validAPIKey := "test-api-key-12345"
	middleware := AuthMiddleware(validAPIKey)

	// Create test router
	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Test with lowercase 'bearer'
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "bearer "+validAPIKey)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should fail - Bearer should be capitalized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for lowercase 'bearer', got %d", w.Code)
	}
}

func TestAuthMiddleware_EmptyAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Empty valid API key
	middleware := AuthMiddleware("")

	// Create test router
	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Even with empty token in request, should fail
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_MultipleRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validAPIKey := "test-api-key-12345"
	middleware := AuthMiddleware(validAPIKey)

	// Create test router
	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make multiple requests with valid token
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+validAPIKey)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i, w.Code)
		}
	}
}
