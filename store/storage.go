package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/johansundell/template-service/types"
)

// Store is the storage contract handlers and middleware depend on. Each
// backend owns its connection; the caller must Close it.
type Store interface {
	Ping(ctx context.Context) error
	// GetLogs returns the entries with from <= CreatedAt < to, oldest first.
	GetLogs(ctx context.Context, from, to time.Time) ([]types.UsageLog, error)
	LogRequest(ctx context.Context, entry types.UsageLog) error
	Close() error
}

// SQLStore keeps request logs in a SQL database (SQLite or MySQL).
type SQLStore struct {
	db *sql.DB
	// timeArg converts a timestamp into the driver argument for created_at.
	timeArg func(time.Time) any
}

func (s *SQLStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *SQLStore) Close() error {
	return s.db.Close()
}

func (s *SQLStore) LogRequest(ctx context.Context, entry types.UsageLog) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO request_logs (status, method, error, endpoint, created_at, response, request) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entry.Status, entry.Method, entry.Error, entry.Endpoint, s.timeArg(entry.CreatedAt), string(entry.Response), string(entry.Request))
	return err
}

func (s *SQLStore) GetLogs(ctx context.Context, from, to time.Time) ([]types.UsageLog, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, status, method, error, endpoint, created_at, response, request FROM request_logs WHERE created_at >= ? AND created_at < ? ORDER BY created_at, id`, s.timeArg(from), s.timeArg(to))
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}
