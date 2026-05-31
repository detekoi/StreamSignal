package secure

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/secrets/memory"
	sqlitestore "StreamSignal/internal/storage/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sqlitestore.OpenInMemory()
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

func TestSecureDestinationRepositoryStoresOnlySecretReferencesInSQLite(t *testing.T) {
	db := openTestDB(t)
	repository := NewDestinationRepository(sqlitestore.NewDestinationRepository(db), memory.NewStore())
	now := time.Now().UTC().Round(time.Microsecond)

	expected := domain.Destination{
		ID:         "discord-main",
		Platform:   domain.PlatformDiscord,
		Name:       "Main Discord",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"serverName":"My Server","channelName":"go-live","webhookKey":"https://discord.example/webhook-secret"}`,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := repository.Save(context.Background(), expected); err != nil {
		t.Fatalf("save destination: %v", err)
	}

	var rawConfig string
	if err := db.QueryRow(`select config_json from destinations where id = ?`, expected.ID).Scan(&rawConfig); err != nil {
		t.Fatalf("query raw destination config: %v", err)
	}
	if strings.Contains(rawConfig, "webhook-secret") {
		t.Fatalf("expected SQLite to avoid storing raw secret, got %s", rawConfig)
	}
	if !strings.Contains(rawConfig, "secret://streamsignal/destinations/discord-main/discord-webhook") {
		t.Fatalf("expected secret reference in SQLite, got %s", rawConfig)
	}

	actual, err := repository.Get(context.Background(), expected.ID)
	if err != nil {
		t.Fatalf("get destination: %v", err)
	}
	if !strings.Contains(actual.ConfigJSON, "webhook-secret") {
		t.Fatalf("expected hydrated secret in loaded destination, got %s", actual.ConfigJSON)
	}
}

func TestSecureDestinationRepositoryMigratesLegacySQLiteSecretsOnRead(t *testing.T) {
	db := openTestDB(t)
	rawRepository := sqlitestore.NewDestinationRepository(db)
	repository := NewDestinationRepository(rawRepository, memory.NewStore())
	now := time.Now().UTC().Round(time.Microsecond)

	legacy := domain.Destination{
		ID:         "discord-main",
		Platform:   domain.PlatformDiscord,
		Name:       "Main Discord",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"webhookKey":"https://discord.example/legacy-secret"}`,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := rawRepository.Save(context.Background(), legacy); err != nil {
		t.Fatalf("seed raw legacy destination: %v", err)
	}

	loaded, err := repository.Get(context.Background(), legacy.ID)
	if err != nil {
		t.Fatalf("load migrated destination: %v", err)
	}
	if !strings.Contains(loaded.ConfigJSON, "legacy-secret") {
		t.Fatalf("expected hydrated legacy secret, got %s", loaded.ConfigJSON)
	}

	var rawConfig string
	if err := db.QueryRow(`select config_json from destinations where id = ?`, legacy.ID).Scan(&rawConfig); err != nil {
		t.Fatalf("query migrated raw destination config: %v", err)
	}
	if strings.Contains(rawConfig, "legacy-secret") {
		t.Fatalf("expected legacy secret to be migrated out of SQLite, got %s", rawConfig)
	}
}

func TestSecureDestinationRepositoryDeleteRemovesDestinationAndStoredSecrets(t *testing.T) {
	db := openTestDB(t)
	secretStore := memory.NewStore()
	repository := NewDestinationRepository(sqlitestore.NewDestinationRepository(db), secretStore)
	now := time.Now().UTC().Round(time.Microsecond)

	item := domain.Destination{
		ID:         "bluesky-main",
		Platform:   domain.PlatformBluesky,
		Name:       "Main Bluesky",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"accountIdentifier":"don.main","credentialKey":"bluesky-app-password"}`,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := repository.Save(context.Background(), item); err != nil {
		t.Fatalf("save destination: %v", err)
	}

	if err := repository.Delete(context.Background(), item.ID); err != nil {
		t.Fatalf("delete destination: %v", err)
	}

	list, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("list destinations: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected no destinations after delete, got %+v", list)
	}

	if _, err := secretStore.Get(context.Background(), destinationSecretKey(item.ID, "bluesky-credential")); err == nil {
		t.Fatalf("expected destination secret to be removed")
	}
}

func TestSecureSettingsRepositoryStoresOnlySecretReferencesInSQLite(t *testing.T) {
	db := openTestDB(t)
	repository := NewSettingsRepository(sqlitestore.NewSettingsRepository(db), memory.NewStore())

	expected := domain.AppSettings{
		TestModeEnabled:              true,
		TestDiscordWebhookKey:        "https://discord.example/test-webhook",
		TestBlueskyAccountIdentifier: "don.test",
		TestBlueskyCredentialKey:     "bluesky-app-password",
		TestMastodonCredentialKey:    "mastodon-access-token",
		TestMastodonInstanceURL:      "https://mastodon.test",
		DefaultStreamURL:             "https://example.com/live",
		DefaultHashtags:              "#streamsignal",
		DuplicateProtectionEnabled:   true,
		DuplicateWindowMinutes:       15,
		EndStreamPostEnabled:         true,
		EndStreamTemplate:            "Thanks for watching!",
	}

	if err := repository.Save(context.Background(), expected); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	var rawDiscord, rawBluesky, rawMastodon string
	if err := db.QueryRow(`
select test_discord_webhook_key, test_bluesky_credential_key, test_mastodon_credential_key
from settings where id = 1`).Scan(&rawDiscord, &rawBluesky, &rawMastodon); err != nil {
		t.Fatalf("query raw settings secrets: %v", err)
	}
	for _, raw := range []string{rawDiscord, rawBluesky, rawMastodon} {
		if strings.Contains(raw, "discord.example") || strings.Contains(raw, "app-password") || strings.Contains(raw, "access-token") {
			t.Fatalf("expected secret reference instead of raw secret, got %q", raw)
		}
		if !strings.HasPrefix(raw, "secret://streamsignal/settings/") {
			t.Fatalf("expected settings secret reference, got %q", raw)
		}
	}

	actual, err := repository.Load(context.Background())
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if actual.TestDiscordWebhookKey != expected.TestDiscordWebhookKey ||
		actual.TestBlueskyCredentialKey != expected.TestBlueskyCredentialKey ||
		actual.TestMastodonCredentialKey != expected.TestMastodonCredentialKey {
		t.Fatalf("expected hydrated secret settings %+v, got %+v", expected, actual)
	}
}

func TestSecureSettingsRepositoryMigratesLegacySQLiteSecretsOnLoad(t *testing.T) {
	db := openTestDB(t)
	rawRepository := sqlitestore.NewSettingsRepository(db)
	repository := NewSettingsRepository(rawRepository, memory.NewStore())

	legacy := domain.AppSettings{
		TestModeEnabled:              true,
		TestDiscordWebhookKey:        "https://discord.example/legacy-webhook",
		TestBlueskyAccountIdentifier: "don.test",
		TestBlueskyCredentialKey:     "legacy-bluesky-password",
		TestMastodonCredentialKey:    "legacy-mastodon-token",
		TestMastodonInstanceURL:      "https://mastodon.test",
		DefaultStreamURL:             "https://example.com/live",
		DefaultHashtags:              "#streamsignal",
		DuplicateProtectionEnabled:   true,
		DuplicateWindowMinutes:       15,
		EndStreamPostEnabled:         false,
		EndStreamTemplate:            "",
	}

	if err := rawRepository.Save(context.Background(), legacy); err != nil {
		t.Fatalf("seed raw legacy settings: %v", err)
	}

	loaded, err := repository.Load(context.Background())
	if err != nil {
		t.Fatalf("load migrated settings: %v", err)
	}
	if loaded.TestDiscordWebhookKey != legacy.TestDiscordWebhookKey ||
		loaded.TestBlueskyCredentialKey != legacy.TestBlueskyCredentialKey ||
		loaded.TestMastodonCredentialKey != legacy.TestMastodonCredentialKey {
		t.Fatalf("expected hydrated legacy settings, got %+v", loaded)
	}

	var rawDiscord, rawBluesky, rawMastodon string
	if err := db.QueryRow(`
select test_discord_webhook_key, test_bluesky_credential_key, test_mastodon_credential_key
from settings where id = 1`).Scan(&rawDiscord, &rawBluesky, &rawMastodon); err != nil {
		t.Fatalf("query migrated raw settings: %v", err)
	}
	if strings.Contains(rawDiscord, "legacy-webhook") || strings.Contains(rawBluesky, "legacy-bluesky-password") || strings.Contains(rawMastodon, "legacy-mastodon-token") {
		t.Fatalf("expected legacy settings secrets to be migrated out of SQLite")
	}
}

func TestSecureLiveNowSessionRepositoryStoresOnlySecretReferencesInSQLite(t *testing.T) {
	db := openTestDB(t)
	repository := NewLiveNowSessionRepository(sqlitestore.NewLiveNowSessionRepository(db), memory.NewStore())

	expected := domain.ActiveLiveNowSession{
		DestinationID:     "bluesky-main",
		DestinationName:   "Main Bluesky",
		Platform:          string(domain.PlatformBluesky),
		AccountIdentifier: "don.main",
		CredentialKey:     "bluesky-app-password",
		StreamURL:         "https://example.com/live",
		StreamTitle:       "Going Live",
		StartedAt:         time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC),
	}

	if err := repository.Upsert(context.Background(), expected); err != nil {
		t.Fatalf("upsert session: %v", err)
	}

	var rawCredential string
	if err := db.QueryRow(`select credential_key from bluesky_live_now_sessions where destination_id = ?`, expected.DestinationID).Scan(&rawCredential); err != nil {
		t.Fatalf("query raw session credential: %v", err)
	}
	if rawCredential == expected.CredentialKey {
		t.Fatalf("expected SQLite to avoid storing raw session credential")
	}
	if !strings.HasPrefix(rawCredential, "secret://streamsignal/live-now/") {
		t.Fatalf("expected session secret reference, got %q", rawCredential)
	}

	items, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(items) != 1 || items[0].CredentialKey != expected.CredentialKey {
		t.Fatalf("expected hydrated session credential, got %+v", items)
	}
}
