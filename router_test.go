package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/johansundell/template-service/handlers"
	"github.com/johansundell/template-service/store"
	"github.com/johansundell/template-service/types"
)

func TestAuthHeaderCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		authToken      string
		headerValue    string
		expectedStatus int
	}{
		{
			name:           "Valid Bearer token",
			authToken:      "test-token-123",
			headerValue:    "Bearer test-token-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid plain token",
			authToken:      "test-token-123",
			headerValue:    "test-token-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing authorization header",
			authToken:      "test-token-123",
			headerValue:    "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid token",
			authToken:      "test-token-123",
			headerValue:    "Bearer wrong-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "No auth token configured - should allow",
			authToken:      "",
			headerValue:    "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test settings
			settings = types.AppSettings{
				AuthToken: tt.authToken,
			}

			// Create a test router with a simple handler
			router := gin.New()
			testHandler := func(c *gin.Context) error {
				c.JSON(http.StatusOK, gin.H{"result": "success"})
				return nil
			}

			wrappedHandler := handlerWithLogger(testHandler, false, true)
			router.GET("/test", wrappedHandler)

			// Create request
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.headerValue != "" {
				req.Header.Set("Authorization", tt.headerValue)
			}

			// Record response
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestRouteWithoutAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set up test settings with auth token
	settings = types.AppSettings{
		AuthToken: "test-token-123",
	}

	// Create a test router with UseAuth=false
	router := gin.New()
	testHandler := func(c *gin.Context) error {
		c.JSON(http.StatusOK, gin.H{"result": "success"})
		return nil
	}

	wrappedHandler := handlerWithLogger(testHandler, false, false)
	router.GET("/test", wrappedHandler)

	// Create request without auth header
	req, _ := http.NewRequest("GET", "/test", nil)

	// Record response
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should succeed even without auth header when UseAuth=false
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for route without auth, got %d", http.StatusOK, w.Code)
	}
}

func TestNewRouterWithAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set up test settings
	settings = types.AppSettings{
		Port:          ":8080",
		UseFileSystem: true,
		AuthToken:     "test-token-123",
	}

	// Create a test database
	db, err := store.NewSqliteDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	storage := store.NewStorage(db)
	handler := handlers.NewHandler(storage, true, tpls, "test-service", "test-version")

	// Create router
	router := NewRouter(handler)

	// Test health check endpoint (no auth required)
	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for health check, got %d", http.StatusOK, w.Code)
	}

	// Test ping endpoint without auth (should fail)
	req, _ = http.NewRequest("GET", "/ping/test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d for ping without auth, got %d", http.StatusUnauthorized, w.Code)
	}

	// Test ping endpoint with auth (should succeed)
	req, _ = http.NewRequest("GET", "/ping/test", nil)
	req.Header.Set("Authorization", "Bearer test-token-123")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for ping with auth, got %d", http.StatusOK, w.Code)
	}
}
