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
	sessions     ports.LiveNowSessionRepository
	clock        Clock
}

func NewPreviewService(destinations ports.DestinationRepository, settings ports.SettingsRepository, sessions ports.LiveNowSessionRepository) *PreviewService {
	return &PreviewService{
		destinations: destinations,
		settings:     settings,
		sessions:     sessions,
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
	now := s.clock.Now()

	items := make([]domain.PreviewItem, 0, len(destinations)*2)
	for _, destination := range destinations {
		content := templates.Render(destination.Template, normalized, destination.Platform, now)
		notes := domain.ValidateAnnouncementForTemplate(normalized, destination.Template)
		notes = append(notes, domain.ValidatePreviewContent(destination.Platform, content)...)
		if err := validateDestinationConfig(destination); err != nil {
			notes = append(notes, err.Error())
		}

		items = append(items, domain.PreviewItem{
			DestinationID:   destination.ID,
			DestinationName: destination.Name,
			Platform:        destination.Platform,
			PreviewLabel:    "Go Live",
			Content:         content,
			CharacterCount:  len(content),
			ValidationState: domain.PreviewValidationState(notes),
			ValidationNotes: notes,
		})
	}

	endStreamItems, err := s.generateEndStream(ctx, settings, destinations, false)
	if err != nil {
		return nil, err
	}
	items = append(items, endStreamItems...)

	return items, nil
}

func (s *PreviewService) GenerateEndStream(ctx context.Context, announcement domain.Announcement) ([]domain.PreviewItem, error) {
	settings, err := s.settings.Load(ctx)
	if err != nil {
		return nil, err
	}

	destinations, err := s.destinations.List(ctx)
	if err != nil {
		return nil, err
	}
	destinations = destinationsForSelection(destinations, announcement.DestinationIDs)

	return s.generateEndStream(ctx, settings, destinations, true)
}

func (s *PreviewService) generateEndStream(ctx context.Context, settings domain.AppSettings, destinations []domain.Destination, includeDisabled bool) ([]domain.PreviewItem, error) {
	sessions, err := s.sessions.List(ctx)
	if err != nil {
		return nil, err
	}
	sessionByDestination := make(map[string]domain.ActiveLiveNowSession, len(sessions))
	for _, session := range sessions {
		sessionByDestination[session.DestinationID] = session
	}

	now := s.clock.Now()
	items := make([]domain.PreviewItem, 0, len(destinations))
	for _, destination := range destinations {
		endConfig, err := endStreamConfigForDestination(destination)
		if err != nil {
			items = append(items, domain.PreviewItem{
				DestinationID:   destination.ID,
				DestinationName: destination.Name,
				Platform:        destination.Platform,
				PreviewLabel:    "End Stream",
				ValidationState: domain.PreviewValidationInvalid,
				ValidationNotes: []string{err.Error()},
			})
			continue
		}
		if !endConfig.Enabled {
			if !includeDisabled {
				continue
			}
			items = append(items, domain.PreviewItem{
				DestinationID:   destination.ID,
				DestinationName: destination.Name,
				Platform:        destination.Platform,
				PreviewLabel:    "End Stream",
				ValidationState: domain.PreviewValidationValid,
				ValidationNotes: []string{"End Stream message disabled for this destination."},
			})
			continue
		}

		announcementForDestination := domain.NormalizeAnnouncement(domain.Announcement{
			Message: endConfig.Template,
		}, settings)
		if session, ok := sessionByDestination[destination.ID]; ok {
			announcementForDestination.StreamTitle = session.StreamTitle
			announcementForDestination.StreamURL = session.StreamURL
		}

		content := templates.Render(endConfig.Template, announcementForDestination, destination.Platform, now)
		notes := domain.ValidateAnnouncementForTemplate(announcementForDestination, endConfig.Template)
		notes = append(notes, domain.ValidatePreviewContent(destination.Platform, content)...)
		if err := validateDestinationConfig(destination); err != nil {
			notes = append(notes, err.Error())
		}

		items = append(items, domain.PreviewItem{
			DestinationID:   destination.ID,
			DestinationName: destination.Name,
			Platform:        destination.Platform,
			PreviewLabel:    "End Stream",
			Content:         content,
			CharacterCount:  len(content),
			ValidationState: domain.PreviewValidationState(notes),
			ValidationNotes: notes,
		})
	}

	return items, nil
}
