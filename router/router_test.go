package router_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
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

	tmpFile := "test_router_auth.db"
	defer os.Remove(tmpFile)

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

	t.Run("Missing Auth Header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pong", bytes.NewBufferString(`{"test":"data"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("Invalid Auth Header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pong", bytes.NewBufferString(`{"test":"data"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "wrong-token")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("Valid Auth Header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pong", bytes.NewBufferString(`{"test":"data"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "secret-token")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Valid Bearer Auth Header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/pong", bytes.NewBufferString(`{"test":"data"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer secret-token")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestStartupAuthValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// AuthToken is empty, but default routes contain protected routes (Pong, GetLogs)
	settings := types.AppSettings{
		AuthToken: "",
	}

	h := handlers.NewHandler(nil, false, fstest.MapFS{}, "test", "dev")

	_, err := router.NewRouter(router.Config{
		Handler:  h,
		Settings: settings,
	})
	if err == nil {
		t.Fatalf("Expected NewRouter to fail when protected routes exist without AuthToken, got nil")
	}
}

func TestAuthMiddleware_FailClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Standalone AuthMiddleware with empty token must return 500 and not panic
	mw := router.AuthMiddleware("")
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
