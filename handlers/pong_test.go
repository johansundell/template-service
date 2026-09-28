package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/johansundell/template-service/httperror"
)

func TestPong(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{}

	t.Run("Valid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/pong", bytes.NewBufferString(`{"key":"value"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		err := h.Pong(c)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		msg, ok := response["message"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected message to be a map, got %v", response["message"])
		}
		if msg["key"] != "value" {
			t.Errorf("Expected key 'value', got %v", msg["key"])
		}
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/pong", bytes.NewBufferString(`{invalid json`))
		c.Request.Header.Set("Content-Type", "application/json")

		err := h.Pong(c)
		if err == nil {
			t.Fatal("Expected error for invalid JSON, got nil")
		}

		status := httperror.HTTPStatus(err)
		if status != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
		}
	})

	t.Run("Empty body", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/pong", bytes.NewBufferString(``))
		c.Request.Header.Set("Content-Type", "application/json")

		err := h.Pong(c)
		if err == nil {
			t.Fatal("Expected error for empty body, got nil")
		}

		status := httperror.HTTPStatus(err)
		if status != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, status)
		}
	})
}
