package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"StreamSignal/internal/ports"
)

type PostHistoryRepository struct {
	db *sql.DB
}

func NewPostHistoryRepository(db *sql.DB) *PostHistoryRepository {
	return &PostHistoryRepository{db: db}
}

func (r *PostHistoryRepository) Record(ctx context.Context, destinationID string, renderedContent string, hash string, postedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
insert into post_history(destination_id, rendered_content_hash, rendered_content, posted_at)
values(?, ?, ?, ?)`,
		destinationID,
		hash,
		renderedContent,
		formatTime(postedAt),
	)
	if err != nil {
		return fmt.Errorf("record post history for destination %q: %w", destinationID, err)
	}
	return nil
}

func (r *PostHistoryRepository) FindRecentByDestination(ctx context.Context, destinationID string, since time.Time) ([]ports.PostHistoryRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
select destination_id, rendered_content_hash, posted_at
from post_history
where destination_id = ? and posted_at >= ?
order by posted_at desc`, destinationID, formatTime(since))
	if err != nil {
		return nil, fmt.Errorf("find post history for destination %q: %w", destinationID, err)
	}
	defer rows.Close()

	var records []ports.PostHistoryRecord
	for rows.Next() {
		var record ports.PostHistoryRecord
		var postedAt string
		if err := rows.Scan(&record.DestinationID, &record.ContentHash, &postedAt); err != nil {
			return nil, fmt.Errorf("scan post history: %w", err)
		}
		parsedPostedAt, err := parseTime(postedAt)
		if err != nil {
			return nil, fmt.Errorf("parse post history timestamp: %w", err)
		}
		record.PostedAt = parsedPostedAt
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate post history: %w", err)
	}

	return records, nil
}
