package app

import (
	"context"
	"errors"
	"testing"

	"StreamSignal/internal/domain"
)

type discordCredentialVerifierStub struct {
	webhook string
	err     error
}

func (s *discordCredentialVerifierStub) VerifyWebhook(_ context.Context, webhookKey string) error {
	s.webhook = webhookKey
	return s.err
}

type blueskyCredentialVerifierStub struct {
	accountIdentifier string
	credentialKey     string
	err               error
}

func (s *blueskyCredentialVerifierStub) VerifyCredentials(_ context.Context, accountIdentifier string, credentialKey string) error {
	s.accountIdentifier = accountIdentifier
	s.credentialKey = credentialKey
	return s.err
}

type mastodonCredentialVerifierStub struct {
	credentialKey string
	instanceURL   string
	err           error
}

func (s *mastodonCredentialVerifierStub) VerifyCredentials(_ context.Context, credentialKey string, instanceURL string) error {
	s.credentialKey = credentialKey
	s.instanceURL = instanceURL
	return s.err
}

func TestCredentialSetupServiceTestDestinationSuccess(t *testing.T) {
	discord := &discordCredentialVerifierStub{}
	bluesky := &blueskyCredentialVerifierStub{}
	mastodon := &mastodonCredentialVerifierStub{}
	service := NewCredentialSetupService(discord, bluesky, mastodon)

	result := service.TestDestination(context.Background(), domain.Destination{
		Platform:   domain.PlatformDiscord,
		ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main"}`,
	})

	if result.State != "SUCCESS" {
		t.Fatalf("expected success, got %+v", result)
	}
	if discord.webhook == "" {
		t.Fatal("expected Discord verifier to receive webhook")
	}
}

func TestCredentialSetupServiceTestDestinationValidationError(t *testing.T) {
	service := NewCredentialSetupService(&discordCredentialVerifierStub{}, &blueskyCredentialVerifierStub{}, &mastodonCredentialVerifierStub{})

	result := service.TestDestination(context.Background(), domain.Destination{
		Platform:   domain.PlatformBluesky,
		ConfigJSON: `{"credentialKey":"bluesky/main"}`,
	})

	if result.State != "VALIDATION_ERROR" {
		t.Fatalf("expected validation error, got %+v", result)
	}
}

func TestCredentialSetupServiceTestDestinationFailure(t *testing.T) {
	service := NewCredentialSetupService(
		&discordCredentialVerifierStub{},
		&blueskyCredentialVerifierStub{err: errors.New("Bluesky createSession request failed: 401 Unauthorized")},
		&mastodonCredentialVerifierStub{},
	)

	result := service.TestDestination(context.Background(), domain.Destination{
		Platform:   domain.PlatformBluesky,
		ConfigJSON: `{"accountIdentifier":"don.test","credentialKey":"bad-password"}`,
	})

	if result.State != "FAILED" {
		t.Fatalf("expected failed state, got %+v", result)
	}
	if result.Message == "" {
		t.Fatal("expected failure message")
	}
}
