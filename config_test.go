package main

import (
	"os"
	"testing"
)

func TestLoadSettings(t *testing.T) {
	// Create a temporary env file
	tmpEnv, err := os.CreateTemp("", ".env.*")
	if err != nil {
		t.Fatalf("Failed to create temp env file: %v", err)
	}
	defer os.Remove(tmpEnv.Name())

	_, err = tmpEnv.WriteString("PORT=:9999\nDEBUG=false\n")
	if err != nil {
		t.Fatalf("Failed to write to temp env file: %v", err)
	}
	tmpEnv.Close()

	loadSettings(tmpEnv.Name())

	if settings.Port != ":9999" {
		t.Errorf("expected settings.Port :9999, got %s", settings.Port)
	}
	if settings.Debug != false {
		t.Errorf("expected settings.Debug false, got %v", settings.Debug)
	}

	// Restore original settings
	loadSettings()
}
