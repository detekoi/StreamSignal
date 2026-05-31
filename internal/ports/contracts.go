package ports

import (
	"context"
	"time"

	"StreamSignal/internal/domain"
)

type DestinationRepository interface {
	List(ctx context.Context) ([]domain.Destination, error)
	Get(ctx context.Context, id string) (domain.Destination, error)
	Save(ctx context.Context, destination domain.Destination) error
	Delete(ctx context.Context, id string) error
}

type SettingsRepository interface {
	Load(ctx context.Context) (domain.AppSettings, error)
	Save(ctx context.Context, settings domain.AppSettings) error
}

type LogRepository interface {
	Append(ctx context.Context, entry domain.LogEntry) error
	ListRecent(ctx context.Context, limit int) ([]domain.LogEntry, error)
}

type PostHistoryRecord struct {
	DestinationID string
	ContentHash   string
	PostedAt      time.Time
}

type PostHistoryRepository interface {
	Record(ctx context.Context, destinationID string, renderedContent string, hash string, postedAt time.Time) error
	FindRecentByDestination(ctx context.Context, destinationID string, since time.Time) ([]PostHistoryRecord, error)
}

type LiveNowSessionRepository interface {
	Upsert(ctx context.Context, session domain.ActiveLiveNowSession) error
	List(ctx context.Context) ([]domain.ActiveLiveNowSession, error)
	Delete(ctx context.Context, destinationID string) error
}

type SecretStore interface {
	Get(ctx context.Context, key string) (string, error)
	Put(ctx context.Context, key string, value string) error
	Delete(ctx context.Context, key string) error
}

type DiscordPublisher interface {
	Publish(ctx context.Context, webhookKey string, content string) error
}

type DiscordCredentialVerifier interface {
	VerifyWebhook(ctx context.Context, webhookKey string) error
}

type BlueskyPublisher interface {
	PublishPost(ctx context.Context, accountIdentifier string, credentialKey string, content string) error
}

type BlueskyCredentialVerifier interface {
	VerifyCredentials(ctx context.Context, accountIdentifier string, credentialKey string) error
}

type BlueskyLiveNowManager interface {
	SetLiveNow(ctx context.Context, accountIdentifier string, credentialKey string, status domain.BlueskyLiveNowStatus) error
	ClearLiveNow(ctx context.Context, accountIdentifier string, credentialKey string) error
}

type MastodonPublisher interface {
	PublishPost(ctx context.Context, credentialKey string, instanceURL string, content string) error
}

type MastodonCredentialVerifier interface {
	VerifyCredentials(ctx context.Context, credentialKey string, instanceURL string) error
}
