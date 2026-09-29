package router_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"github.com/johansundell/template-service/handlers"
	"github.com/johansundell/template-service/router"
	"github.com/johansundell/template-service/store"
	"github.com/johansundell/template-service/types"
)

func TestAuthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	settings := types.AppSettings{
		AuthToken: "secret-token",
	}

	tmpFile := filepath.Join(t.TempDir(), "test_router_auth.db")

	db, err := store.NewSqliteDatabase(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	s := store.NewStorage(db)
	h := handlers.NewHandler(s, false, fstest.MapFS{}, "test", "dev")

	r, err := router.NewRouter(router.Config{
		Handler:  h,
		Store:    s,
		Settings: settings,
		Assets:   fstest.MapFS{},
		Version:  "dev",
	})
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}

	tests := []struct {
		name       string
		authHeader string
		body       string
		wantStatus int
	}{
		{
			name:       "Missing Auth Header",
			authHeader: "",
			body:       `{"test":"data"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Invalid Auth Header",
			authHeader: "wrong-token",
			body:       `{"test":"data"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Valid Auth Header",
			authHeader: "secret-token",
			body:       `{"test":"data"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Valid Bearer Auth Header",
			authHeader: "Bearer secret-token",
			body:       `{"test":"data"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Valid Auth With Invalid JSON",
			authHeader: "secret-token",
			body:       `{invalid`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/pong", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("Expected status %d, got %d", tc.wantStatus, w.Code)
			}
		})
	}
}

func TestStartupAuthValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// AuthToken is empty, but default routes contain protected routes (Pong, GetLogs)
	settings := types.AppSettings{
		AuthToken: "",
	}

	mockStore := store.NewStorage(nil)
	h := handlers.NewHandler(mockStore, false, fstest.MapFS{}, "test", "dev")

	_, err := router.NewRouter(router.Config{
		Handler:  h,
		Store:    mockStore,
		Settings: settings,
	})
	if err == nil {
		t.Fatalf("Expected NewRouter to fail when protected routes exist without AuthToken, got nil")
	}
	if !strings.Contains(err.Error(), "AUTH_TOKEN must be configured") {
		t.Fatalf("Expected AUTH_TOKEN error, got: %v", err)
	}
}

func TestAuthMiddleware_FailClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Standalone AuthMiddleware with empty token must return 500 and not panic
	mw := router.AuthMiddleware(nil, "")
	handler := mw(func(c *gin.Context) error {
		return nil
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/protected", nil)

	err := handler(c)
	if err == nil {
		t.Fatalf("Expected error from AuthMiddleware with empty token, got nil")
	}

	// Verify WrapHandler turns it into 500 Internal Server Error
	wrapped := router.WrapHandler(handler, "1.0.0")
	wrapped(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestWrapHandler_VersionHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const testVersion = "v1.2.3-test"
	wrapped := router.WrapHandler(func(c *gin.Context) error {
		c.Status(http.StatusOK)
		return nil
	}, testVersion)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)

	wrapped(c)

	if got := w.Header().Get("X-Version"); got != testVersion {
		t.Errorf("Expected X-Version %q, got %q", testVersion, got)
	}
}

func TestGetRoutes(t *testing.T) {
	mockStore := store.NewStorage(nil)
	h := handlers.NewHandler(mockStore, false, fstest.MapFS{}, "test", "dev")

	routes := router.GetRoutes(h)
	if len(routes) == 0 {
		t.Fatal("Expected GetRoutes to return route definitions, got empty")
	}

	expectedRoutes := map[string]struct {
		method    string
		pattern   string
		useLogger bool
		useAuth   bool
	}{
		"HealthCheck": {method: "GET", pattern: "/", useLogger: false, useAuth: false},
		"Ping":        {method: "GET", pattern: "/ping/:argument", useLogger: true, useAuth: false},
		"Pong":        {method: "POST", pattern: "/pong", useLogger: true, useAuth: true},
		"GetLogs":     {method: "GET", pattern: "/logs/:from/:to", useLogger: false, useAuth: true},
	}

	if len(routes) != len(expectedRoutes) {
		t.Errorf("Expected %d routes, got %d", len(expectedRoutes), len(routes))
	}

	for _, route := range routes {
		expected, exists := expectedRoutes[route.Name]
		if !exists {
			t.Errorf("Unexpected route %q", route.Name)
			continue
		}
		if route.Method != expected.method {
			t.Errorf("Route %q: expected method %q, got %q", route.Name, expected.method, route.Method)
		}
		if route.Pattern != expected.pattern {
			t.Errorf("Route %q: expected pattern %q, got %q", route.Name, expected.pattern, route.Pattern)
		}
		if route.UseLogger != expected.useLogger {
			t.Errorf("Route %q: expected UseLogger=%v, got %v", route.Name, expected.useLogger, route.UseLogger)
		}
		if route.UseAuth != expected.useAuth {
			t.Errorf("Route %q: expected UseAuth=%v, got %v", route.Name, expected.useAuth, route.UseAuth)
		}
		if route.HandlerFunc == nil {
			t.Errorf("Route %q: HandlerFunc should not be nil", route.Name)
		}
	}
}

func TestNewRouter_RequiresHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	_, err := router.NewRouter(router.Config{
		Version: "v1.0.0",
	})
	if err == nil {
		t.Fatalf("Expected NewRouter to fail when Handler is not provided, got nil")
	}
	expected := "handler must be provided"
	if err.Error() != expected {
		t.Errorf("Expected error %q, got %q", expected, err.Error())
	}
}

func TestStartupLoggerStoreValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	settings := types.AppSettings{
		AuthToken: "secret-token",
	}

	h := handlers.NewHandler(nil, false, fstest.MapFS{}, "test", "dev")

	_, err := router.NewRouter(router.Config{
		Handler:  h,
		Store:    nil,
		Settings: settings,
	})
	if err == nil {
		t.Fatalf("Expected NewRouter to fail when logged routes exist without Store, got nil")
	}
	expected := `store must be configured for logged route "Ping"`
	if err.Error() != expected {
		t.Errorf("Expected error %q, got %q", expected, err.Error())
	}
}

func TestNewRouter_EmbeddedModeNilAssetsReturnsError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockStore := store.NewStorage(nil)
	h := handlers.NewHandler(mockStore, false, fstest.MapFS{}, "test", "dev")

	_, err := router.NewRouter(router.Config{
		Handler:  h,
		Store:    mockStore,
		Assets:   nil,
		Settings: types.AppSettings{AuthToken: "secret-token", UseFileSystem: false},
		Version:  "v1.0.0",
	})
	if err == nil {
		t.Fatalf("Expected NewRouter to fail when Assets is nil in embedded mode, got nil")
	}
	expected := "embedded assets filesystem is nil"
	if err.Error() != expected {
		t.Errorf("Expected error %q, got %q", expected, err.Error())
	}
}

func TestNewRouter_FileSystemModeNilAssetsSucceeds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockStore := store.NewStorage(nil)
	h := handlers.NewHandler(mockStore, false, fstest.MapFS{}, "test", "dev")

	r, err := router.NewRouter(router.Config{
		Handler:  h,
		Store:    mockStore,
		Assets:   nil,
		Settings: types.AppSettings{AuthToken: "secret-token", UseFileSystem: true},
		Version:  "v1.0.0",
	})
	if err != nil {
		t.Fatalf("Expected NewRouter to succeed in filesystem mode without embedded assets, got: %v", err)
	}
	if r == nil {
		t.Fatal("Expected non-nil router")
	}
}

type testLogger struct {
	messages []string
}

func (tl *testLogger) Printf(format string, v ...interface{}) {
	tl.messages = append(tl.messages, fmt.Sprintf(format, v...))
}

func TestNewRouter_InjectedLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	settings := types.AppSettings{
		AuthToken: "secret-token",
	}

	tmpFile := filepath.Join(t.TempDir(), "test_router_logger.db")
	db, err := store.NewSqliteDatabase(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	s := store.NewStorage(db)
	h := handlers.NewHandler(s, false, fstest.MapFS{}, "test", "dev")

	tl := &testLogger{}
	r, err := router.NewRouter(router.Config{
		Handler:  h,
		Store:    s,
		Settings: settings,
		Assets:   fstest.MapFS{},
		Version:  "dev",
		Logger:   tl,
	})
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/pong", bytes.NewBufferString(`{"test":"data"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "secret-token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if len(tl.messages) == 0 {
		t.Errorf("Expected injected logger to receive log messages, got none")
	}

	found := false
	for _, msg := range tl.messages {
		if strings.Contains(msg, "request logged: POST /pong 200") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected log containing 'request logged: POST /pong 200', got %v", tl.messages)
	}
}

func TestAuthMiddleware_InjectedLoggerOnEmptyToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tl := &testLogger{}
	mw := router.AuthMiddleware(nil, "", tl)
	handler := mw(func(c *gin.Context) error { return nil })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/protected", nil)

	err := handler(c)
	if err == nil {
		t.Fatalf("Expected error from AuthMiddleware with empty token, got nil")
	}

	if len(tl.messages) == 0 {
		t.Errorf("Expected injected logger to log warning on empty token")
	} else if !strings.Contains(tl.messages[0], "AUTH_TOKEN is not set") {
		t.Errorf("Expected warning message, got %q", tl.messages[0])
	}
}
