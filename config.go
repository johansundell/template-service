package main

import (
	"log"
	"os"
	"strconv"

	"github.com/johansundell/template-service/types"
	"github.com/joho/godotenv"
)

var settings types.AppSettings

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using default/environment values")
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

	settings.UseMySQL, _ = strconv.ParseBool(os.Getenv("USE_MYSQL"))
	if !settings.UseMySQL {
		settings.UseSqlite = true
	}
	settings.AuthToken = os.Getenv("AUTH_TOKEN")

	settings.MySqlSettings.Username = os.Getenv("MYSQL_USERNAME")
	settings.MySqlSettings.Password = os.Getenv("MYSQL_PASSWORD")
	settings.MySqlSettings.Host = os.Getenv("MYSQL_HOST")
	settings.MySqlSettings.Port = os.Getenv("MYSQL_PORT")
	settings.MySqlSettings.Database = os.Getenv("MYSQL_DATABASE")
}
