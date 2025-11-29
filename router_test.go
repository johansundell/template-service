package main

import (
	"net/http"
	"net/http/httptest"
	"os"
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

	// Setup temporary database
	tmpFile := "test_router_auth.db"
	defer os.Remove(tmpFile)

	db, err := store.NewSqliteDatabase(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	s := store.NewStorage(db)
	h := &handlers.Handler{}

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
