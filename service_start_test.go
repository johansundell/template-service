package main

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/johansundell/template-service/types"
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
