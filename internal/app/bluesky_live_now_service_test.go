package app

import (
	"context"
	"testing"

	"StreamSignal/internal/domain"
)

type blueskyLiveNowManagerStub struct {
	setCalls []struct {
		accountIdentifier string
		credentialKey     string
		status            domain.BlueskyLiveNowStatus
	}
	clearCalls []struct {
		accountIdentifier string
		credentialKey     string
	}
	setErr   error
	clearErr error
}

func (s *blueskyLiveNowManagerStub) SetLiveNow(_ context.Context, accountIdentifier string, credentialKey string, status domain.BlueskyLiveNowStatus) error {
	s.setCalls = append(s.setCalls, struct {
		accountIdentifier string
		credentialKey     string
		status            domain.BlueskyLiveNowStatus
	}{accountIdentifier, credentialKey, status})
	return s.setErr
}

func (s *blueskyLiveNowManagerStub) ClearLiveNow(_ context.Context, accountIdentifier string, credentialKey string) error {
	s.clearCalls = append(s.clearCalls, struct {
		accountIdentifier string
		credentialKey     string
	}{accountIdentifier, credentialKey})
	return s.clearErr
}

func TestBlueskyLiveNowServiceSetValidatesAndNormalizes(t *testing.T) {
	manager := &blueskyLiveNowManagerStub{}
	service := NewBlueskyLiveNowService(manager)

	err := service.Set(context.Background(), "streamer.test", "app-password", domain.BlueskyLiveNowStatus{
		URL:             " https://twitch.tv/example-streamer ",
		Title:           " Stream Signal Live ",
		Description:     " Launch stream ",
		DurationMinutes: 90,
	})
	if err != nil {
		t.Fatalf("set live now: %v", err)
	}
	if len(manager.setCalls) != 1 {
		t.Fatalf("expected one set call, got %d", len(manager.setCalls))
	}
	call := manager.setCalls[0]
	if call.accountIdentifier != "streamer.test" || call.credentialKey != "app-password" {
		t.Fatalf("unexpected credentials: %+v", call)
	}
	if call.status.URL != "https://twitch.tv/example-streamer" || call.status.Title != "Stream Signal Live" || call.status.Description != "Launch stream" || call.status.DurationMinutes != 90 {
		t.Fatalf("unexpected status normalization: %+v", call.status)
	}
}

func TestBlueskyLiveNowServiceSetRejectsInvalidURL(t *testing.T) {
	service := NewBlueskyLiveNowService(&blueskyLiveNowManagerStub{})

	err := service.Set(context.Background(), "streamer.test", "app-password", domain.BlueskyLiveNowStatus{
		URL:   "not-a-url",
		Title: "Live",
	})
	if err == nil || err.Error() != "Bluesky Live Now URL must be a valid absolute URL" {
		t.Fatalf("expected invalid URL error, got %v", err)
	}
}

func TestBlueskyLiveNowServiceClearValidatesCredentials(t *testing.T) {
	service := NewBlueskyLiveNowService(&blueskyLiveNowManagerStub{})

	err := service.Clear(context.Background(), "", "app-password")
	if err == nil || err.Error() != "Bluesky account identifier is required" {
		t.Fatalf("expected account identifier error, got %v", err)
	}
}
