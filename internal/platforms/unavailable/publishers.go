package unavailable

import (
	"context"

	"StreamSignal/internal/domain"
)

type DiscordPublisher struct{}

func (DiscordPublisher) Publish(context.Context, string, string, domain.DiscordPostMetadata) error {
	return &domain.IntegrationUnavailableError{
		Platform: domain.PlatformDiscord,
		Message:  "Discord integration is not connected yet",
	}
}

type BlueskyPublisher struct{}

func (BlueskyPublisher) PublishPost(context.Context, string, string, string, domain.BlueskyPostMetadata) error {
	return &domain.IntegrationUnavailableError{
		Platform: domain.PlatformBluesky,
		Message:  "Bluesky integration is not connected yet",
	}
}

type MastodonPublisher struct{}

func (MastodonPublisher) PublishPost(context.Context, string, string, string, domain.MastodonPostMetadata) error {
	return &domain.IntegrationUnavailableError{
		Platform: domain.PlatformMastodon,
		Message:  "Mastodon integration is not connected yet",
	}
}
