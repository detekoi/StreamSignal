package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"StreamSignal/internal/domain"
)

type LogRepository struct {
	db *sql.DB
}

func NewLogRepository(db *sql.DB) *LogRepository {
	return &LogRepository{db: db}
}

func (r *LogRepository) Append(ctx context.Context, entry domain.LogEntry) error {
	_, err := r.db.ExecContext(ctx, `
insert into logs(timestamp, destination, action, status, message)
values(?, ?, ?, ?, ?)`,
		formatTime(entry.Timestamp),
		entry.Destination,
		entry.Action,
		entry.Status,
		entry.Message,
	)
	if err != nil {
		return fmt.Errorf("append log entry: %w", err)
	}
	return nil
}

func (r *LogRepository) ListRecent(ctx context.Context, limit int) ([]domain.LogEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
select timestamp, destination, action, status, message
from logs
order by timestamp desc
limit ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list logs: %w", err)
	}
	defer rows.Close()

	entries := make([]domain.LogEntry, 0)
	for rows.Next() {
		var entry domain.LogEntry
		var timestamp string
		if err := rows.Scan(&timestamp, &entry.Destination, &entry.Action, &entry.Status, &entry.Message); err != nil {
			return nil, fmt.Errorf("scan log entry: %w", err)
		}

		parsedTime, err := parseTime(timestamp)
		if err != nil {
			return nil, fmt.Errorf("parse log timestamp: %w", err)
		}
		entry.Timestamp = parsedTime
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate log entries: %w", err)
	}

	return entries, nil
}
