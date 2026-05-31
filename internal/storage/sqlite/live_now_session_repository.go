package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"StreamSignal/internal/domain"
)

type LiveNowSessionRepository struct {
	db *sql.DB
}

func NewLiveNowSessionRepository(db *sql.DB) *LiveNowSessionRepository {
	return &LiveNowSessionRepository{db: db}
}

func (r *LiveNowSessionRepository) Upsert(ctx context.Context, session domain.ActiveLiveNowSession) error {
	_, err := r.db.ExecContext(ctx, `
insert into bluesky_live_now_sessions(
  destination_id, destination_name, platform, account_identifier, credential_key, stream_url, stream_title, started_at
)
values(?, ?, ?, ?, ?, ?, ?, ?)
on conflict(destination_id) do update set
  destination_name = excluded.destination_name,
  platform = excluded.platform,
  account_identifier = excluded.account_identifier,
  credential_key = excluded.credential_key,
  stream_url = excluded.stream_url,
  stream_title = excluded.stream_title,
  started_at = excluded.started_at`,
		session.DestinationID,
		session.DestinationName,
		session.Platform,
		session.AccountIdentifier,
		session.CredentialKey,
		session.StreamURL,
		session.StreamTitle,
		session.StartedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("upsert live now session for %q: %w", session.DestinationID, err)
	}
	return nil
}

func (r *LiveNowSessionRepository) List(ctx context.Context) ([]domain.ActiveLiveNowSession, error) {
	rows, err := r.db.QueryContext(ctx, `
select destination_id, destination_name, platform, account_identifier, credential_key, stream_url, stream_title, started_at
from bluesky_live_now_sessions
order by destination_name asc`)
	if err != nil {
		return nil, fmt.Errorf("list live now sessions: %w", err)
	}
	defer rows.Close()

	var sessions []domain.ActiveLiveNowSession
	for rows.Next() {
		var session domain.ActiveLiveNowSession
		var startedAt string
		if err := rows.Scan(
			&session.DestinationID,
			&session.DestinationName,
			&session.Platform,
			&session.AccountIdentifier,
			&session.CredentialKey,
			&session.StreamURL,
			&session.StreamTitle,
			&startedAt,
		); err != nil {
			return nil, fmt.Errorf("scan live now session: %w", err)
		}
		session.StartedAt, err = time.Parse(time.RFC3339Nano, startedAt)
		if err != nil {
			return nil, fmt.Errorf("parse live now session started_at: %w", err)
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate live now sessions: %w", err)
	}
	return sessions, nil
}

func (r *LiveNowSessionRepository) Delete(ctx context.Context, destinationID string) error {
	_, err := r.db.ExecContext(ctx, `delete from bluesky_live_now_sessions where destination_id = ?`, destinationID)
	if err != nil {
		return fmt.Errorf("delete live now session for %q: %w", destinationID, err)
	}
	return nil
}
