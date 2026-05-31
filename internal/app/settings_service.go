package app

import (
	"context"
	"fmt"
	"strings"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/ports"
)

type SettingsService struct {
	repository ports.SettingsRepository
}

func NewSettingsService(repository ports.SettingsRepository) *SettingsService {
	return &SettingsService{repository: repository}
}

func (s *SettingsService) Load(ctx context.Context) (domain.AppSettings, error) {
	return s.repository.Load(ctx)
}

func (s *SettingsService) Save(ctx context.Context, settings domain.AppSettings) (domain.AppSettings, error) {
	if settings.DuplicateWindowMinutes <= 0 {
		return domain.AppSettings{}, fmt.Errorf("duplicate window minutes must be greater than zero")
	}
	if settings.EndStreamPostEnabled && settings.EndStreamTemplate == "" {
		return domain.AppSettings{}, fmt.Errorf("end stream template is required when end stream posting is enabled")
	}
	if settings.TestModeEnabled {
		switch {
		case strings.TrimSpace(settings.TestDiscordWebhookKey) == "":
			return domain.AppSettings{}, fmt.Errorf("test Discord webhook key is required when test mode is enabled")
		case strings.TrimSpace(settings.TestBlueskyAccountIdentifier) == "":
			return domain.AppSettings{}, fmt.Errorf("test Bluesky account identifier is required when test mode is enabled")
		case strings.TrimSpace(settings.TestBlueskyCredentialKey) == "":
			return domain.AppSettings{}, fmt.Errorf("test Bluesky credential key is required when test mode is enabled")
		case strings.TrimSpace(settings.TestMastodonCredentialKey) == "":
			return domain.AppSettings{}, fmt.Errorf("test Mastodon credential key is required when test mode is enabled")
		case strings.TrimSpace(settings.TestMastodonInstanceURL) == "":
			return domain.AppSettings{}, fmt.Errorf("test Mastodon instance URL is required when test mode is enabled")
		}
	}
	if err := s.repository.Save(ctx, settings); err != nil {
		return domain.AppSettings{}, err
	}
	return settings, nil
}
