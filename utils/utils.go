package utils

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func GetUrl(r *http.Request, path string) string {
	query := r.URL.RawQuery
	if query != "" {
		return path + "?" + query
	}
	return path
}

func GetBinaryBasePath() string {
	exe, err := os.Executable()
	if err != nil {
		log.Println("Failed to get executable path, using current directory as base path:", err)
		return "./"
	}
	return filepath.Dir(exe)
}
