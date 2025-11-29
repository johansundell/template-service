package store

import "database/sql"

type Storage struct {
	db *sql.DB
}

func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

type Store interface {
	Ping() error
}

func (s *Storage) Ping() error {
	return s.db.Ping()
}

func (s *Storage) LogRequest(status int, method, errStr, endpoint string, createdAt string, response, request string) error {
	_, err := s.db.Exec(`INSERT INTO request_logs (status, method, error, endpoint, created_at, response, request) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		status, method, errStr, endpoint, createdAt, response, request)
	return err
}
