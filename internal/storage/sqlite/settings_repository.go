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
select test_mode_enabled, default_stream_url, default_hashtags, duplicate_protection_enabled,
       duplicate_window_minutes, end_stream_post_enabled, end_stream_template,
       test_discord_webhook_key, test_bluesky_account_identifier, test_bluesky_credential_key, test_mastodon_credential_key, test_mastodon_instance_url
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
values(1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
on conflict(id) do update set
  test_mode_enabled = excluded.test_mode_enabled,
  default_stream_url = excluded.default_stream_url,
  default_hashtags = excluded.default_hashtags,
  duplicate_protection_enabled = excluded.duplicate_protection_enabled,
  duplicate_window_minutes = excluded.duplicate_window_minutes,
  end_stream_post_enabled = excluded.end_stream_post_enabled,
  end_stream_template = excluded.end_stream_template,
  test_discord_webhook_key = excluded.test_discord_webhook_key,
  test_bluesky_account_identifier = excluded.test_bluesky_account_identifier,
  test_bluesky_credential_key = excluded.test_bluesky_credential_key,
  test_mastodon_credential_key = excluded.test_mastodon_credential_key,
  test_mastodon_instance_url = excluded.test_mastodon_instance_url`,
		boolToInt(settings.TestModeEnabled),
		settings.DefaultStreamURL,
		settings.DefaultHashtags,
		boolToInt(settings.DuplicateProtectionEnabled),
		settings.DuplicateWindowMinutes,
		boolToInt(settings.EndStreamPostEnabled),
		settings.EndStreamTemplate,
		settings.TestDiscordWebhookKey,
		settings.TestBlueskyAccountIdentifier,
		settings.TestBlueskyCredentialKey,
		settings.TestMastodonCredentialKey,
		settings.TestMastodonInstanceURL,
	)
	if err != nil {
		return fmt.Errorf("save settings: %w", err)
	}

	return nil
}

func scanSettings(scanner rowScanner) (domain.AppSettings, error) {
	var settings domain.AppSettings
	var testModeEnabled int
	var duplicateProtectionEnabled int
	var endStreamPostEnabled int

	err := scanner.Scan(
		&testModeEnabled,
		&settings.DefaultStreamURL,
		&settings.DefaultHashtags,
		&duplicateProtectionEnabled,
		&settings.DuplicateWindowMinutes,
		&endStreamPostEnabled,
		&settings.EndStreamTemplate,
		&settings.TestDiscordWebhookKey,
		&settings.TestBlueskyAccountIdentifier,
		&settings.TestBlueskyCredentialKey,
		&settings.TestMastodonCredentialKey,
		&settings.TestMastodonInstanceURL,
	)
	if err != nil {
		return domain.AppSettings{}, err
	}

	settings.TestModeEnabled = intToBool(testModeEnabled)
	settings.DuplicateProtectionEnabled = intToBool(duplicateProtectionEnabled)
	settings.EndStreamPostEnabled = intToBool(endStreamPostEnabled)

	return settings, nil
}
