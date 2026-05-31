package app

import (
	"context"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/ports"
	"StreamSignal/internal/templates"
)

type PreviewService struct {
	destinations ports.DestinationRepository
	settings     ports.SettingsRepository
	clock        Clock
}

func NewPreviewService(destinations ports.DestinationRepository, settings ports.SettingsRepository) *PreviewService {
	return &PreviewService{
		destinations: destinations,
		settings:     settings,
		clock:        systemClock{},
	}
}

func (s *PreviewService) Generate(ctx context.Context, announcement domain.Announcement) ([]domain.PreviewItem, error) {
	settings, err := s.settings.Load(ctx)
	if err != nil {
		return nil, err
	}

	destinations, err := s.destinations.List(ctx)
	if err != nil {
		return nil, err
	}
	destinations = destinationsForSelection(destinations, announcement.DestinationIDs)

	normalized := domain.NormalizeAnnouncement(announcement, settings)
	announcementNotes := domain.ValidateAnnouncement(normalized)
	now := s.clock.Now()

	items := make([]domain.PreviewItem, 0, len(destinations))
	for _, destination := range destinations {
		content := templates.Render(destination.Template, normalized, destination.Platform, now)
		notes := append([]string{}, announcementNotes...)
		notes = append(notes, domain.ValidatePreviewContent(destination.Platform, content)...)

		items = append(items, domain.PreviewItem{
			DestinationID:   destination.ID,
			DestinationName: destination.Name,
			Platform:        destination.Platform,
			Content:         content,
			CharacterCount:  len(content),
			ValidationState: domain.PreviewValidationState(notes),
			ValidationNotes: notes,
		})
	}

	return items, nil
}
