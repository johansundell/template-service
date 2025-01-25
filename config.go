package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/johansundell/template-service/types"
)

const filenameSettings = "settings.json"

var settings types.AppSettings

func init() {
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	dir, _ := filepath.Split(ex)
	dat, err := os.ReadFile(dir + filenameSettings)
	if err != nil {
		settings = types.AppSettings{}
		settings.Timeout = 15
		settings.Port = ":8080"
		settings.UseFileSystem = false
		data, err := json.Marshal(settings)
		if err != nil {
			log.Fatal("Could not write settings", err)
		}
		os.WriteFile(dir+filenameSettings, data, 0664)
		log.Fatal("settings.json missing")
	}

	if err := json.Unmarshal(dat, &settings); err != nil {
		log.Fatal(err)
	}
}
