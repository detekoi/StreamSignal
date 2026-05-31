package app

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"StreamSignal/internal/domain"
)

func validateDestinationConfig(destination domain.Destination) error {
	switch destination.Platform {
	case domain.PlatformDiscord:
		var config discordDestinationConfig
		if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
			return fmt.Errorf("invalid Discord config")
		}
		if strings.TrimSpace(config.WebhookKey) == "" {
			return fmt.Errorf("Discord webhook key is required")
		}
		if _, err := url.ParseRequestURI(strings.TrimSpace(config.WebhookKey)); err != nil {
			return fmt.Errorf("Discord webhook key must be a valid absolute URL")
		}
		return nil
	case domain.PlatformBluesky:
		var config blueskyDestinationConfig
		if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
			return fmt.Errorf("invalid Bluesky config")
		}
		if strings.TrimSpace(config.AccountIdentifier) == "" {
			return fmt.Errorf("Bluesky account identifier is required")
		}
		if strings.TrimSpace(config.CredentialKey) == "" {
			return fmt.Errorf("Bluesky credential key is required")
		}
		return nil
	case domain.PlatformMastodon:
		var config mastodonDestinationConfig
		if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
			return fmt.Errorf("invalid Mastodon config")
		}
		if strings.TrimSpace(config.CredentialKey) == "" {
			return fmt.Errorf("Mastodon credential key is required")
		}
		if strings.TrimSpace(config.InstanceURL) == "" {
			return fmt.Errorf("Mastodon instance URL is required")
		}
		if _, err := url.ParseRequestURI(strings.TrimSpace(config.InstanceURL)); err != nil {
			return fmt.Errorf("Mastodon instance URL must be a valid absolute URL")
		}
		return nil
	default:
		return fmt.Errorf("unsupported destination platform")
	}
}
