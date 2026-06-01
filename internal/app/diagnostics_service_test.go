package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"StreamSignal/internal/domain"
)

type logRepositoryStub struct {
	items []domain.LogEntry
}

func (s *logRepositoryStub) Append(_ context.Context, entry domain.LogEntry) error {
	s.items = append(s.items, entry)
	return nil
}

func (s *logRepositoryStub) ListRecent(context.Context, int) ([]domain.LogEntry, error) {
	return append([]domain.LogEntry(nil), s.items...), nil
}

func TestDiagnosticsServiceBuildIncludesSafeSummary(t *testing.T) {
	settingsRepo := &settingsRepositoryStub{
		item: domain.AppSettings{
			TestModeEnabled:            true,
			DuplicateProtectionEnabled: true,
			DuplicateWindowMinutes:     10,
		},
	}
	destinationsRepo := newDestinationRepositoryStub()
	destinationsRepo.items["discord-main"] = domain.Destination{
		ID:       "discord-main",
		Platform: domain.PlatformDiscord,
		Name:     "Main Discord",
		Enabled:  true,
	}
	logRepo := &logRepositoryStub{
		items: []domain.LogEntry{
			{
				Timestamp:   time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC),
				Destination: "Main Discord",
				Action:      "generate_preview",
				Status:      "SUCCESS",
				Message:     "Generated preview.",
			},
		},
	}
	sessionRepo := &liveNowSessionRepositoryStub{
		items: []domain.ActiveLiveNowSession{
			{
				DestinationID:     "bluesky-main",
				DestinationName:   "Main Bluesky",
				Platform:          string(domain.PlatformBluesky),
				AccountIdentifier: "don.main",
				CredentialKey:     "bluesky/main",
				StreamURL:         "https://example.com/live",
				StreamTitle:       "Going Live",
				StartedAt:         time.Date(2026, 5, 31, 11, 55, 0, 0, time.UTC),
			},
		},
	}

	service := NewDiagnosticsService(settingsRepo, destinationsRepo, logRepo, sessionRepo)
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 13, 0, 0, 0, time.UTC)}

	diagnostics, err := service.Build(context.Background())
	if err != nil {
		t.Fatalf("build diagnostics: %v", err)
	}

	for _, expected := range []string{
		"StreamSignal Diagnostics",
		"End Stream Post Enabled: false",
		"Destinations Configured: 1",
		"Pending Live Now Sessions: 1",
		"Sensitive values: redacted",
		"Main Discord [discord] enabled=true",
		`Main Bluesky [bluesky] title="[redacted]" url="[redacted]"`,
		"Preview | SUCCESS | Generated preview.",
	} {
		if !strings.Contains(diagnostics, expected) {
			t.Fatalf("expected diagnostics to contain %q, got:\n%s", expected, diagnostics)
		}
	}

	for _, unexpected := range []string{"Going Live", "https://example.com/live"} {
		if strings.Contains(diagnostics, unexpected) {
			t.Fatalf("expected diagnostics to redact %q, got:\n%s", unexpected, diagnostics)
		}
	}
}
