package store

import (
	"database/sql"

	//_ "github.com/mattn/go-sqlite3"
	//_ "modernc.org/sqlite"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func NewSqliteDatabase(file string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "file:"+file)
	if err != nil {
		return nil, err
	}

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
		return nil, err
	}

	return db, nil
}
