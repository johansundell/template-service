package handlers

import (
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
	db, err := store.NewSqliteDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to create db: %v", err)
	}
	defer db.Close()
	s := store.NewStorage(db)

	// Insert some test data
	err = s.LogRequest(200, "GET", "", "/test", time.Now().Format(time.RFC3339), "{}", "{}")
	if err != nil {
		t.Fatalf("Failed to insert log: %v", err)
	}

	h := NewHandler(s, false, mockFS, "test-service", "v1.0")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Set params
	c.Params = gin.Params{
		{Key: "from", Value: time.Now().Format("2006-01-02")},
		{Key: "to", Value: time.Now().Format("2006-01-02")},
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
