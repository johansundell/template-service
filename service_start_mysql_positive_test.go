package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/johansundell/template-service/store"
	"github.com/johansundell/template-service/types"
)

func TestStart_WithMySQLStorage_UsesMockedConstructor(t *testing.T) {
	originalSettings := settings
	settings = types.AppSettings{Port: freeAddr(t), Timeout: 15 * time.Second, Storage: types.StorageMySQL, AuthToken: "test-token"}
	settings.MySqlSettings.Username = "user"
	settings.MySqlSettings.Host = "localhost"
	settings.MySqlSettings.Port = "3306"
	settings.MySqlSettings.Database = "db"
	defer func() { settings = originalSettings }()

	// Return a SQLite store instead of connecting to MySQL, and capture the config
	var captured types.MySQLSettings
	orig := newMySQLStore
	newMySQLStore = func(cfg types.MySQLSettings) (store.Store, error) {
		captured = cfg
		return store.NewSQLite(filepath.Join(t.TempDir(), "test_mysql_positive.db"))
	}
	defer func() { newMySQLStore = orig }()

	p := newProgram()
	if err := p.startWorker(); err != nil {
		t.Fatalf("expected startup to succeed, got error: %v", err)
	}
	if err := p.Stop(nil); err != nil {
		t.Fatalf("expected Stop to succeed, got error: %v", err)
	}

	if captured != settings.MySqlSettings {
		t.Errorf("expected the MYSQL_* settings to be passed on, got %+v", captured)
	}
}
