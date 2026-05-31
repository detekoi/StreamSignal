package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/secrets/memory"
	"StreamSignal/internal/storage/sqlite"
)

func newTestApp(t *testing.T) *App {
	t.Helper()

	app := newAppWithDependencies(t.TempDir()+"\\streamsignal-test.db", memory.NewStore())
	app.ctx = context.Background()
	t.Cleanup(func() {
		if app.db != nil {
			_ = app.db.Close()
		}
	})

	return app
}

func TestExecutionLogStatus(t *testing.T) {
	tests := []struct {
		name    string
		summary domain.ExecutionSummary
		want    string
	}{
		{
			name:    "success",
			summary: domain.ExecutionSummary{Status: domain.ExecutionSummaryStatusSuccess, SuccessCount: 2, TotalCount: 2},
			want:    "SUCCESS",
		},
		{
			name:    "duplicate confirmation warning",
			summary: domain.ExecutionSummary{Status: domain.ExecutionSummaryStatusWarning, RequiresDuplicateConfirmation: true},
			want:    "WARNING",
		},
		{
			name:    "partial failure",
			summary: domain.ExecutionSummary{Status: domain.ExecutionSummaryStatusPartial, FailedCount: 1, TotalCount: 1},
			want:    "PARTIAL",
		},
		{
			name:    "validation attention",
			summary: domain.ExecutionSummary{Status: domain.ExecutionSummaryStatusPartial, ValidationErrorCount: 1, TotalCount: 1},
			want:    "PARTIAL",
		},
		{
			name:    "integration unavailable warning",
			summary: domain.ExecutionSummary{Status: domain.ExecutionSummaryStatusWarning, SkippedCount: 1, TotalCount: 1},
			want:    "WARNING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := executionLogStatus(tt.summary); got != tt.want {
				t.Fatalf("executionLogStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExecutionLogMessageIncludesCountsAndConfirmationState(t *testing.T) {
	summary := domain.ExecutionSummary{
		Status:                        domain.ExecutionSummaryStatusWarning,
		TestModeActive:                true,
		TotalCount:                    3,
		SuccessCount:                  1,
		FailedCount:                   1,
		SkippedCount:                  1,
		ValidationErrorCount:          0,
		RequiresDuplicateConfirmation: true,
	}

	got := executionLogMessage("Go Live", summary)
	wantParts := []string{
		"Go Live returned 3 results",
		"1 success, 1 failed, 1 skipped, 0 validation",
		"Test Mode routing was active.",
		"Duplicate confirmation required.",
	}

	for _, part := range wantParts {
		if !strings.Contains(got, part) {
			t.Fatalf("executionLogMessage() = %q, missing %q", got, part)
		}
	}
}

func TestAppSaveSettingsPersistsAndLogs(t *testing.T) {
	app := newTestApp(t)

	settings := domain.DefaultAppSettings()
	settings.DefaultStreamURL = "https://example.com/live"
	settings.DefaultHashtags = "#streamsignal"

	saved, err := app.SaveSettings(settings)
	if err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if saved.DefaultStreamURL != "https://example.com/live" {
		t.Fatalf("expected persisted stream url, got %+v", saved)
	}

	loaded, err := app.GetSettings()
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if loaded.DefaultHashtags != "#streamsignal" {
		t.Fatalf("expected persisted hashtags, got %+v", loaded)
	}

	logs, err := app.GetLogs()
	if err != nil {
		t.Fatalf("get logs: %v", err)
	}
	found := false
	for _, entry := range logs {
		if entry.Action == "save_settings" && entry.Status == "SUCCESS" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected save_settings log entry, got %+v", logs)
	}
}

func TestAppGeneratePreviewAndDiagnostics(t *testing.T) {
	app := newTestApp(t)

	_, err := app.SaveDestination(domain.Destination{
		ID:         "discord-main",
		Platform:   domain.PlatformDiscord,
		Name:       "Main Discord",
		Enabled:    true,
		Template:   "Going live: {{stream_title}} {{stream_url}}",
		ConfigJSON: `{"webhookKey":"https://discord.example/webhook"}`,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("save destination: %v", err)
	}

	items, err := app.GeneratePreview(domain.Announcement{
		StreamTitle: "Late Night Variety",
		StreamURL:   "https://example.com/live",
	})
	if err != nil {
		t.Fatalf("generate preview: %v", err)
	}
	if len(items) != 1 || !strings.Contains(items[0].Content, "Late Night Variety") {
		t.Fatalf("expected rendered preview, got %+v", items)
	}

	diagnostics, err := app.GetDiagnostics()
	if err != nil {
		t.Fatalf("get diagnostics: %v", err)
	}
	for _, expected := range []string{
		"Destinations Configured: 1",
		"Main Discord [discord] enabled=true",
		"Preview | SUCCESS",
	} {
		if !strings.Contains(diagnostics, expected) {
			t.Fatalf("expected diagnostics to contain %q, got:\n%s", expected, diagnostics)
		}
	}
}

func TestAppDryRunDelegatesAndLogs(t *testing.T) {
	app := newTestApp(t)

	_, err := app.SaveDestination(domain.Destination{
		ID:         "discord-main",
		Platform:   domain.PlatformDiscord,
		Name:       "Main Discord",
		Enabled:    true,
		Template:   "{{stream_title}} {{stream_url}}",
		ConfigJSON: `{"webhookKey":"https://discord.example/webhook"}`,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("save destination: %v", err)
	}

	summary, err := app.DryRun(domain.Announcement{
		StreamTitle: "Live",
		StreamURL:   "https://example.com/live",
	})
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if summary.Mode != domain.ExecutionModeDryRun || summary.SuccessCount != 1 {
		t.Fatalf("unexpected dry run summary: %+v", summary)
	}

	logs, err := app.GetLogs()
	if err != nil {
		t.Fatalf("get logs: %v", err)
	}
	found := false
	for _, entry := range logs {
		if entry.Action == "dry_run" && entry.Status == "SUCCESS" {
			found = true
			if !strings.Contains(entry.Message, "Dry Run returned 1 results") {
				t.Fatalf("expected dry run summary log message, got %+v", entry)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected dry_run log entry, got %+v", logs)
	}
}

func TestAppEndStreamDelegatesToExecutionServiceAndLogs(t *testing.T) {
	app := newTestApp(t)

	destination, err := app.SaveDestination(domain.Destination{
		ID:         "bluesky-main",
		Platform:   domain.PlatformBluesky,
		Name:       "Main Bluesky",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"accountIdentifier":"don.main","credentialKey":"bluesky/main"}`,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("save destination: %v", err)
	}
	if destination.ID == "" {
		t.Fatal("expected saved destination id")
	}

	summary, err := app.EndStream()
	if err != nil {
		t.Fatalf("end stream: %v", err)
	}
	if summary.Mode != domain.ExecutionModeEndStream {
		t.Fatalf("expected end stream mode, got %+v", summary)
	}

	logs, err := app.GetLogs()
	if err != nil {
		t.Fatalf("get logs: %v", err)
	}
	found := false
	for _, entry := range logs {
		if entry.Action == "end_stream" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected end_stream log entry, got %+v", logs)
	}
}

func TestAppRecoveryBindingsReturnErrorsForMissingSessions(t *testing.T) {
	app := newTestApp(t)

	pending, err := app.ListPendingLiveNowSessions()
	if err != nil {
		t.Fatalf("list pending live now sessions: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected no pending sessions, got %+v", pending)
	}

	_, err = app.ClearPendingLiveNowSession("bluesky-main")
	if err == nil {
		t.Fatal("expected missing-session error")
	}

	logs, logErr := app.GetLogs()
	if logErr != nil {
		t.Fatalf("get logs: %v", logErr)
	}

	found := false
	for _, entry := range logs {
		if entry.Action == "clear_pending_live_now" && entry.Status == "FAILED" {
			found = true
			if !strings.Contains(entry.Message, "no pending Live Now session found") {
				t.Fatalf("expected missing-session log message, got %+v", entry)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected failed clear_pending_live_now log entry, got %+v", logs)
	}
}

func TestAppRecoveryBindingsSurfaceSeededSession(t *testing.T) {
	app := newTestApp(t)

	repository := sqlite.NewLiveNowSessionRepository(app.db)
	err := repository.Upsert(context.Background(), domain.ActiveLiveNowSession{
		DestinationID:     "bluesky-main",
		DestinationName:   "Main Bluesky",
		Platform:          string(domain.PlatformBluesky),
		AccountIdentifier: "don.main",
		CredentialKey:     "bluesky/main",
		StreamURL:         "https://example.com/live",
		StreamTitle:       "Going Live",
		StartedAt:         time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seed live now session: %v", err)
	}

	pending, err := app.ListPendingLiveNowSessions()
	if err != nil {
		t.Fatalf("list pending live now sessions: %v", err)
	}
	if len(pending) != 1 || pending[0].DestinationID != "bluesky-main" {
		t.Fatalf("expected seeded pending session, got %+v", pending)
	}
}
