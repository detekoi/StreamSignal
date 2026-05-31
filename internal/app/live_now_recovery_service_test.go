package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"StreamSignal/internal/domain"
)

type liveNowSessionRepositoryStub struct {
	items     []domain.ActiveLiveNowSession
	upsertErr error
	listErr   error
	deleteErr error
	upserts   []domain.ActiveLiveNowSession
	deletes   []string
}

func (s *liveNowSessionRepositoryStub) Upsert(_ context.Context, session domain.ActiveLiveNowSession) error {
	s.upserts = append(s.upserts, session)
	return s.upsertErr
}

func (s *liveNowSessionRepositoryStub) List(context.Context) ([]domain.ActiveLiveNowSession, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return append([]domain.ActiveLiveNowSession(nil), s.items...), nil
}

func (s *liveNowSessionRepositoryStub) Delete(_ context.Context, destinationID string) error {
	s.deletes = append(s.deletes, destinationID)
	return s.deleteErr
}

func TestLiveNowRecoveryServiceListsPendingSessions(t *testing.T) {
	repository := &liveNowSessionRepositoryStub{
		items: []domain.ActiveLiveNowSession{{
			DestinationID:     "bluesky-main",
			DestinationName:   "Main Bluesky",
			Platform:          string(domain.PlatformBluesky),
			AccountIdentifier: "don.main",
			CredentialKey:     "bluesky/main",
			StreamURL:         "https://example.com/live",
			StreamTitle:       "Going Live",
			StartedAt:         time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC),
		}},
	}
	service := NewLiveNowRecoveryService(repository, &blueskyLiveNowManagerStub{})

	sessions, err := service.ListPending(context.Background())
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(sessions) != 1 || sessions[0].DestinationID != "bluesky-main" {
		t.Fatalf("unexpected sessions: %+v", sessions)
	}
}

func TestLiveNowRecoveryServiceClearsPendingSession(t *testing.T) {
	repository := &liveNowSessionRepositoryStub{
		items: []domain.ActiveLiveNowSession{{
			DestinationID:     "bluesky-main",
			DestinationName:   "Main Bluesky",
			Platform:          string(domain.PlatformBluesky),
			AccountIdentifier: "don.main",
			CredentialKey:     "bluesky/main",
			StreamURL:         "https://example.com/live",
			StreamTitle:       "Going Live",
			StartedAt:         time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC),
		}},
	}
	manager := &blueskyLiveNowManagerStub{}
	service := NewLiveNowRecoveryService(repository, manager)

	result, err := service.ClearPending(context.Background(), "bluesky-main")
	if err != nil {
		t.Fatalf("clear pending: %v", err)
	}
	if result.State != domain.ExecutionStateSuccess {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(manager.clearCalls) != 1 || manager.clearCalls[0].accountIdentifier != "don.main" {
		t.Fatalf("expected clear call, got %+v", manager.clearCalls)
	}
	if len(repository.deletes) != 1 || repository.deletes[0] != "bluesky-main" {
		t.Fatalf("expected delete call, got %+v", repository.deletes)
	}
}

func TestLiveNowRecoveryServiceReturnsFailureResultWhenClearFails(t *testing.T) {
	repository := &liveNowSessionRepositoryStub{
		items: []domain.ActiveLiveNowSession{{
			DestinationID:     "bluesky-main",
			DestinationName:   "Main Bluesky",
			Platform:          string(domain.PlatformBluesky),
			AccountIdentifier: "don.main",
			CredentialKey:     "bluesky/main",
			StartedAt:         time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC),
		}},
	}
	manager := &blueskyLiveNowManagerStub{clearErr: errors.New("clear denied")}
	service := NewLiveNowRecoveryService(repository, manager)

	result, err := service.ClearPending(context.Background(), "bluesky-main")
	if err != nil {
		t.Fatalf("clear pending: %v", err)
	}
	if result.State != domain.ExecutionStateFailed || result.Message != "Live Now recovery clear failed: clear denied" {
		t.Fatalf("unexpected failure result: %+v", result)
	}
}
