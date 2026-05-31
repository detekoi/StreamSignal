package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"StreamSignal/internal/ports"
)

type DiagnosticsService struct {
	settings     ports.SettingsRepository
	destinations ports.DestinationRepository
	logs         ports.LogRepository
	sessions     ports.LiveNowSessionRepository
	clock        Clock
}

func NewDiagnosticsService(settings ports.SettingsRepository, destinations ports.DestinationRepository, logs ports.LogRepository, sessions ports.LiveNowSessionRepository) *DiagnosticsService {
	return &DiagnosticsService{
		settings:     settings,
		destinations: destinations,
		logs:         logs,
		sessions:     sessions,
		clock:        systemClock{},
	}
}

func (s *DiagnosticsService) Build(ctx context.Context) (string, error) {
	settings, err := s.settings.Load(ctx)
	if err != nil {
		return "", err
	}

	destinations, err := s.destinations.List(ctx)
	if err != nil {
		return "", err
	}

	logs, err := s.logs.ListRecent(ctx, 50)
	if err != nil {
		return "", err
	}
	sessions, err := s.sessions.List(ctx)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "StreamSignal Diagnostics\n")
	fmt.Fprintf(&builder, "Generated At: %s\n", s.clock.Now().Format(time.RFC3339Nano))
	fmt.Fprintf(&builder, "Test Mode Enabled: %t\n", settings.TestModeEnabled)
	fmt.Fprintf(&builder, "Duplicate Protection Enabled: %t\n", settings.DuplicateProtectionEnabled)
	fmt.Fprintf(&builder, "Duplicate Window Minutes: %d\n", settings.DuplicateWindowMinutes)
	fmt.Fprintf(&builder, "End Stream Post Enabled: %t\n", settings.EndStreamPostEnabled)
	fmt.Fprintf(&builder, "Destinations Configured: %d\n", len(destinations))
	fmt.Fprintf(&builder, "Pending Live Now Sessions: %d\n", len(sessions))
	fmt.Fprintf(&builder, "Sensitive values: redacted\n")
	fmt.Fprintf(&builder, "\nDestinations\n")
	for _, destination := range destinations {
		fmt.Fprintf(&builder, "- %s [%s] enabled=%t\n", destination.Name, destination.Platform, destination.Enabled)
	}

	fmt.Fprintf(&builder, "\nPending Live Now Sessions\n")
	if len(sessions) == 0 {
		fmt.Fprintf(&builder, "- none\n")
	}
	for _, session := range sessions {
		fmt.Fprintf(&builder, "- %s [%s] title=%q url=%q started_at=%s\n",
			session.DestinationName,
			session.Platform,
			redactedDiagnosticValue(session.StreamTitle),
			redactedDiagnosticValue(session.StreamURL),
			session.StartedAt.Format(time.RFC3339Nano),
		)
	}

	fmt.Fprintf(&builder, "\nRecent Logs\n")
	if len(logs) == 0 {
		fmt.Fprintf(&builder, "- none\n")
	}
	for _, entry := range logs {
		fmt.Fprintf(&builder, "- %s | %s | %s | %s | %s\n",
			entry.Timestamp.Format(time.RFC3339Nano),
			entry.Destination,
			diagnosticsActionLabel(entry.Action),
			entry.Status,
			entry.Message,
		)
	}

	return builder.String(), nil
}

func redactedDiagnosticValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}

	return "[redacted]"
}

func diagnosticsActionLabel(action string) string {
	switch action {
	case "generate_preview":
		return "Preview"
	case "dry_run":
		return "Dry Run"
	case "go_live":
		return "Go Live"
	case "force_go_live":
		return "Confirmed Go Live"
	case "end_stream":
		return "End Stream"
	case "clear_pending_live_now":
		return "Live Now Recovery"
	case "save_destination":
		return "Save Destination"
	case "delete_destination":
		return "Delete Destination"
	case "save_settings":
		return "Save Settings"
	default:
		return action
	}
}
