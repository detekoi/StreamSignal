package unavailable

import (
	"context"
	"testing"

	"StreamSignal/internal/domain"
)

func TestUnavailablePublishersReturnIntegrationUnavailableErrors(t *testing.T) {
	tests := []struct {
		name     string
		platform domain.DestinationPlatform
		call     func() error
	}{
		{
			name:     "discord",
			platform: domain.PlatformDiscord,
			call: func() error {
				return (DiscordPublisher{}).Publish(context.Background(), "ignored", "hello")
			},
		},
		{
			name:     "bluesky",
			platform: domain.PlatformBluesky,
			call: func() error {
				return (BlueskyPublisher{}).PublishPost(context.Background(), "account", "credential", "hello", domain.BlueskyPostMetadata{})
			},
		},
		{
			name:     "mastodon",
			platform: domain.PlatformMastodon,
			call: func() error {
				return (MastodonPublisher{}).PublishPost(context.Background(), "credential", "https://example.social", "hello")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatal("expected unavailable integration error")
			}

			unavailableErr, ok := err.(*domain.IntegrationUnavailableError)
			if !ok {
				t.Fatalf("expected IntegrationUnavailableError, got %T", err)
			}
			if unavailableErr.Platform != tt.platform {
				t.Fatalf("expected platform %q, got %+v", tt.platform, unavailableErr)
			}
			if !domain.IsIntegrationUnavailable(err) {
				t.Fatalf("expected integration unavailable helper to recognize %+v", err)
			}
		})
	}
}
