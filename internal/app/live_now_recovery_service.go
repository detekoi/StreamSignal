package app

import (
	"context"
	"fmt"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/ports"
)

type LiveNowRecoveryService struct {
	destinations ports.DestinationRepository
	sessions     ports.LiveNowSessionRepository
	liveNow      *BlueskyLiveNowService
}

func NewLiveNowRecoveryService(destinations ports.DestinationRepository, sessions ports.LiveNowSessionRepository, manager ports.BlueskyLiveNowManager) *LiveNowRecoveryService {
	return &LiveNowRecoveryService{
		destinations: destinations,
		sessions:     sessions,
		liveNow:      NewBlueskyLiveNowService(manager),
	}
}

func (s *LiveNowRecoveryService) ListPending(ctx context.Context) ([]domain.ActiveLiveNowSession, error) {
	return s.validPendingSessions(ctx)
}

func (s *LiveNowRecoveryService) ClearPending(ctx context.Context, destinationID string) (domain.ExecutionResult, error) {
	sessions, err := s.validPendingSessions(ctx)
	if err != nil {
		return domain.ExecutionResult{}, err
	}

	for _, session := range sessions {
		if session.DestinationID != destinationID {
			continue
		}
		if err := s.liveNow.Clear(ctx, session.AccountIdentifier, session.CredentialKey); err != nil {
			return domain.ExecutionResult{
				DestinationID:   session.DestinationID,
				DestinationName: session.DestinationName,
				Platform:        domain.PlatformBluesky,
				State:           domain.ExecutionStateFailed,
				Message:         fmt.Sprintf("Live Now recovery clear failed: %s", err.Error()),
			}, nil
		}
		if err := s.sessions.Delete(ctx, session.DestinationID); err != nil {
			return domain.ExecutionResult{}, err
		}
		return domain.ExecutionResult{
			DestinationID:   session.DestinationID,
			DestinationName: session.DestinationName,
			Platform:        domain.PlatformBluesky,
			State:           domain.ExecutionStateSuccess,
			Message:         "Recovered and cleared pending Live Now session.",
		}, nil
	}

	return domain.ExecutionResult{}, fmt.Errorf("no pending Live Now session found for destination %q", destinationID)
}

func (s *LiveNowRecoveryService) validPendingSessions(ctx context.Context) ([]domain.ActiveLiveNowSession, error) {
	destinations, err := s.destinations.List(ctx)
	if err != nil {
		return nil, err
	}
	destinationByID := make(map[string]domain.Destination, len(destinations))
	for _, destination := range destinations {
		destinationByID[destination.ID] = destination
	}

	sessions, err := s.sessions.List(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]domain.ActiveLiveNowSession, 0, len(sessions))
	for _, session := range sessions {
		destination, ok := destinationByID[session.DestinationID]
		if !ok || !trackedSessionMatchesDestination(destination, session) {
			if err := s.sessions.Delete(ctx, session.DestinationID); err != nil {
				return nil, err
			}
			continue
		}
		filtered = append(filtered, session)
	}

	return filtered, nil
}
