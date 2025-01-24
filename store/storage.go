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
