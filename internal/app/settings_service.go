package app

import (
	"context"
	"fmt"

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
	if err := s.repository.Save(ctx, settings); err != nil {
		return domain.AppSettings{}, err
	}
	return settings, nil
}
