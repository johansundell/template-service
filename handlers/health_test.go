package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"github.com/johansundell/template-service/store"
)

func TestHealthCheck(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Mock FS
	mockFS := fstest.MapFS{
		"tmpl/base.html":   {Data: []byte(`{{define "base"}}{{template "content" .}}{{end}}`)},
		"tmpl/health.html": {Data: []byte(`{{define "content"}}Database: {{.dbStatus}}{{end}}`)},
	}

	s, err := store.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("Failed to create db: %v", err)
	}
	defer s.Close()

	h := NewHandler(s, false, mockFS, "test-service", "v1.0")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	err = h.HealthCheck(c)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	expectedBody := "Database: OK"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, w.Body.String())
	}
}

func TestHealthCheckJSON(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Mock FS (not needed for JSON, but required for NewHandler)
	mockFS := fstest.MapFS{}

	s, err := store.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("Failed to create db: %v", err)
	}
	defer s.Close()

	h := NewHandler(s, false, mockFS, "test-service", "v1.0")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Set Accept header
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Accept", "application/json")

	err = h.HealthCheck(c)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	expectedJSON := `{"dbStatus":"OK","name":"test-service","title":"Health Check","version":"v1.0"}`
	if w.Body.String() != expectedJSON {
		t.Errorf("Expected body '%s', got '%s'", expectedJSON, w.Body.String())
	}
}
