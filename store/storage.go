package store

import (
	"database/sql"
	"time"

	"github.com/johansundell/template-service/types"
)

type Storage struct {
	db *sql.DB
}

func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

type Store interface {
	Ping() error
	GetLogs(from, to time.Time) ([]types.UsageLog, error)
	LogRequest(status int, method, errStr, endpoint string, createdAt string, response, request string) error
}

func (s *Storage) Ping() error {
	return s.db.Ping()
}

func (s *Storage) LogRequest(status int, method, errStr, endpoint string, createdAt string, response, request string) error {
	_, err := s.db.Exec(`INSERT INTO request_logs (status, method, error, endpoint, created_at, response, request) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		status, method, errStr, endpoint, createdAt, response, request)
	return err
}

func (s *Storage) GetLogs(from, to time.Time) ([]types.UsageLog, error) {
	rows, err := s.db.Query(`SELECT id, status, method, error, endpoint, created_at, response, request FROM request_logs WHERE created_at BETWEEN ? AND ?`, from.Format(time.RFC3339), to.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []types.UsageLog
	for rows.Next() {
		var l types.UsageLog
		if err := rows.Scan(&l.ID, &l.Status, &l.Method, &l.Error, &l.Endpoint, &l.CreatedAt, &l.Response, &l.Request); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}
