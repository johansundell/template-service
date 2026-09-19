package store

import (
	"database/sql"
	"time"

	//_ "github.com/mattn/go-sqlite3"
	//_ "modernc.org/sqlite"
	_ "github.com/ncruces/go-sqlite3/driver"
	//_ "github.com/ncruces/go-sqlite3/embed"
)

func NewSqliteDatabase(file string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "file:"+file)
	if err != nil {
		return nil, err
	}

	// SQLite-specific pragmas for better concurrency and durability
	// Use WAL mode and set a busy timeout to reduce SQLITE_BUSY errors under contention
	db.Exec(`PRAGMA journal_mode = WAL`)
	db.Exec(`PRAGMA busy_timeout = 5000`)

	// Connection pool: limit to a single writer connection for file-backed SQLite
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0 * time.Second)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS request_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		status INTEGER,
		method TEXT,
		error TEXT,
		endpoint TEXT,
		created_at DATETIME,
		response TEXT,
		request TEXT
	)`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
