package store

import (
	"database/sql"
	"time"

	"github.com/go-sql-driver/mysql"
)

// NewMySQL connects to MySQL and returns a store that owns the connection.
// Timestamps are stored in UTC.
func NewMySQL(cfg mysql.Config) (*SQLStore, error) {
	cfg.ParseTime = true
	cfg.Loc = time.UTC

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}

	// Connection pool tuning
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS request_logs (
		id INT AUTO_INCREMENT PRIMARY KEY,
		status INT,
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

	return &SQLStore{
		db:      db,
		timeArg: func(t time.Time) any { return t.UTC() },
	}, nil
}
