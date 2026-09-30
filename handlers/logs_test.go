package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/johansundell/template-service/store"
	"github.com/johansundell/template-service/types"
)

func TestGetLogsHandler(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Mock FS (not used but required by NewHandler)
	mockFS := fstest.MapFS{}

	// Create in-memory DB
	s, err := store.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("Failed to create db: %v", err)
	}
	defer s.Close()

	// Insert some test data
	now := time.Now().UTC()
	err = s.LogRequest(context.Background(), types.UsageLog{Status: 200, Method: "GET", Endpoint: "/test", CreatedAt: now, Response: "{}", Request: "{}"})
	if err != nil {
		t.Fatalf("Failed to insert log: %v", err)
	}

	h := NewHandler(s, false, mockFS, "test-service", "v1.0")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/logs", nil)

	// Set params: dates are UTC days
	c.Params = gin.Params{
		{Key: "from", Value: now.Format("2006-01-02")},
		{Key: "to", Value: now.Format("2006-01-02")},
	}

	err = h.GetLogsHandler(c)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var logs []types.UsageLog
	if err := json.Unmarshal(w.Body.Bytes(), &logs); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(logs))
	}
}

func TestGetLogsHandler_ToIsInclusiveUTCDay(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s, err := store.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("Failed to create db: %v", err)
	}
	defer s.Close()

	day := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	for endpoint, at := range map[string]time.Time{
		"/late-on-day": day.Add(23*time.Hour + 59*time.Minute),
		"/next-day":    day.AddDate(0, 0, 1),
	} {
		if err := s.LogRequest(context.Background(), types.UsageLog{Endpoint: endpoint, CreatedAt: at}); err != nil {
			t.Fatalf("Failed to insert log: %v", err)
		}
	}

	h := NewHandler(s, false, fstest.MapFS{}, "test-service", "v1.0")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/logs", nil)
	c.Params = gin.Params{{Key: "from", Value: "2026-09-30"}, {Key: "to", Value: "2026-09-30"}}

	if err := h.GetLogsHandler(c); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var logs []types.UsageLog
	if err := json.Unmarshal(w.Body.Bytes(), &logs); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if len(logs) != 1 || logs[0].Endpoint != "/late-on-day" {
		t.Errorf("Expected only /late-on-day, got %+v", logs)
	}
}
