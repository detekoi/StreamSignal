package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/ports"
)

type discordPublisherStub struct {
	calls []struct {
		key      string
		content  string
		metadata domain.DiscordPostMetadata
	}
	err error
}

func (s *discordPublisherStub) Publish(_ context.Context, webhookKey string, content string, metadata domain.DiscordPostMetadata) error {
	s.calls = append(s.calls, struct {
		key      string
		content  string
		metadata domain.DiscordPostMetadata
	}{webhookKey, content, metadata})
	return s.err
}

type blueskyPublisherStub struct {
	calls []struct {
		identifier string
		key        string
		content    string
		metadata   domain.BlueskyPostMetadata
	}
	liveNowSetCalls []struct {
		identifier string
		key        string
		status     domain.BlueskyLiveNowStatus
	}
	liveNowClearCalls []struct {
		identifier string
		key        string
	}
	err             error
	liveNowSetErr   error
	liveNowClearErr error
}

func (s *blueskyPublisherStub) PublishPost(_ context.Context, accountIdentifier string, credentialKey string, content string, metadata domain.BlueskyPostMetadata) error {
	s.calls = append(s.calls, struct {
		identifier string
		key        string
		content    string
		metadata   domain.BlueskyPostMetadata
	}{accountIdentifier, credentialKey, content, metadata})
	return s.err
}

func (s *blueskyPublisherStub) SetLiveNow(_ context.Context, accountIdentifier string, credentialKey string, status domain.BlueskyLiveNowStatus) error {
	s.liveNowSetCalls = append(s.liveNowSetCalls, struct {
		identifier string
		key        string
		status     domain.BlueskyLiveNowStatus
	}{accountIdentifier, credentialKey, status})
	return s.liveNowSetErr
}

func (s *blueskyPublisherStub) ClearLiveNow(_ context.Context, accountIdentifier string, credentialKey string) error {
	s.liveNowClearCalls = append(s.liveNowClearCalls, struct {
		identifier string
		key        string
	}{accountIdentifier, credentialKey})
	return s.liveNowClearErr
}

type mastodonPublisherStub struct {
	calls []struct {
		key         string
		instanceURL string
		content     string
		metadata    domain.MastodonPostMetadata
	}
	err error
}

func (s *mastodonPublisherStub) PublishPost(_ context.Context, credentialKey string, instanceURL string, content string, metadata domain.MastodonPostMetadata) error {
	s.calls = append(s.calls, struct {
		key         string
		instanceURL string
		content     string
		metadata    domain.MastodonPostMetadata
	}{credentialKey, instanceURL, content, metadata})
	return s.err
}

type postHistoryRepositoryStub struct {
	recordsByDestination map[string][]ports.PostHistoryRecord
	recordCalls          []struct {
		destinationID string
		content       string
		hash          string
		postedAt      time.Time
	}
}

func (s *postHistoryRepositoryStub) Record(_ context.Context, destinationID string, renderedContent string, hash string, postedAt time.Time) error {
	s.recordCalls = append(s.recordCalls, struct {
		destinationID string
		content       string
		hash          string
		postedAt      time.Time
	}{destinationID, renderedContent, hash, postedAt})
	return nil
}

func (s *postHistoryRepositoryStub) FindRecentByDestination(_ context.Context, destinationID string, _ time.Time) ([]ports.PostHistoryRecord, error) {
	return append([]ports.PostHistoryRecord(nil), s.recordsByDestination[destinationID]...), nil
}

type liveNowSessionRepositoryExecutionStub struct {
	items     []domain.ActiveLiveNowSession
	upserts   []domain.ActiveLiveNowSession
	deletes   []string
	upsertErr error
	deleteErr error
}

func (s *liveNowSessionRepositoryExecutionStub) Upsert(_ context.Context, session domain.ActiveLiveNowSession) error {
	s.upserts = append(s.upserts, session)
	return s.upsertErr
}

func (s *liveNowSessionRepositoryExecutionStub) List(context.Context) ([]domain.ActiveLiveNowSession, error) {
	return append([]domain.ActiveLiveNowSession(nil), s.items...), nil
}

func (s *liveNowSessionRepositoryExecutionStub) Delete(_ context.Context, destinationID string) error {
	s.deletes = append(s.deletes, destinationID)
	return s.deleteErr
}

type unavailableDiscordPublisherStub struct{}

func (unavailableDiscordPublisherStub) Publish(context.Context, string, string, domain.DiscordPostMetadata) error {
	return &domain.IntegrationUnavailableError{
		Platform: domain.PlatformDiscord,
		Message:  "Discord integration is not connected yet",
	}
}

func TestExecutionServiceGoLiveIsolatesDestinationFailures(t *testing.T) {
	destRepo := newDestinationRepositoryStub()
	destRepo.listItems = []domain.Destination{
		{
			ID:         "discord-main",
			Platform:   domain.PlatformDiscord,
			Name:       "Main Discord",
			Enabled:    true,
			Template:   "{{stream_title}}",
			ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main","cardThumbnailURL":"https://example.com/card.png","endStreamEnabled":true,"endStreamTemplate":"Bye"}`,
		},
		{
			ID:         "bluesky-main",
			Platform:   domain.PlatformBluesky,
			Name:       "Main Bluesky",
			Enabled:    true,
			Template:   "{{stream_title}}",
			ConfigJSON: `{"accountIdentifier":"don.main","credentialKey":"bluesky/main"}`,
		},
	}
	settingsRepo := &settingsRepositoryStub{item: domain.DefaultAppSettings()}
	history := &postHistoryRepositoryStub{recordsByDestination: map[string][]ports.PostHistoryRecord{}}
	sessions := &liveNowSessionRepositoryExecutionStub{}
	discord := &discordPublisherStub{err: errors.New("discord failed")}
	bluesky := &blueskyPublisherStub{}
	mastodon := &mastodonPublisherStub{}
	service := NewExecutionService(destRepo, settingsRepo, history, sessions, discord, bluesky, mastodon)
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	summary, err := service.GoLive(context.Background(), domain.Announcement{
		StreamTitle: "Going Live",
		StreamURL:   "https://example.com/live",
	})
	if err != nil {
		t.Fatalf("go live: %v", err)
	}

	if len(summary.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(summary.Results))
	}
	if summary.Results[0].State != domain.ExecutionStateFailed {
		t.Fatalf("expected first result failed, got %+v", summary.Results[0])
	}
	if summary.Results[1].State != domain.ExecutionStateSuccess {
		t.Fatalf("expected second result success, got %+v", summary.Results[1])
	}
	if len(bluesky.calls) != 1 {
		t.Fatalf("expected bluesky publisher call, got %d", len(bluesky.calls))
	}
	if len(bluesky.liveNowSetCalls) != 1 {
		t.Fatalf("expected bluesky live now set call, got %d", len(bluesky.liveNowSetCalls))
	}
	if bluesky.liveNowSetCalls[0].status.DurationMinutes != defaultBlueskyLiveNowDurationMinutes {
		t.Fatalf("expected default live now duration, got %+v", bluesky.liveNowSetCalls[0].status)
	}
	if len(sessions.upserts) != 1 || sessions.upserts[0].DestinationID != "bluesky-main" {
		t.Fatalf("expected tracked live now session, got %+v", sessions.upserts)
	}
	if summary.TotalCount != 2 || summary.SuccessCount != 1 || summary.FailedCount != 1 || summary.ValidationErrorCount != 0 {
		t.Fatalf("unexpected go live summary counts: %+v", summary)
	}
	if summary.Status != domain.ExecutionSummaryStatusPartial {
		t.Fatalf("expected partial summary status, got %+v", summary)
	}
}

func TestExecutionServiceReturnsValidationErrorsPerDestination(t *testing.T) {
	destRepo := newDestinationRepositoryStub()
	destRepo.items["discord-main"] = domain.Destination{
		ID:         "discord-main",
		Platform:   domain.PlatformDiscord,
		Name:       "Main Discord",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"webhookKey":"not-a-url"}`,
	}
	settingsRepo := &settingsRepositoryStub{item: domain.DefaultAppSettings()}
	history := &postHistoryRepositoryStub{recordsByDestination: map[string][]ports.PostHistoryRecord{}}
	service := NewExecutionService(destRepo, settingsRepo, history, &liveNowSessionRepositoryExecutionStub{}, &discordPublisherStub{}, &blueskyPublisherStub{}, &mastodonPublisherStub{})
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	summary, err := service.GoLive(context.Background(), domain.Announcement{
		StreamTitle: "Going Live",
		StreamURL:   "https://example.com/live",
	})
	if err != nil {
		t.Fatalf("go live: %v", err)
	}
	if len(summary.Results) != 1 || summary.Results[0].State != domain.ExecutionStateValidationError {
		t.Fatalf("unexpected validation result: %+v", summary.Results)
	}
	if summary.Results[0].Message != "Discord webhook key must be a valid absolute URL" {
		t.Fatalf("expected config validation message, got %+v", summary.Results[0])
	}
	if summary.TotalCount != 1 || summary.ValidationErrorCount != 1 || summary.SuccessCount != 0 {
		t.Fatalf("unexpected validation summary counts: %+v", summary)
	}
	if summary.Status != domain.ExecutionSummaryStatusPartial {
		t.Fatalf("expected partial summary status, got %+v", summary)
	}
}

func TestExecutionServiceGoLiveReturnsDuplicateWarningBeforePublishing(t *testing.T) {
	destRepo := newDestinationRepositoryStub()
	destRepo.items["discord-main"] = domain.Destination{
		ID:         "discord-main",
		Platform:   domain.PlatformDiscord,
		Name:       "Main Discord",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main"}`,
	}
	settings := domain.DefaultAppSettings()
	settings.DuplicateProtectionEnabled = true
	settings.DuplicateWindowMinutes = 10
	settingsRepo := &settingsRepositoryStub{item: settings}
	history := &postHistoryRepositoryStub{
		recordsByDestination: map[string][]ports.PostHistoryRecord{
			"discord-main": {{
				DestinationID: "discord-main",
				ContentHash:   "c6d1d52d55994f8fd66bb81e4f9fab9a9796d7cb6f5e26e759c8ae21afee4ab5",
				PostedAt:      time.Date(2026, 5, 31, 17, 55, 0, 0, time.UTC),
			}},
		},
	}
	discord := &discordPublisherStub{}
	service := NewExecutionService(destRepo, settingsRepo, history, &liveNowSessionRepositoryExecutionStub{}, discord, &blueskyPublisherStub{}, &mastodonPublisherStub{})
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	summary, err := service.GoLive(context.Background(), domain.Announcement{
		StreamTitle: "Going Live",
		StreamURL:   "https://example.com/live",
	})
	if err != nil {
		t.Fatalf("go live: %v", err)
	}
	if !summary.RequiresDuplicateConfirmation {
		t.Fatalf("expected duplicate warning, got %+v", summary)
	}
	if len(summary.Results) != 0 {
		t.Fatalf("expected no result rows before confirmation, got %+v", summary.Results)
	}
	if summary.Status != domain.ExecutionSummaryStatusWarning {
		t.Fatalf("expected warning summary status, got %+v", summary)
	}
	if len(discord.calls) != 0 {
		t.Fatal("expected no publish calls before confirmation")
	}
}

func TestExecutionServiceGoLivePreservesValidationResultsWhenDuplicateConfirmationIsRequired(t *testing.T) {
	destRepo := newDestinationRepositoryStub()
	destRepo.listItems = []domain.Destination{
		{
			ID:         "discord-valid",
			Platform:   domain.PlatformDiscord,
			Name:       "Valid Discord",
			Enabled:    true,
			Template:   "{{stream_title}}",
			ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/valid"}`,
		},
		{
			ID:         "discord-invalid",
			Platform:   domain.PlatformDiscord,
			Name:       "Invalid Discord",
			Enabled:    true,
			Template:   "{{message}}",
			ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/invalid"}`,
		},
	}
	settings := domain.DefaultAppSettings()
	settings.DuplicateProtectionEnabled = true
	settings.DuplicateWindowMinutes = 10
	settingsRepo := &settingsRepositoryStub{item: settings}
	history := &postHistoryRepositoryStub{
		recordsByDestination: map[string][]ports.PostHistoryRecord{
			"discord-valid": {{
				DestinationID: "discord-valid",
				ContentHash:   "c6d1d52d55994f8fd66bb81e4f9fab9a9796d7cb6f5e26e759c8ae21afee4ab5",
				PostedAt:      time.Date(2026, 5, 31, 17, 55, 0, 0, time.UTC),
			}},
		},
	}
	service := NewExecutionService(destRepo, settingsRepo, history, &liveNowSessionRepositoryExecutionStub{}, &discordPublisherStub{}, &blueskyPublisherStub{}, &mastodonPublisherStub{})
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	summary, err := service.GoLive(context.Background(), domain.Announcement{
		StreamTitle: "Going Live",
		StreamURL:   "https://example.com/live",
	})
	if err != nil {
		t.Fatalf("go live: %v", err)
	}
	if !summary.RequiresDuplicateConfirmation {
		t.Fatalf("expected duplicate warning, got %+v", summary)
	}
	if len(summary.Results) != 1 || summary.Results[0].State != domain.ExecutionStateValidationError {
		t.Fatalf("expected validation result to be preserved, got %+v", summary.Results)
	}
	if summary.TotalCount != 1 || summary.ValidationErrorCount != 1 || summary.SuccessCount != 0 || summary.FailedCount != 0 {
		t.Fatalf("unexpected summary counts: %+v", summary)
	}
	if summary.Status != domain.ExecutionSummaryStatusPartial {
		t.Fatalf("expected partial summary status, got %+v", summary)
	}
}

func TestExecutionServiceGoLiveMarksUnavailableIntegrationsAsSkipped(t *testing.T) {
	destRepo := newDestinationRepositoryStub()
	destRepo.items["discord-main"] = domain.Destination{
		ID:         "discord-main",
		Platform:   domain.PlatformDiscord,
		Name:       "Main Discord",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main"}`,
	}
	settingsRepo := &settingsRepositoryStub{item: domain.DefaultAppSettings()}
	history := &postHistoryRepositoryStub{recordsByDestination: map[string][]ports.PostHistoryRecord{}}
	service := NewExecutionService(destRepo, settingsRepo, history, &liveNowSessionRepositoryExecutionStub{}, unavailableDiscordPublisherStub{}, &blueskyPublisherStub{}, &mastodonPublisherStub{})
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	summary, err := service.GoLive(context.Background(), domain.Announcement{
		StreamTitle: "Going Live",
		StreamURL:   "https://example.com/live",
	})
	if err != nil {
		t.Fatalf("go live: %v", err)
	}
	if len(summary.Results) != 1 || summary.Results[0].State != domain.ExecutionStateSkipped {
		t.Fatalf("expected skipped result, got %+v", summary.Results)
	}
	if summary.TotalCount != 1 || summary.SkippedCount != 1 || summary.FailedCount != 0 || summary.SuccessCount != 0 {
		t.Fatalf("unexpected skipped summary counts: %+v", summary)
	}
	if summary.Status != domain.ExecutionSummaryStatusWarning {
		t.Fatalf("expected warning summary status, got %+v", summary)
	}
	if len(history.recordCalls) != 0 {
		t.Fatalf("expected no post history records for skipped integrations, got %d", len(history.recordCalls))
	}
}

func TestExecutionServiceGoLiveUsesSelectedDestinationTargets(t *testing.T) {
	destRepo := newDestinationRepositoryStub()
	destRepo.listItems = []domain.Destination{
		{
			ID:         "discord-main",
			Platform:   domain.PlatformDiscord,
			Name:       "Main Discord",
			Enabled:    true,
			Template:   "{{stream_title}}",
			ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main","cardThumbnailURL":"https://example.com/card.png","endStreamEnabled":true,"endStreamTemplate":"Bye"}`,
		},
		{
			ID:         "bluesky-main",
			Platform:   domain.PlatformBluesky,
			Name:       "Main Bluesky",
			Enabled:    true,
			Template:   "{{stream_title}}",
			ConfigJSON: `{"accountIdentifier":"don.main","credentialKey":"bluesky/main","cardThumbnailURL":"https://example.com/avatar.png","liveNowDurationMinutes":90}`,
		},
		{
			ID:         "mastodon-main",
			Platform:   domain.PlatformMastodon,
			Name:       "Main Mastodon",
			Enabled:    true,
			Template:   "{{stream_title}}",
			ConfigJSON: `{"credentialKey":"mastodon/main","instanceURL":"https://mastodon.social"}`,
		},
	}
	settingsRepo := &settingsRepositoryStub{item: domain.DefaultAppSettings()}
	history := &postHistoryRepositoryStub{recordsByDestination: map[string][]ports.PostHistoryRecord{}}
	sessions := &liveNowSessionRepositoryExecutionStub{}
	discord := &discordPublisherStub{}
	bluesky := &blueskyPublisherStub{}
	mastodon := &mastodonPublisherStub{}
	service := NewExecutionService(destRepo, settingsRepo, history, sessions, discord, bluesky, mastodon)
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	summary, err := service.GoLive(context.Background(), domain.Announcement{
		StreamTitle: "Going Live",
		StreamURL:   "https://example.com/live",
	})
	if err != nil {
		t.Fatalf("go live: %v", err)
	}
	if len(discord.calls) != 1 || discord.calls[0].key != "https://discord.com/api/webhooks/123/main" {
		t.Fatalf("expected selected discord key, got %+v", discord.calls)
	}
	if discord.calls[0].metadata.ThumbnailURL != "https://example.com/card.png" {
		t.Fatalf("expected discord post metadata, got %+v", discord.calls[0].metadata)
	}
	if len(bluesky.calls) != 1 || bluesky.calls[0].identifier != "don.main" || bluesky.calls[0].key != "bluesky/main" {
		t.Fatalf("expected selected bluesky key, got %+v", bluesky.calls)
	}
	if bluesky.calls[0].metadata.StreamURL != "https://example.com/live" || bluesky.calls[0].metadata.StreamTitle != "Going Live" || bluesky.calls[0].metadata.ThumbnailURL != "https://example.com/avatar.png" {
		t.Fatalf("expected bluesky post metadata, got %+v", bluesky.calls[0].metadata)
	}
	if len(bluesky.liveNowSetCalls) != 1 || bluesky.liveNowSetCalls[0].identifier != "don.main" || bluesky.liveNowSetCalls[0].key != "bluesky/main" {
		t.Fatalf("expected bluesky live now to use selected key, got %+v", bluesky.liveNowSetCalls)
	}
	if bluesky.liveNowSetCalls[0].status.DurationMinutes != 90 {
		t.Fatalf("expected configured live now duration, got %+v", bluesky.liveNowSetCalls[0].status)
	}
	if len(sessions.upserts) != 1 || sessions.upserts[0].AccountIdentifier != "don.main" || sessions.upserts[0].CredentialKey != "bluesky/main" {
		t.Fatalf("expected tracked session to use selected destination, got %+v", sessions.upserts)
	}
	if len(mastodon.calls) != 1 || mastodon.calls[0].key != "mastodon/main" || mastodon.calls[0].instanceURL != "https://mastodon.social" {
		t.Fatalf("expected selected mastodon target, got %+v", mastodon.calls)
	}
	if len(summary.Results) != 3 {
		t.Fatalf("expected 3 results, got %+v", summary.Results)
	}
	if summary.TestModeActive {
		t.Fatalf("expected legacy summary flag to stay false, got %+v", summary)
	}
	if summary.Status != domain.ExecutionSummaryStatusSuccess {
		t.Fatalf("expected success summary status, got %+v", summary)
	}
	for _, result := range summary.Results {
		if result.State != domain.ExecutionStateSuccess {
			t.Fatalf("expected success results, got %+v", summary.Results)
		}
		if result.Platform == domain.PlatformBluesky {
			if result.Message != "Published successfully. Live Now set." {
				t.Fatalf("expected bluesky live now success message, got %+v", summary.Results)
			}
			continue
		}
		if result.Message != "Published successfully." {
			t.Fatalf("expected publish success message, got %+v", summary.Results)
		}
	}
}

func TestExecutionServiceGoLiveReturnsFailureWhenBlueskyLiveNowUpdateFailsAfterPost(t *testing.T) {
	destRepo := newDestinationRepositoryStub()
	destRepo.items["bluesky-main"] = domain.Destination{
		ID:         "bluesky-main",
		Platform:   domain.PlatformBluesky,
		Name:       "Main Bluesky",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"accountIdentifier":"don.main","credentialKey":"bluesky/main"}`,
	}
	settingsRepo := &settingsRepositoryStub{item: domain.DefaultAppSettings()}
	history := &postHistoryRepositoryStub{recordsByDestination: map[string][]ports.PostHistoryRecord{}}
	bluesky := &blueskyPublisherStub{liveNowSetErr: errors.New("live now denied")}
	sessions := &liveNowSessionRepositoryExecutionStub{}
	service := NewExecutionService(destRepo, settingsRepo, history, sessions, &discordPublisherStub{}, bluesky, &mastodonPublisherStub{})
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	summary, err := service.GoLive(context.Background(), domain.Announcement{
		StreamTitle: "Going Live",
		StreamURL:   "https://example.com/live",
		Message:     "See you there",
	})
	if err != nil {
		t.Fatalf("go live: %v", err)
	}
	if len(summary.Results) != 1 || summary.Results[0].State != domain.ExecutionStateFailed {
		t.Fatalf("expected failed result, got %+v", summary.Results)
	}
	if summary.Results[0].Message != "Published successfully, but Live Now update failed: live now denied" {
		t.Fatalf("unexpected message: %+v", summary.Results[0])
	}
	if len(history.recordCalls) != 1 {
		t.Fatalf("expected history record after successful publish, got %d", len(history.recordCalls))
	}
	if len(sessions.upserts) != 0 {
		t.Fatalf("expected no live now tracking on set failure, got %+v", sessions.upserts)
	}
}

func TestExecutionServiceEndStreamClearsBlueskyLiveNow(t *testing.T) {
	destRepo := newDestinationRepositoryStub()
	destRepo.listItems = []domain.Destination{
		{
			ID:         "bluesky-main",
			Platform:   domain.PlatformBluesky,
			Name:       "Main Bluesky",
			Enabled:    true,
			Template:   "{{stream_title}}",
			ConfigJSON: `{"accountIdentifier":"don.main","credentialKey":"bluesky/main"}`,
		},
		{
			ID:         "discord-main",
			Platform:   domain.PlatformDiscord,
			Name:       "Main Discord",
			Enabled:    true,
			Template:   "{{stream_title}}",
			ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main","endStreamEnabled":false,"endStreamTemplate":"Discord should not post"}`,
		},
	}
	settingsRepo := &settingsRepositoryStub{item: domain.DefaultAppSettings()}
	bluesky := &blueskyPublisherStub{}
	sessions := &liveNowSessionRepositoryExecutionStub{
		items: []domain.ActiveLiveNowSession{{
			DestinationID:     "bluesky-main",
			DestinationName:   "Main Bluesky",
			Platform:          string(domain.PlatformBluesky),
			AccountIdentifier: "don.main",
			CredentialKey:     "bluesky/main",
			StreamURL:         "https://example.com/live",
			StreamTitle:       "Going Live",
			StartedAt:         time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC),
		}},
	}
	service := NewExecutionService(destRepo, settingsRepo, &postHistoryRepositoryStub{recordsByDestination: map[string][]ports.PostHistoryRecord{}}, sessions, &discordPublisherStub{}, bluesky, &mastodonPublisherStub{})

	summary, err := service.EndStream(context.Background(), domain.Announcement{})
	if err != nil {
		t.Fatalf("end stream: %v", err)
	}
	if len(bluesky.liveNowClearCalls) != 1 || bluesky.liveNowClearCalls[0].identifier != "don.main" || bluesky.liveNowClearCalls[0].key != "bluesky/main" {
		t.Fatalf("expected live now clear call, got %+v", bluesky.liveNowClearCalls)
	}
	if len(summary.Results) != 3 || summary.Results[0].State != domain.ExecutionStateSuccess || summary.SkippedCount != 2 {
		t.Fatalf("expected live now clear plus disabled post skips, got %+v", summary)
	}
	if len(sessions.deletes) != 1 || sessions.deletes[0] != "bluesky-main" {
		t.Fatalf("expected live now session delete, got %+v", sessions.deletes)
	}
}

func TestExecutionServiceEndStreamOptionallyPublishesConfiguredPosts(t *testing.T) {
	destRepo := newDestinationRepositoryStub()
	destRepo.listItems = []domain.Destination{
		{
			ID:         "bluesky-main",
			Platform:   domain.PlatformBluesky,
			Name:       "Main Bluesky",
			Enabled:    true,
			Template:   "{{stream_title}}",
			ConfigJSON: `{"accountIdentifier":"don.main","credentialKey":"bluesky/main","endStreamEnabled":true,"endStreamTemplate":"Thanks for hanging out at {{stream_title}} {{stream_url}}"}`,
		},
		{
			ID:         "discord-main",
			Platform:   domain.PlatformDiscord,
			Name:       "Main Discord",
			Enabled:    true,
			Template:   "{{stream_title}}",
			ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main","endStreamEnabled":false,"endStreamTemplate":"Discord should not post"}`,
		},
	}
	settings := domain.DefaultAppSettings()
	settingsRepo := &settingsRepositoryStub{item: settings}
	discord := &discordPublisherStub{}
	bluesky := &blueskyPublisherStub{}
	sessions := &liveNowSessionRepositoryExecutionStub{
		items: []domain.ActiveLiveNowSession{{
			DestinationID:     "bluesky-main",
			DestinationName:   "Main Bluesky",
			Platform:          string(domain.PlatformBluesky),
			AccountIdentifier: "don.main",
			CredentialKey:     "bluesky/main",
			StreamURL:         "https://example.com/live",
			StreamTitle:       "Going Live",
			StartedAt:         time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC),
		}},
	}
	service := NewExecutionService(destRepo, settingsRepo, &postHistoryRepositoryStub{recordsByDestination: map[string][]ports.PostHistoryRecord{}}, sessions, discord, bluesky, &mastodonPublisherStub{})
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 19, 0, 0, 0, time.UTC)}

	summary, err := service.EndStream(context.Background(), domain.Announcement{})
	if err != nil {
		t.Fatalf("end stream: %v", err)
	}
	if len(bluesky.liveNowClearCalls) != 1 {
		t.Fatalf("expected one live now clear call, got %+v", bluesky.liveNowClearCalls)
	}
	if len(discord.calls) != 0 {
		t.Fatalf("expected disabled discord end stream post to be skipped, got %+v", discord.calls)
	}
	if len(bluesky.calls) != 1 || bluesky.calls[0].content != "Thanks for hanging out at Going Live https://example.com/live" {
		t.Fatalf("expected bluesky end stream post with tracked session details, got %+v", bluesky.calls)
	}
	if len(summary.Results) != 3 {
		t.Fatalf("expected three end stream results, got %+v", summary.Results)
	}
	if summary.SkippedCount != 1 {
		t.Fatalf("expected one skipped disabled destination, got %+v", summary)
	}
}

func TestExecutionServiceEndStreamUsesSelectedDestinationTargets(t *testing.T) {
	destRepo := newDestinationRepositoryStub()
	destRepo.listItems = []domain.Destination{
		{
			ID:         "bluesky-main",
			Platform:   domain.PlatformBluesky,
			Name:       "Main Bluesky",
			Enabled:    true,
			Template:   "{{stream_title}}",
			ConfigJSON: `{"accountIdentifier":"don.main","credentialKey":"bluesky/main"}`,
		},
		{
			ID:         "discord-main",
			Platform:   domain.PlatformDiscord,
			Name:       "Main Discord",
			Enabled:    true,
			Template:   "{{stream_title}}",
			ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main","cardThumbnailURL":"https://example.com/card.png","endStreamEnabled":true,"endStreamTemplate":"Bye"}`,
		},
		{
			ID:         "discord-other",
			Platform:   domain.PlatformDiscord,
			Name:       "Other Discord",
			Enabled:    true,
			Template:   "{{stream_title}}",
			ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/other"}`,
		},
	}
	settings := domain.DefaultAppSettings()
	settingsRepo := &settingsRepositoryStub{item: settings}
	discord := &discordPublisherStub{}
	bluesky := &blueskyPublisherStub{}
	sessions := &liveNowSessionRepositoryExecutionStub{
		items: []domain.ActiveLiveNowSession{{
			DestinationID:     "bluesky-main",
			DestinationName:   "Main Bluesky",
			Platform:          string(domain.PlatformBluesky),
			AccountIdentifier: "don.main",
			CredentialKey:     "bluesky/main",
			StreamURL:         "https://example.com/live",
			StreamTitle:       "Going Live",
			StartedAt:         time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC),
		}},
	}
	service := NewExecutionService(destRepo, settingsRepo, &postHistoryRepositoryStub{recordsByDestination: map[string][]ports.PostHistoryRecord{}}, sessions, discord, bluesky, &mastodonPublisherStub{})

	summary, err := service.EndStream(context.Background(), domain.Announcement{
		DestinationIDs: []string{"discord-main"},
	})
	if err != nil {
		t.Fatalf("end stream: %v", err)
	}
	if len(bluesky.liveNowClearCalls) != 0 || len(bluesky.calls) != 0 {
		t.Fatalf("expected unselected bluesky to be untouched, clear=%+v posts=%+v", bluesky.liveNowClearCalls, bluesky.calls)
	}
	if len(discord.calls) != 1 || discord.calls[0].key != "https://discord.com/api/webhooks/123/main" {
		t.Fatalf("expected selected discord only, got %+v", discord.calls)
	}
	if discord.calls[0].metadata.ThumbnailURL != "https://example.com/card.png" {
		t.Fatalf("expected discord thumbnail metadata, got %+v", discord.calls[0].metadata)
	}
	if len(summary.Results) != 1 {
		t.Fatalf("expected one selected end stream result, got %+v", summary.Results)
	}
}

func TestExecutionServiceEndStreamDoesNotClearWhenNoTrackedLiveNowSessionExists(t *testing.T) {
	destRepo := newDestinationRepositoryStub()
	destRepo.listItems = []domain.Destination{
		{
			ID:         "bluesky-main",
			Platform:   domain.PlatformBluesky,
			Name:       "Main Bluesky",
			Enabled:    true,
			Template:   "{{stream_title}}",
			ConfigJSON: `{"accountIdentifier":"don.main","credentialKey":"bluesky/main"}`,
		},
	}
	settingsRepo := &settingsRepositoryStub{item: domain.DefaultAppSettings()}
	bluesky := &blueskyPublisherStub{}
	sessions := &liveNowSessionRepositoryExecutionStub{}
	service := NewExecutionService(destRepo, settingsRepo, &postHistoryRepositoryStub{recordsByDestination: map[string][]ports.PostHistoryRecord{}}, sessions, &discordPublisherStub{}, bluesky, &mastodonPublisherStub{})

	summary, err := service.EndStream(context.Background(), domain.Announcement{})
	if err != nil {
		t.Fatalf("end stream: %v", err)
	}
	if len(bluesky.liveNowClearCalls) != 0 {
		t.Fatalf("expected no live now clear call, got %+v", bluesky.liveNowClearCalls)
	}
	if len(summary.Results) != 1 || summary.Results[0].State != domain.ExecutionStateSkipped {
		t.Fatalf("expected disabled end stream message skip without a tracked session, got %+v", summary.Results)
	}
}

func TestExecutionServiceForceGoLivePublishesAndRecordsHistory(t *testing.T) {
	destRepo := newDestinationRepositoryStub()
	destRepo.items["discord-main"] = domain.Destination{
		ID:         "discord-main",
		Platform:   domain.PlatformDiscord,
		Name:       "Main Discord",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main"}`,
	}
	settings := domain.DefaultAppSettings()
	settings.DuplicateProtectionEnabled = true
	settings.DuplicateWindowMinutes = 10
	settingsRepo := &settingsRepositoryStub{item: settings}
	history := &postHistoryRepositoryStub{recordsByDestination: map[string][]ports.PostHistoryRecord{}}
	discord := &discordPublisherStub{}
	service := NewExecutionService(destRepo, settingsRepo, history, &liveNowSessionRepositoryExecutionStub{}, discord, &blueskyPublisherStub{}, &mastodonPublisherStub{})
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	summary, err := service.ForceGoLive(context.Background(), domain.Announcement{
		StreamTitle: "Going Live",
		StreamURL:   "https://example.com/live",
	})
	if err != nil {
		t.Fatalf("force go live: %v", err)
	}
	if len(summary.Results) != 1 || summary.Results[0].State != domain.ExecutionStateSuccess {
		t.Fatalf("unexpected results: %+v", summary.Results)
	}
	if len(discord.calls) != 1 {
		t.Fatalf("expected one publish call, got %d", len(discord.calls))
	}
	if len(history.recordCalls) != 1 {
		t.Fatalf("expected one post history record, got %d", len(history.recordCalls))
	}
	if summary.TotalCount != 1 || summary.SuccessCount != 1 || summary.FailedCount != 0 || summary.ValidationErrorCount != 0 {
		t.Fatalf("unexpected force go live summary counts: %+v", summary)
	}
	if summary.Status != domain.ExecutionSummaryStatusSuccess {
		t.Fatalf("expected success summary status, got %+v", summary)
	}
}
