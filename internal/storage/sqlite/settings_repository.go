package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"StreamSignal/internal/domain"
)

type SettingsRepository struct {
	db *sql.DB
}

func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

func (r *SettingsRepository) Load(ctx context.Context) (domain.AppSettings, error) {
	row := r.db.QueryRowContext(ctx, `
select default_stream_url, default_hashtags, duplicate_protection_enabled, duplicate_window_minutes
from settings
where id = 1`)

	settings, err := scanSettings(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.DefaultAppSettings(), nil
		}
		return domain.AppSettings{}, fmt.Errorf("load settings: %w", err)
	}

	return settings, nil
}

func (r *SettingsRepository) Save(ctx context.Context, settings domain.AppSettings) error {
	_, err := r.db.ExecContext(ctx, `
insert into settings(
  id, test_mode_enabled, default_stream_url, default_hashtags,
  duplicate_protection_enabled, duplicate_window_minutes, end_stream_post_enabled, end_stream_template,
  test_discord_webhook_key, test_bluesky_account_identifier, test_bluesky_credential_key, test_mastodon_credential_key, test_mastodon_instance_url
)
values(1, 0, ?, ?, ?, ?, 0, '', '', '', '', '', '')
on conflict(id) do update set
  default_stream_url = excluded.default_stream_url,
  default_hashtags = excluded.default_hashtags,
  duplicate_protection_enabled = excluded.duplicate_protection_enabled,
  duplicate_window_minutes = excluded.duplicate_window_minutes`,
		settings.DefaultStreamURL,
		settings.DefaultHashtags,
		boolToInt(settings.DuplicateProtectionEnabled),
		settings.DuplicateWindowMinutes,
	)
	if err != nil {
		return fmt.Errorf("save settings: %w", err)
	}

	return nil
}

func scanSettings(scanner rowScanner) (domain.AppSettings, error) {
	var settings domain.AppSettings
	var duplicateProtectionEnabled int

	err := scanner.Scan(
		&settings.DefaultStreamURL,
		&settings.DefaultHashtags,
		&duplicateProtectionEnabled,
		&settings.DuplicateWindowMinutes,
	)
	if err != nil {
		return domain.AppSettings{}, err
	}

	settings.DuplicateProtectionEnabled = intToBool(duplicateProtectionEnabled)

	return settings, nil
}
