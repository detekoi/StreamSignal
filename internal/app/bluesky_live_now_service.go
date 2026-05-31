package app

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/ports"
)

type BlueskyLiveNowService struct {
	manager ports.BlueskyLiveNowManager
}

func NewBlueskyLiveNowService(manager ports.BlueskyLiveNowManager) *BlueskyLiveNowService {
	return &BlueskyLiveNowService{manager: manager}
}

func (s *BlueskyLiveNowService) Set(ctx context.Context, accountIdentifier string, credentialKey string, status domain.BlueskyLiveNowStatus) error {
	if strings.TrimSpace(accountIdentifier) == "" {
		return fmt.Errorf("Bluesky account identifier is required")
	}
	if strings.TrimSpace(credentialKey) == "" {
		return fmt.Errorf("Bluesky credential key is required")
	}
	if err := validateBlueskyLiveNowStatus(status); err != nil {
		return err
	}
	return s.manager.SetLiveNow(ctx, accountIdentifier, credentialKey, normalizeBlueskyLiveNowStatus(status))
}

func (s *BlueskyLiveNowService) Clear(ctx context.Context, accountIdentifier string, credentialKey string) error {
	if strings.TrimSpace(accountIdentifier) == "" {
		return fmt.Errorf("Bluesky account identifier is required")
	}
	if strings.TrimSpace(credentialKey) == "" {
		return fmt.Errorf("Bluesky credential key is required")
	}
	return s.manager.ClearLiveNow(ctx, accountIdentifier, credentialKey)
}

func validateBlueskyLiveNowStatus(status domain.BlueskyLiveNowStatus) error {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(status.URL))
	if err != nil || parsed == nil || !parsed.IsAbs() {
		return fmt.Errorf("Bluesky Live Now URL must be a valid absolute URL")
	}
	if strings.TrimSpace(status.Title) == "" {
		return fmt.Errorf("Bluesky Live Now title is required")
	}
	if status.DurationMinutes < 0 {
		return fmt.Errorf("Bluesky Live Now duration cannot be negative")
	}
	return nil
}

func normalizeBlueskyLiveNowStatus(status domain.BlueskyLiveNowStatus) domain.BlueskyLiveNowStatus {
	return domain.BlueskyLiveNowStatus{
		URL:             strings.TrimSpace(status.URL),
		Title:           strings.TrimSpace(status.Title),
		Description:     strings.TrimSpace(status.Description),
		DurationMinutes: status.DurationMinutes,
	}
}
