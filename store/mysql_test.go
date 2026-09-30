package store

import (
	"testing"
	"time"

	"github.com/johansundell/template-service/types"
)

func TestMySQLConfig(t *testing.T) {
	cfg := mysqlConfig(types.MySQLSettings{Username: "user", Password: "pw", Host: "db.local", Port: "3307", Database: "logs"})

	if cfg.User != "user" || cfg.Passwd != "pw" || cfg.Net != "tcp" || cfg.Addr != "db.local:3307" || cfg.DBName != "logs" {
		t.Errorf("unexpected mapping: %+v", cfg)
	}
	if !cfg.ParseTime || cfg.Loc != time.UTC || !cfg.AllowNativePasswords {
		t.Errorf("expected ParseTime, UTC and native passwords, got ParseTime=%v Loc=%v Native=%v", cfg.ParseTime, cfg.Loc, cfg.AllowNativePasswords)
	}
}
