package app

import (
	"context"
	"fmt"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/ports"
)

type CredentialSetupService struct {
	discord  ports.DiscordCredentialVerifier
	bluesky  ports.BlueskyCredentialVerifier
	mastodon ports.MastodonCredentialVerifier
}

func NewCredentialSetupService(
	discord ports.DiscordCredentialVerifier,
	bluesky ports.BlueskyCredentialVerifier,
	mastodon ports.MastodonCredentialVerifier,
) *CredentialSetupService {
	return &CredentialSetupService{
		discord:  discord,
		bluesky:  bluesky,
		mastodon: mastodon,
	}
}

func (s *CredentialSetupService) TestDestination(ctx context.Context, destination domain.Destination) domain.CredentialCheckResult {
	result := domain.CredentialCheckResult{
		Platform: string(destination.Platform),
		State:    "FAILED",
		Message:  "Unable to verify credentials.",
	}

	if err := validateDestinationCredentialConfig(destination); err != nil {
		result.State = "VALIDATION_ERROR"
		result.Message = err.Error()
		return result
	}

	switch destination.Platform {
	case domain.PlatformDiscord:
		config, err := parseDiscordDestinationConfig(destination.ConfigJSON)
		if err != nil {
			result.State = "VALIDATION_ERROR"
			result.Message = err.Error()
			return result
		}
		if err := s.discord.VerifyWebhook(ctx, config.WebhookKey); err != nil {
			result.Message = err.Error()
			return result
		}
		result.State = "SUCCESS"
		result.Message = "Discord webhook verified successfully."
		return result
	case domain.PlatformBluesky:
		config, err := parseBlueskyDestinationConfig(destination.ConfigJSON)
		if err != nil {
			result.State = "VALIDATION_ERROR"
			result.Message = err.Error()
			return result
		}
		if err := s.bluesky.VerifyCredentials(ctx, config.AccountIdentifier, config.CredentialKey); err != nil {
			result.Message = err.Error()
			return result
		}
		result.State = "SUCCESS"
		result.Message = "Bluesky credentials verified successfully."
		return result
	case domain.PlatformMastodon:
		config, err := parseMastodonDestinationConfig(destination.ConfigJSON)
		if err != nil {
			result.State = "VALIDATION_ERROR"
			result.Message = err.Error()
			return result
		}
		if err := s.mastodon.VerifyCredentials(ctx, config.CredentialKey, config.InstanceURL); err != nil {
			result.Message = err.Error()
			return result
		}
		result.State = "SUCCESS"
		result.Message = "Mastodon credentials verified successfully."
		return result
	default:
		result.State = "VALIDATION_ERROR"
		result.Message = fmt.Sprintf("unsupported destination platform %q", destination.Platform)
		return result
	}
}
