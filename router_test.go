package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/johansundell/template-service/handlers"
	"github.com/johansundell/template-service/store"
)

func TestAuthCheck(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Mock settings
	settings.AuthToken = "secret-token"

	// Mock handler and store (nil is fine for this test as we stop at auth)
	h := &handlers.Handler{}
	s := &store.Storage{}

	router := NewRouter(h, s)

	t.Run("Missing Auth Header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pong/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("Invalid Auth Header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pong/test", nil)
		req.Header.Set("Authorization", "wrong-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("Valid Auth Header", func(t *testing.T) {
		// We need a real handler for success case because it will try to call handler.Pong
		// But handler.Pong might panic if dependencies are nil?
		// Let's check handler.Pong implementation. It just returns JSON.
		// So it should be fine.

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pong/test", nil)
		req.Header.Set("Authorization", "secret-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Valid Bearer Auth Header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/pong/test", nil)
		req.Header.Set("Authorization", "Bearer secret-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}
