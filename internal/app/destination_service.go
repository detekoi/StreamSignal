package app

import (
	"context"
	"fmt"
	"strings"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/ports"

	"github.com/google/uuid"
)

type DestinationService struct {
	repository ports.DestinationRepository
	clock      Clock
}

func NewDestinationService(repository ports.DestinationRepository) *DestinationService {
	return &DestinationService{
		repository: repository,
		clock:      systemClock{},
	}
}

func (s *DestinationService) List(ctx context.Context) ([]domain.Destination, error) {
	return s.repository.List(ctx)
}

func (s *DestinationService) Save(ctx context.Context, destination domain.Destination) (domain.Destination, error) {
	if err := validateDestination(destination); err != nil {
		return domain.Destination{}, err
	}

	now := s.clock.Now()
	if destination.ID == "" {
		destination.ID = uuid.NewString()
		destination.CreatedAt = now
	} else {
		existing, err := s.repository.Get(ctx, destination.ID)
		if err == nil {
			destination.CreatedAt = existing.CreatedAt
		} else if destination.CreatedAt.IsZero() {
			destination.CreatedAt = now
		}
	}

	if destination.CreatedAt.IsZero() {
		destination.CreatedAt = now
	}
	destination.UpdatedAt = now

	if err := s.repository.Save(ctx, destination); err != nil {
		return domain.Destination{}, err
	}

	return destination, nil
}

func (s *DestinationService) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("destination id is required")
	}
	return s.repository.Delete(ctx, id)
}

func validateDestination(destination domain.Destination) error {
	if strings.TrimSpace(string(destination.Platform)) == "" {
		return fmt.Errorf("destination platform is required")
	}
	if strings.TrimSpace(destination.Name) == "" {
		return fmt.Errorf("destination name is required")
	}
	if strings.TrimSpace(destination.Template) == "" {
		return fmt.Errorf("destination template is required")
	}
	if strings.TrimSpace(destination.ConfigJSON) == "" {
		return fmt.Errorf("destination config is required")
	}
	return validateDestinationConfig(destination)
}
