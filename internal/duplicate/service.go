package duplicate

import (
	"context"
	"fmt"
	"time"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/ports"
)

type Warning struct {
	Message string `json:"message"`
}

type Service struct {
	history ports.PostHistoryRepository
}

func NewService(history ports.PostHistoryRepository) *Service {
	return &Service{history: history}
}

func (s *Service) Check(ctx context.Context, settings domain.AppSettings, candidates map[domain.Destination]string, now time.Time) (*Warning, error) {
	if !settings.DuplicateProtectionEnabled || len(candidates) == 0 {
		return nil, nil
	}

	windowStart := now.Add(-time.Duration(settings.DuplicateWindowMinutes) * time.Minute)
	for destination, content := range candidates {
		records, err := s.history.FindRecentByDestination(ctx, destination.ID, windowStart)
		if err != nil {
			return nil, err
		}
		hash := HashContent(content)
		for _, record := range records {
			if record.ContentHash == hash || !record.PostedAt.Before(windowStart) {
				return &Warning{
					Message: fmt.Sprintf("A similar announcement was recently posted within the last %d minutes. Continue?", settings.DuplicateWindowMinutes),
				}, nil
			}
		}
	}

	return nil, nil
}
