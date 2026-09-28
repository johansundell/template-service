package store

import (
	"os"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
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

func TestNewSqliteDatabase_Pragmas(t *testing.T) {
	tmpFile := "test_pragmas.db"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + "-wal")
	defer os.Remove(tmpFile + "-shm")

	db, err := NewSqliteDatabase(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("Failed to query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("Expected journal_mode wal, got %q", journalMode)
	}

	var busyTimeout int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("Failed to query busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Errorf("Expected busy_timeout 5000, got %d", busyTimeout)
	}
}
