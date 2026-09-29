package main

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/johansundell/template-service/types"
	"github.com/johansundell/template-service/utils"
)

func TestStart_InvalidTimeoutReturnsError(t *testing.T) {
	settings := types.AppSettings{Port: ":8080", Timeout: 0}
	if err := settings.Validate(); err == nil {
		t.Fatal("expected validation to fail with an invalid timeout")
	}
}

func TestStart_DatabaseInitializationErrorReturnsError(t *testing.T) {
	originalConstructor := newSqliteDatabase
	newSqliteDatabase = func(string) (*sql.DB, error) {
		return nil, errors.New("database unavailable")
	}
	defer func() { newSqliteDatabase = originalConstructor }()

	originalSettings := settings
	settings = types.AppSettings{Port: ":8080", Timeout: 15, UseSqlite: true}
	defer func() { settings = originalSettings }()

	p := &program{}
	if err := p.run(make(chan error, 1)); err == nil {
		t.Fatal("expected run to return the database initialization error")
	}
}

func TestEnsureAuthToken(t *testing.T) {
	orig := settings.AuthToken
	defer func() { settings.AuthToken = orig }()

	settings.AuthToken = ""
	ensureAuthToken()
	first := settings.AuthToken
	if first == "" {
		t.Fatal("expected a generated token, got empty string")
	}

	settings.AuthToken = ""
	ensureAuthToken()
	if settings.AuthToken == first {
		t.Errorf("expected a new random token on each call, got %q twice", first)
	}

	settings.AuthToken = "configured"
	ensureAuthToken()
	if settings.AuthToken != "configured" {
		t.Errorf("expected configured token to be kept, got %q", settings.AuthToken)
	}
}

func TestStart_UsesConfiguredSqlitePath(t *testing.T) {
	var capturedPath string
	originalConstructor := newSqliteDatabase
	newSqliteDatabase = func(path string) (*sql.DB, error) {
		capturedPath = path
		return nil, errors.New("stop here")
	}
	defer func() { newSqliteDatabase = originalConstructor }()

	originalSettings := settings
	settings = types.AppSettings{Port: ":8080", Timeout: 15, UseSqlite: true, SqlitePath: "/custom/data/my.db"}
	defer func() { settings = originalSettings }()

	p := &program{}
	_ = p.run(make(chan error, 1))

	if capturedPath != "/custom/data/my.db" {
		t.Errorf("expected sqlite path /custom/data/my.db, got %q", capturedPath)
	}
}

func TestStart_UsesDefaultSqlitePathWhenEmpty(t *testing.T) {
	var capturedPath string
	originalConstructor := newSqliteDatabase
	newSqliteDatabase = func(path string) (*sql.DB, error) {
		capturedPath = path
		return nil, errors.New("stop here")
	}
	defer func() { newSqliteDatabase = originalConstructor }()

	originalSettings := settings
	settings = types.AppSettings{Port: ":8080", Timeout: 15, UseSqlite: true, SqlitePath: ""}
	defer func() { settings = originalSettings }()

	p := &program{}
	_ = p.run(make(chan error, 1))

	expected := filepath.Join(utils.GetBinaryBasePath(), nameOfService+".db")
	if capturedPath != expected {
		t.Errorf("expected default sqlite path %q, got %q", expected, capturedPath)
	}
}
