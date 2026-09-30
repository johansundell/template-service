package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/johansundell/template-service/types"
	"github.com/johansundell/template-service/utils"
	"github.com/joho/godotenv"
)

var settings types.AppSettings

func init() {
	loadSettings()
}

func loadSettings(filenames ...string) {
	// Load or reload .env file (Overload overrides already set environment variables)
	if len(filenames) > 0 {
		if err := godotenv.Overload(filenames...); err != nil {
			log.Println("No .env file found, using default/environment values")
		}
	} else {
		if err := godotenv.Overload(); err != nil {
			if exe, exeErr := os.Executable(); exeErr == nil {
				envPath := filepath.Join(filepath.Dir(exe), ".env")
				if err = godotenv.Overload(envPath); err != nil {
					log.Println("No .env file found, using default/environment values")
				}
			} else {
				log.Println("No .env file found, using default/environment values")
			}
		}
	}

	settings = types.AppSettings{}

	settings.Debug, _ = strconv.ParseBool(os.Getenv("DEBUG"))
	settings.Port = os.Getenv("PORT")
	if settings.Port == "" {
		settings.Port = ":8080"
	}
	settings.UseFileSystem, _ = strconv.ParseBool(os.Getenv("USE_FILE_SYSTEM"))

	timeoutStr := os.Getenv("TIMEOUT")
	if timeoutStr != "" {
		settings.Timeout, _ = strconv.Atoi(timeoutStr)
	} else {
		settings.Timeout = 15
	}

	settings.Storage = strings.ToLower(strings.TrimSpace(os.Getenv("STORAGE")))
	if settings.Storage == "" {
		settings.Storage = types.StorageSQLite
	}
	settings.SqlitePath = os.Getenv("SQLITE_PATH")
	if settings.SqlitePath == "" {
		settings.SqlitePath = filepath.Join(utils.GetBinaryBasePath(), nameOfService+".db")
	}
	settings.AuthToken = os.Getenv("AUTH_TOKEN")

	settings.MySqlSettings.Username = os.Getenv("MYSQL_USERNAME")
	settings.MySqlSettings.Password = os.Getenv("MYSQL_PASSWORD")
	settings.MySqlSettings.Host = os.Getenv("MYSQL_HOST")
	settings.MySqlSettings.Port = strings.TrimPrefix(os.Getenv("MYSQL_PORT"), ":")
	if settings.MySqlSettings.Port == "" {
		settings.MySqlSettings.Port = "3306"
	}
	settings.MySqlSettings.Database = os.Getenv("MYSQL_DATABASE")

}
