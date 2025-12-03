package store

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
)

func NewMySQLStorage(cfg mysql.Config) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}

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
		return nil, err
	}

	return db, nil
}
