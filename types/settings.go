package types

import (
	"fmt"
	"strings"
)

// Storage backends selectable with STORAGE.
const (
	StorageSQLite = "sqlite"
	StorageMySQL  = "mysql"
)

type AppSettings struct {
	Debug         bool   `json:"debug"`
	Port          string `json:"port"`
	UseFileSystem bool   `json:"useFileSystem"`
	Timeout       int    `json:"timeout"`
	Storage       string `json:"storage"`
	AuthToken     string `json:"authToken"`
	SqlitePath    string `json:"sqlitePath"`
	MySqlSettings struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Host     string `json:"host"`
		Port     string `json:"port"`
		Database string `json:"database"`
	} `json:"mysql"`
}

// Validate verifies required settings and returns an error when configuration is invalid.
func (s AppSettings) Validate() error {
	if s.Port == "" {
		return fmt.Errorf("PORT must be set")
	}
	if s.Timeout <= 0 {
		return fmt.Errorf("TIMEOUT must be > 0")
	}
	switch s.Storage {
	case StorageSQLite:
	case StorageMySQL:
		if s.MySqlSettings.Username == "" || s.MySqlSettings.Host == "" || s.MySqlSettings.Database == "" {
			return fmt.Errorf("MYSQL_USERNAME, MYSQL_HOST and MYSQL_DATABASE must be set when STORAGE=mysql")
		}
		if strings.HasPrefix(s.MySqlSettings.Port, ":") {
			return fmt.Errorf("MYSQL_PORT must not contain leading ':'")
		}
	default:
		return fmt.Errorf("STORAGE must be %q or %q, got %q", StorageSQLite, StorageMySQL, s.Storage)
	}
	return nil
}
