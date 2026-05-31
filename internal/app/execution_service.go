package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/duplicate"
	"StreamSignal/internal/ports"
	"StreamSignal/internal/templates"
)

type discordDestinationConfig struct {
	WebhookKey string `json:"webhookKey"`
}

type blueskyDestinationConfig struct {
	AccountIdentifier string `json:"accountIdentifier"`
	CredentialKey     string `json:"credentialKey"`
}

type mastodonDestinationConfig struct {
	CredentialKey string `json:"credentialKey"`
	InstanceURL   string `json:"instanceURL"`
}

type renderedExecutionTarget struct {
	destination domain.Destination
	content     string
}

type ExecutionService struct {
	destinations ports.DestinationRepository
	settings     ports.SettingsRepository
	history      ports.PostHistoryRepository
	sessions     ports.LiveNowSessionRepository
	discord      ports.DiscordPublisher
	bluesky      ports.BlueskyPublisher
	mastodon     ports.MastodonPublisher
	liveNow      *BlueskyLiveNowService
	duplicates   *duplicate.Service
	clock        Clock
}

func NewExecutionService(
	destinations ports.DestinationRepository,
	settings ports.SettingsRepository,
	history ports.PostHistoryRepository,
	sessions ports.LiveNowSessionRepository,
	discord ports.DiscordPublisher,
	bluesky ports.BlueskyPublisher,
	mastodon ports.MastodonPublisher,
) *ExecutionService {
	manager, _ := bluesky.(ports.BlueskyLiveNowManager)
	return &ExecutionService{
		destinations: destinations,
		settings:     settings,
		history:      history,
		sessions:     sessions,
		discord:      discord,
		bluesky:      bluesky,
		mastodon:     mastodon,
		liveNow:      NewBlueskyLiveNowService(manager),
		duplicates:   duplicate.NewService(history),
		clock:        systemClock{},
	}
}

func (s *ExecutionService) DryRun(ctx context.Context, announcement domain.Announcement) (domain.ExecutionSummary, error) {
	results, settings, _, err := s.execute(ctx, domain.ExecutionModeDryRun, announcement, false)
	if err != nil {
		return domain.ExecutionSummary{}, err
	}
	return buildExecutionSummary(domain.ExecutionModeDryRun, settings, results, nil), nil
}

func (s *ExecutionService) GoLive(ctx context.Context, announcement domain.Announcement) (domain.ExecutionSummary, error) {
	results, settings, warning, err := s.execute(ctx, domain.ExecutionModeGoLive, announcement, false)
	if err != nil {
		return domain.ExecutionSummary{}, err
	}
	return buildExecutionSummary(domain.ExecutionModeGoLive, settings, results, warning), nil
}

func (s *ExecutionService) ForceGoLive(ctx context.Context, announcement domain.Announcement) (domain.ExecutionSummary, error) {
	results, settings, _, err := s.execute(ctx, domain.ExecutionModeGoLive, announcement, true)
	if err != nil {
		return domain.ExecutionSummary{}, err
	}
	return buildExecutionSummary(domain.ExecutionModeGoLive, settings, results, nil), nil
}

func (s *ExecutionService) EndStream(ctx context.Context) (domain.ExecutionSummary, error) {
	settings, err := s.settings.Load(ctx)
	if err != nil {
		return domain.ExecutionSummary{}, err
	}

	destinations, err := s.destinations.List(ctx)
	if err != nil {
		return domain.ExecutionSummary{}, err
	}

	sessions, err := s.sessions.List(ctx)
	if err != nil {
		return domain.ExecutionSummary{}, err
	}
	sessionByDestination := make(map[string]domain.ActiveLiveNowSession, len(sessions))
	for _, session := range sessions {
		sessionByDestination[session.DestinationID] = session
	}

	results := make([]domain.ExecutionResult, 0, len(destinations)*2)
	for _, destination := range destinations {
		if destination.Platform != domain.PlatformBluesky {
			continue
		}

		session, ok := sessionByDestination[destination.ID]
		if !ok {
			continue
		}
		if !trackedSessionMatchesDestination(destination, session) {
			if err := s.sessions.Delete(ctx, destination.ID); err != nil {
				return domain.ExecutionSummary{}, err
			}
			continue
		}

		if err := validateDestinationConfig(destination); err != nil {
			results = append(results, domain.ExecutionResult{
				DestinationID:   destination.ID,
				DestinationName: destination.Name,
				Platform:        destination.Platform,
				State:           domain.ExecutionStateValidationError,
				Message:         err.Error(),
			})
			continue
		}

		routedDestination, routingNote, err := routeDestinationForExecution(destination, settings)
		if err != nil {
			results = append(results, domain.ExecutionResult{
				DestinationID:   destination.ID,
				DestinationName: destination.Name,
				Platform:        destination.Platform,
				State:           domain.ExecutionStateValidationError,
				Message:         err.Error(),
			})
			continue
		}

		err = s.clearBlueskyLiveNow(ctx, routedDestination, session)
		if err != nil {
			state := domain.ExecutionStateFailed
			if domain.IsIntegrationUnavailable(err) {
				state = domain.ExecutionStateSkipped
			}
			results = append(results, domain.ExecutionResult{
				DestinationID:   destination.ID,
				DestinationName: destination.Name,
				Platform:        destination.Platform,
				State:           state,
				Message:         withRoutingNote(err.Error(), routingNote),
			})
			continue
		}
		if err := s.sessions.Delete(ctx, destination.ID); err != nil {
			return domain.ExecutionSummary{}, err
		}

		results = append(results, domain.ExecutionResult{
			DestinationID:   destination.ID,
			DestinationName: destination.Name,
			Platform:        destination.Platform,
			State:           domain.ExecutionStateSuccess,
			Message:         withRoutingNote("Live Now cleared successfully.", routingNote),
		})
	}

	if settings.EndStreamPostEnabled {
		announcement := domain.NormalizeAnnouncement(domain.Announcement{
			Message: settings.EndStreamTemplate,
		}, settings)

		for _, destination := range destinations {
			if err := validateDestinationConfig(destination); err != nil {
				results = append(results, domain.ExecutionResult{
					DestinationID:   destination.ID,
					DestinationName: destination.Name,
					Platform:        destination.Platform,
					State:           domain.ExecutionStateValidationError,
					Message:         err.Error(),
				})
				continue
			}

			routedDestination, routingNote, err := routeDestinationForExecution(destination, settings)
			if err != nil {
				results = append(results, domain.ExecutionResult{
					DestinationID:   destination.ID,
					DestinationName: destination.Name,
					Platform:        destination.Platform,
					State:           domain.ExecutionStateValidationError,
					Message:         err.Error(),
				})
				continue
			}

			announcementForDestination := announcement
			if session, ok := sessionByDestination[destination.ID]; ok {
				announcementForDestination.StreamTitle = session.StreamTitle
				announcementForDestination.StreamURL = session.StreamURL
			}

			content := templates.Render(settings.EndStreamTemplate, announcementForDestination, destination.Platform, s.clock.Now())
			notes := domain.ValidatePreviewContent(destination.Platform, content)
			if len(notes) > 0 {
				results = append(results, domain.ExecutionResult{
					DestinationID:   destination.ID,
					DestinationName: destination.Name,
					Platform:        destination.Platform,
					State:           domain.ExecutionStateValidationError,
					Message:         notes[0],
					Content:         content,
				})
				continue
			}

			if err := s.publish(ctx, routedDestination, content); err != nil {
				state := domain.ExecutionStateFailed
				if domain.IsIntegrationUnavailable(err) {
					state = domain.ExecutionStateSkipped
				}
				results = append(results, domain.ExecutionResult{
					DestinationID:   destination.ID,
					DestinationName: destination.Name,
					Platform:        destination.Platform,
					State:           state,
					Message:         withRoutingNote(fmt.Sprintf("End stream post failed: %s", err.Error()), routingNote),
					Content:         content,
				})
				continue
			}

			results = append(results, domain.ExecutionResult{
				DestinationID:   destination.ID,
				DestinationName: destination.Name,
				Platform:        destination.Platform,
				State:           domain.ExecutionStateSuccess,
				Message:         withRoutingNote("End stream post published successfully.", routingNote),
				Content:         content,
			})
		}
	}

	return buildExecutionSummary(domain.ExecutionModeEndStream, settings, results, nil), nil
}

func (s *ExecutionService) execute(ctx context.Context, mode domain.ExecutionMode, announcement domain.Announcement, skipDuplicateWarning bool) ([]domain.ExecutionResult, domain.AppSettings, *duplicate.Warning, error) {
	settings, err := s.settings.Load(ctx)
	if err != nil {
		return nil, domain.AppSettings{}, nil, err
	}

	destinations, err := s.destinations.List(ctx)
	if err != nil {
		return nil, domain.AppSettings{}, nil, err
	}
	destinations = destinationsForSelection(destinations, announcement.DestinationIDs)

	normalized := domain.NormalizeAnnouncement(announcement, settings)
	announcementNotes := domain.ValidateAnnouncement(normalized)
	now := s.clock.Now()
	results := make([]domain.ExecutionResult, 0, len(destinations))
	renderedTargets := make([]renderedExecutionTarget, 0, len(destinations))

	for _, destination := range destinations {
		content := templates.Render(destination.Template, normalized, destination.Platform, now)
		renderedTargets = append(renderedTargets, renderedExecutionTarget{destination: destination, content: content})
		notes := append([]string{}, announcementNotes...)
		notes = append(notes, domain.ValidatePreviewContent(destination.Platform, content)...)
		if len(notes) > 0 {
			results = append(results, domain.ExecutionResult{
				DestinationID:   destination.ID,
				DestinationName: destination.Name,
				Platform:        destination.Platform,
				State:           domain.ExecutionStateValidationError,
				Message:         notes[0],
				Content:         content,
			})
			continue
		}
	}

	if mode == domain.ExecutionModeDryRun {
		for _, rendered := range renderedTargets {
			if hasValidationResult(results, rendered.destination.ID) {
				continue
			}
			if err := validateDestinationConfig(rendered.destination); err != nil {
				results = append(results, domain.ExecutionResult{
					DestinationID:   rendered.destination.ID,
					DestinationName: rendered.destination.Name,
					Platform:        rendered.destination.Platform,
					State:           domain.ExecutionStateValidationError,
					Message:         err.Error(),
					Content:         rendered.content,
				})
				continue
			}
			results = append(results, domain.ExecutionResult{
				DestinationID:   rendered.destination.ID,
				DestinationName: rendered.destination.Name,
				Platform:        rendered.destination.Platform,
				State:           domain.ExecutionStateSuccess,
				Message:         "Dry Run simulated successfully.",
				Content:         rendered.content,
			})
		}
		return results, settings, nil, nil
	}

	if !skipDuplicateWarning {
		candidates := make(map[domain.Destination]string)
		for _, rendered := range renderedTargets {
			if hasValidationResult(results, rendered.destination.ID) {
				continue
			}
			candidates[rendered.destination] = rendered.content
		}
		warning, err := s.duplicates.Check(ctx, settings, candidates, now)
		if err != nil {
			return nil, domain.AppSettings{}, nil, err
		}
		if warning != nil {
			return results, settings, warning, nil
		}
	}

	for _, rendered := range renderedTargets {
		destination := rendered.destination
		content := rendered.content
		if hasValidationResult(results, destination.ID) {
			continue
		}

		if err := validateDestinationConfig(destination); err != nil {
			results = append(results, domain.ExecutionResult{
				DestinationID:   destination.ID,
				DestinationName: destination.Name,
				Platform:        destination.Platform,
				State:           domain.ExecutionStateValidationError,
				Message:         err.Error(),
				Content:         content,
			})
			continue
		}

		routedDestination, routingNote, err := routeDestinationForExecution(destination, settings)
		if err != nil {
			results = append(results, domain.ExecutionResult{
				DestinationID:   destination.ID,
				DestinationName: destination.Name,
				Platform:        destination.Platform,
				State:           domain.ExecutionStateValidationError,
				Message:         err.Error(),
				Content:         content,
			})
			continue
		}

		err = s.publish(ctx, routedDestination, content)
		if err != nil {
			state := domain.ExecutionStateFailed
			if domain.IsIntegrationUnavailable(err) {
				state = domain.ExecutionStateSkipped
			}
			results = append(results, domain.ExecutionResult{
				DestinationID:   destination.ID,
				DestinationName: destination.Name,
				Platform:        destination.Platform,
				State:           state,
				Message:         withRoutingNote(err.Error(), routingNote),
				Content:         content,
			})
			continue
		}

		message := withRoutingNote("Published successfully.", routingNote)
		state := domain.ExecutionStateSuccess

		if destination.Platform == domain.PlatformBluesky {
			if err := s.applyBlueskyLiveNow(ctx, routedDestination, normalized); err != nil {
				if err := s.history.Record(ctx, destination.ID, content, duplicate.HashContent(content), now); err != nil {
					return nil, domain.AppSettings{}, nil, err
				}
				results = append(results, domain.ExecutionResult{
					DestinationID:   destination.ID,
					DestinationName: destination.Name,
					Platform:        destination.Platform,
					State:           domain.ExecutionStateFailed,
					Message:         withRoutingNote(fmt.Sprintf("Published successfully, but Live Now update failed: %s", err.Error()), routingNote),
					Content:         content,
				})
				continue
			}
			if err := s.trackBlueskyLiveNowSession(ctx, destination, routedDestination, normalized, now); err != nil {
				if err := s.history.Record(ctx, destination.ID, content, duplicate.HashContent(content), now); err != nil {
					return nil, domain.AppSettings{}, nil, err
				}
				results = append(results, domain.ExecutionResult{
					DestinationID:   destination.ID,
					DestinationName: destination.Name,
					Platform:        destination.Platform,
					State:           domain.ExecutionStateFailed,
					Message:         withRoutingNote(fmt.Sprintf("Published successfully and Live Now was set, but recovery tracking failed: %s", err.Error()), routingNote),
					Content:         content,
				})
				continue
			}
			message = withRoutingNote("Published successfully. Live Now set.", routingNote)
		}

		if err := s.history.Record(ctx, destination.ID, content, duplicate.HashContent(content), now); err != nil {
			return nil, domain.AppSettings{}, nil, err
		}

		results = append(results, domain.ExecutionResult{
			DestinationID:   destination.ID,
			DestinationName: destination.Name,
			Platform:        destination.Platform,
			State:           state,
			Message:         message,
			Content:         content,
		})
	}

	return results, settings, nil, nil
}

func destinationsForSelection(destinations []domain.Destination, selectedIDs []string) []domain.Destination {
	if selectedIDs == nil {
		return destinations
	}

	allowed := make(map[string]struct{}, len(selectedIDs))
	for _, id := range selectedIDs {
		allowed[id] = struct{}{}
	}

	filtered := make([]domain.Destination, 0, len(destinations))
	for _, destination := range destinations {
		if _, ok := allowed[destination.ID]; ok {
			filtered = append(filtered, destination)
		}
	}

	return filtered
}

func (s *ExecutionService) applyBlueskyLiveNow(ctx context.Context, destination domain.Destination, announcement domain.Announcement) error {
	var config blueskyDestinationConfig
	if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
		return fmt.Errorf("invalid Bluesky config")
	}
	return s.liveNow.Set(ctx, config.AccountIdentifier, config.CredentialKey, domain.BlueskyLiveNowStatus{
		URL:         announcement.StreamURL,
		Title:       announcement.StreamTitle,
		Description: announcement.Message,
	})
}

func (s *ExecutionService) clearBlueskyLiveNow(ctx context.Context, destination domain.Destination, session domain.ActiveLiveNowSession) error {
	if strings.TrimSpace(session.AccountIdentifier) != "" && strings.TrimSpace(session.CredentialKey) != "" {
		return s.liveNow.Clear(ctx, session.AccountIdentifier, session.CredentialKey)
	}
	var config blueskyDestinationConfig
	if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
		return fmt.Errorf("invalid Bluesky config")
	}
	return s.liveNow.Clear(ctx, config.AccountIdentifier, config.CredentialKey)
}

func (s *ExecutionService) trackBlueskyLiveNowSession(ctx context.Context, original domain.Destination, routed domain.Destination, announcement domain.Announcement, startedAt time.Time) error {
	var config blueskyDestinationConfig
	if err := json.Unmarshal([]byte(routed.ConfigJSON), &config); err != nil {
		return fmt.Errorf("invalid Bluesky config")
	}
	return s.sessions.Upsert(ctx, domain.ActiveLiveNowSession{
		DestinationID:     original.ID,
		DestinationName:   original.Name,
		Platform:          string(original.Platform),
		AccountIdentifier: config.AccountIdentifier,
		CredentialKey:     config.CredentialKey,
		StreamURL:         announcement.StreamURL,
		StreamTitle:       announcement.StreamTitle,
		StartedAt:         startedAt,
	})
}

func (s *ExecutionService) publish(ctx context.Context, destination domain.Destination, content string) error {
	switch destination.Platform {
	case domain.PlatformDiscord:
		var config discordDestinationConfig
		if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
			return fmt.Errorf("invalid Discord config")
		}
		if config.WebhookKey == "" {
			return fmt.Errorf("missing Discord webhook key")
		}
		return s.discord.Publish(ctx, config.WebhookKey, content)
	case domain.PlatformBluesky:
		var config blueskyDestinationConfig
		if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
			return fmt.Errorf("invalid Bluesky config")
		}
		if config.AccountIdentifier == "" || config.CredentialKey == "" {
			return fmt.Errorf("missing Bluesky connection details")
		}
		return s.bluesky.PublishPost(ctx, config.AccountIdentifier, config.CredentialKey, content)
	case domain.PlatformMastodon:
		var config mastodonDestinationConfig
		if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
			return fmt.Errorf("invalid Mastodon config")
		}
		if config.CredentialKey == "" || config.InstanceURL == "" {
			return fmt.Errorf("missing Mastodon connection details")
		}
		return s.mastodon.PublishPost(ctx, config.CredentialKey, config.InstanceURL, content)
	default:
		return fmt.Errorf("unsupported destination platform")
	}
}

func hasValidationResult(results []domain.ExecutionResult, destinationID string) bool {
	for _, result := range results {
		if result.DestinationID == destinationID && result.State == domain.ExecutionStateValidationError {
			return true
		}
	}
	return false
}

func warningMessage(warning *duplicate.Warning) string {
	if warning == nil {
		return ""
	}
	return warning.Message
}

func routeDestinationForExecution(destination domain.Destination, settings domain.AppSettings) (domain.Destination, string, error) {
	if !settings.TestModeEnabled {
		return destination, "", nil
	}

	routed := destination

	switch destination.Platform {
	case domain.PlatformDiscord:
		var config discordDestinationConfig
		if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
			return domain.Destination{}, "", fmt.Errorf("invalid Discord config")
		}
		if strings.TrimSpace(settings.TestDiscordWebhookKey) == "" {
			return domain.Destination{}, "", fmt.Errorf("test Discord webhook key is required when test mode is enabled")
		}
		config.WebhookKey = settings.TestDiscordWebhookKey
		encoded, err := json.Marshal(config)
		if err != nil {
			return domain.Destination{}, "", fmt.Errorf("encode Discord test config: %w", err)
		}
		routed.ConfigJSON = string(encoded)
	case domain.PlatformBluesky:
		var config blueskyDestinationConfig
		if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
			return domain.Destination{}, "", fmt.Errorf("invalid Bluesky config")
		}
		if strings.TrimSpace(settings.TestBlueskyAccountIdentifier) == "" {
			return domain.Destination{}, "", fmt.Errorf("test Bluesky account identifier is required when test mode is enabled")
		}
		if strings.TrimSpace(settings.TestBlueskyCredentialKey) == "" {
			return domain.Destination{}, "", fmt.Errorf("test Bluesky credential key is required when test mode is enabled")
		}
		config.AccountIdentifier = settings.TestBlueskyAccountIdentifier
		config.CredentialKey = settings.TestBlueskyCredentialKey
		encoded, err := json.Marshal(config)
		if err != nil {
			return domain.Destination{}, "", fmt.Errorf("encode Bluesky test config: %w", err)
		}
		routed.ConfigJSON = string(encoded)
	case domain.PlatformMastodon:
		var config mastodonDestinationConfig
		if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
			return domain.Destination{}, "", fmt.Errorf("invalid Mastodon config")
		}
		if strings.TrimSpace(settings.TestMastodonCredentialKey) == "" {
			return domain.Destination{}, "", fmt.Errorf("test Mastodon credential key is required when test mode is enabled")
		}
		if strings.TrimSpace(settings.TestMastodonInstanceURL) == "" {
			return domain.Destination{}, "", fmt.Errorf("test Mastodon instance URL is required when test mode is enabled")
		}
		config.CredentialKey = settings.TestMastodonCredentialKey
		config.InstanceURL = settings.TestMastodonInstanceURL
		encoded, err := json.Marshal(config)
		if err != nil {
			return domain.Destination{}, "", fmt.Errorf("encode Mastodon test config: %w", err)
		}
		routed.ConfigJSON = string(encoded)
	default:
		return domain.Destination{}, "", fmt.Errorf("unsupported destination platform")
	}

	return routed, "Test Mode route applied.", nil
}

func withRoutingNote(message, routingNote string) string {
	if routingNote == "" {
		return message
	}
	return fmt.Sprintf("%s %s", message, routingNote)
}

func buildExecutionSummary(
	mode domain.ExecutionMode,
	settings domain.AppSettings,
	results []domain.ExecutionResult,
	warning *duplicate.Warning,
) domain.ExecutionSummary {
	summary := summarizeExecution(mode, results)
	summary.TestModeActive = settings.TestModeEnabled
	summary.RequiresDuplicateConfirmation = warning != nil
	summary.DuplicateWarningMessage = warningMessage(warning)
	summary.Status = executionSummaryStatus(summary)
	return summary
}

func summarizeExecution(mode domain.ExecutionMode, results []domain.ExecutionResult) domain.ExecutionSummary {
	summary := domain.ExecutionSummary{
		Mode:    mode,
		Status:  domain.ExecutionSummaryStatusSuccess,
		Results: results,
	}

	for _, result := range results {
		switch result.State {
		case domain.ExecutionStateSuccess:
			summary.SuccessCount++
		case domain.ExecutionStateFailed:
			summary.FailedCount++
		case domain.ExecutionStateSkipped:
			summary.SkippedCount++
		case domain.ExecutionStateValidationError:
			summary.ValidationErrorCount++
		}
	}

	summary.TotalCount = len(results)
	return summary
}

func executionSummaryStatus(summary domain.ExecutionSummary) domain.ExecutionSummaryStatus {
	switch {
	case summary.FailedCount > 0 || summary.ValidationErrorCount > 0:
		return domain.ExecutionSummaryStatusPartial
	case summary.RequiresDuplicateConfirmation || summary.TestModeActive || summary.SkippedCount > 0:
		return domain.ExecutionSummaryStatusWarning
	default:
		return domain.ExecutionSummaryStatusSuccess
	}
}
