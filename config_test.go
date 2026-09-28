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

func TestLoadSettings_NoDefaultAuthToken(t *testing.T) {
	t.Setenv("AUTH_TOKEN", "")

	tmpEnv, err := os.CreateTemp("", ".env.*")
	if err != nil {
		t.Fatalf("Failed to create temp env file: %v", err)
	}
	defer os.Remove(tmpEnv.Name())

	_, err = tmpEnv.WriteString("PORT=:9999\n")
	if err != nil {
		t.Fatalf("Failed to write to temp env file: %v", err)
	}
	tmpEnv.Close()

	loadSettings(tmpEnv.Name())

	if settings.AuthToken != "" {
		t.Errorf("expected empty settings.AuthToken, got %q", settings.AuthToken)
	}

	// Restore original settings
	loadSettings()
}

func TestLoadSettings_MySQLPort(t *testing.T) {
	testPort := func(t *testing.T, content string) string {
		tmpEnv, err := os.CreateTemp("", ".env.*")
		if err != nil {
			t.Fatalf("Failed to create temp env file: %v", err)
		}
		defer os.Remove(tmpEnv.Name())

		if _, err := tmpEnv.WriteString(content); err != nil {
			t.Fatalf("Failed to write to temp env file: %v", err)
		}
		tmpEnv.Close()

		loadSettings(tmpEnv.Name())
		return settings.MySqlSettings.Port
	}

	t.Run("port with leading colon", func(t *testing.T) {
		got := testPort(t, "MYSQL_PORT=:3306\n")
		if got != "3306" {
			t.Errorf("expected settings.MySqlSettings.Port '3306', got %q", got)
		}
	})

	t.Run("port without leading colon", func(t *testing.T) {
		got := testPort(t, "MYSQL_PORT=3307\n")
		if got != "3307" {
			t.Errorf("expected settings.MySqlSettings.Port '3307', got %q", got)
		}
	})

	t.Run("default port when unset", func(t *testing.T) {
		t.Setenv("MYSQL_PORT", "")
		got := testPort(t, "DEBUG=true\n")
		if got != "3306" {
			t.Errorf("expected default settings.MySqlSettings.Port '3306', got %q", got)
		}
	})

	// Restore original settings
	loadSettings()
}
