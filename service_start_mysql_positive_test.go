package main

import (
	"database/sql"
	"os"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/johansundell/template-service/store"
)

func TestStart_WithMySQLEnv_UsesMockedConstructor(t *testing.T) {
	// Backup envs
	keys := []string{"USE_MYSQL", "MYSQL_USERNAME", "MYSQL_HOST", "MYSQL_DATABASE", "MYSQL_PORT"}
	bak := map[string]string{}
	for _, k := range keys {
		bak[k] = os.Getenv(k)
	}
	defer func() {
		for k, v := range bak {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	os.Setenv("USE_MYSQL", "true")
	os.Setenv("MYSQL_USERNAME", "user")
	os.Setenv("MYSQL_HOST", "localhost")
	os.Setenv("MYSQL_DATABASE", "db")
	os.Setenv("MYSQL_PORT", "3306")

	// Override newMySQLStorage to return an on-disk sqlite DB so initialization succeeds
	orig := newMySQLStorage
	newMySQLStorage = func(cfg mysql.Config) (*sql.DB, error) {
		return store.NewSqliteDatabase("test_mysql_positive.db")
	}
	defer func() { newMySQLStorage = orig }()

	p := &program{}
	if err := p.Start(nil); err != nil {
		t.Fatalf("expected Start to succeed, got error: %v", err)
	}
	// stop immediately
	p.Stop(nil)
	// cleanup
	os.Remove("test_mysql_positive.db")
}
