package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetBinaryBasePath(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() returned an error: %v", err)
	}

	want := filepath.Dir(executable)
	if got := GetBinaryBasePath(); got != want {
		t.Fatalf("GetBinaryBasePath() = %q, want %q", got, want)
	}
}
