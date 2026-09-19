package store

import (
	"os"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func TestLogRequest(t *testing.T) {
	// Setup temporary database
	tmpFile := "test_log.db"
	defer os.Remove(tmpFile)

	db, err := NewSqliteDatabase(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	s := NewStorage(db)

	// Test LogRequest
	err = s.LogRequest(200, "GET", "", "/test", time.Now().Format(time.RFC3339), "{}", "{}")
	if err != nil {
		t.Errorf("LogRequest failed: %v", err)
	}

	// Verify log entry
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM request_logs").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query logs: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 log entry, got %d", count)
	}
}

func TestGetLogs(t *testing.T) {
	tmpFile := "test_get_logs.db"
	defer os.Remove(tmpFile)

	db, err := NewSqliteDatabase(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	s := NewStorage(db)

	now := time.Now()
	err = s.LogRequest(200, "GET", "", "/test", now.Format(time.RFC3339), "{}", "{}")
	if err != nil {
		t.Fatalf("LogRequest failed: %v", err)
	}

	from := now.Add(-1 * time.Hour)
	to := now.Add(1 * time.Hour)
	logs, err := s.GetLogs(from, to)
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}
	if logs[0].Endpoint != "/test" {
		t.Errorf("Expected endpoint /test, got %s", logs[0].Endpoint)
	}
}
