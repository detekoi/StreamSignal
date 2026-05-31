package app

import (
	"context"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/ports"
)

type LogService struct {
	repository ports.LogRepository
	clock      Clock
}

func NewLogService(repository ports.LogRepository) *LogService {
	return &LogService{
		repository: repository,
		clock:      systemClock{},
	}
}

func (s *LogService) Append(ctx context.Context, destination, action, status, message string) error {
	return s.repository.Append(ctx, domain.LogEntry{
		Timestamp:   s.clock.Now(),
		Destination: destination,
		Action:      action,
		Status:      status,
		Message:     message,
	})
}

func (s *LogService) ListRecent(ctx context.Context, limit int) ([]domain.LogEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repository.ListRecent(ctx, limit)
}
