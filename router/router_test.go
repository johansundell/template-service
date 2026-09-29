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

func TestCustomRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	called := false
	customRoutes := router.Routes{
		router.Route{
			Name:    "CustomRoute",
			Method:  "GET",
			Pattern: "/custom",
			HandlerFunc: func(c *gin.Context) error {
				called = true
				c.String(http.StatusOK, "custom response")
				return nil
			},
		},
	}

	r, err := router.NewRouter(router.Config{
		Routes:  customRoutes,
		Assets:  fstest.MapFS{},
		Version: "v1.0.0",
	})
	if err != nil {
		t.Fatalf("NewRouter with custom routes failed: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/custom", nil)
	r.ServeHTTP(w, req)

	if !called {
		t.Errorf("Expected custom handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestNewRouter_RequiresRoutesOrHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	_, err := router.NewRouter(router.Config{
		Version: "v1.0.0",
	})
	if err == nil {
		t.Fatalf("Expected NewRouter to fail when neither Routes nor Handler is provided, got nil")
	}
	expected := "routes or handler must be provided"
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

func TestNewRouter_LoggedRouteRequiresStore(t *testing.T) {
	gin.SetMode(gin.TestMode)

	customRoutes := router.Routes{
		router.Route{
			Name:        "LoggedRoute",
			Method:      "GET",
			Pattern:     "/logged",
			HandlerFunc: func(c *gin.Context) error { return nil },
			UseLogger:   true,
		},
	}

	_, err := router.NewRouter(router.Config{
		Routes:  customRoutes,
		Store:   nil,
		Version: "v1.0.0",
	})
	if err == nil {
		t.Fatalf("Expected NewRouter to fail when logged route has no Store, got nil")
	}
	expected := `store must be configured for logged route "LoggedRoute"`
	if err.Error() != expected {
		t.Errorf("Expected error %q, got %q", expected, err.Error())
	}
}

func TestNewRouter_EmbeddedModeNilAssetsReturnsError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	customRoutes := router.Routes{
		router.Route{
			Name:        "TestRoute",
			Method:      "GET",
			Pattern:     "/test",
			HandlerFunc: func(c *gin.Context) error { return nil },
		},
	}

	_, err := router.NewRouter(router.Config{
		Routes:   customRoutes,
		Assets:   nil,
		Settings: types.AppSettings{UseFileSystem: false},
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

	customRoutes := router.Routes{
		router.Route{
			Name:        "TestRoute",
			Method:      "GET",
			Pattern:     "/test",
			HandlerFunc: func(c *gin.Context) error { return nil },
		},
	}

	r, err := router.NewRouter(router.Config{
		Routes:   customRoutes,
		Assets:   nil,
		Settings: types.AppSettings{UseFileSystem: true},
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
