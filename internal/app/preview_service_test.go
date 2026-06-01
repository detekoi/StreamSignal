package app

import (
	"context"
	"slices"
	"testing"
	"time"

	"StreamSignal/internal/domain"
)

type destinationListStub struct {
	items []domain.Destination
}

func (s *destinationListStub) List(context.Context) ([]domain.Destination, error) {
	return append([]domain.Destination(nil), s.items...), nil
}

func (s *destinationListStub) Get(context.Context, string) (domain.Destination, error) {
	panic("unexpected Get call")
}

func (s *destinationListStub) Save(context.Context, domain.Destination) error {
	panic("unexpected Save call")
}

func (s *destinationListStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

type settingsLoadStub struct {
	item domain.AppSettings
}

func (s *settingsLoadStub) Load(context.Context) (domain.AppSettings, error) {
	return s.item, nil
}

func (s *settingsLoadStub) Save(context.Context, domain.AppSettings) error {
	panic("unexpected Save call")
}

type previewSessionRepositoryStub struct {
	items []domain.ActiveLiveNowSession
}

func (s *previewSessionRepositoryStub) Upsert(context.Context, domain.ActiveLiveNowSession) error {
	panic("unexpected Upsert call")
}

func (s *previewSessionRepositoryStub) List(context.Context) ([]domain.ActiveLiveNowSession, error) {
	return append([]domain.ActiveLiveNowSession(nil), s.items...), nil
}

func (s *previewSessionRepositoryStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestPreviewServiceGeneratesItemsForSelectedDestinations(t *testing.T) {
	destinations := &destinationListStub{
		items: []domain.Destination{
			{
				ID:         "discord-main",
				Platform:   domain.PlatformDiscord,
				Name:       "Main Discord",
				Template:   "{{stream_title}} {{stream_url}} {{hashtags}}",
				ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main"}`,
			},
			{
				ID:         "bsky-disabled",
				Platform:   domain.PlatformBluesky,
				Name:       "Secondary Bluesky",
				Template:   "{{stream_title}}",
				ConfigJSON: `{"accountIdentifier":"don.test","credentialKey":"bluesky/main"}`,
			},
		},
	}
	settings := &settingsLoadStub{
		item: domain.AppSettings{
			DefaultStreamURL: "https://example.com/live",
			DefaultHashtags:  "#default",
		},
	}
	service := NewPreviewService(destinations, settings, &previewSessionRepositoryStub{})
	now := time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)
	service.clock = fixedClock{now: now}

	items, err := service.Generate(context.Background(), domain.Announcement{
		StreamTitle:    "Going Live",
		DestinationIDs: []string{"discord-main"},
	})
	if err != nil {
		t.Fatalf("generate preview: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 preview item, got %d", len(items))
	}

	item := items[0]
	if item.Content != "Going Live https://example.com/live #default" {
		t.Fatalf("unexpected preview content %q", item.Content)
	}
	if item.ValidationState != domain.PreviewValidationValid {
		t.Fatalf("expected valid preview state, got %q with notes %v", item.ValidationState, item.ValidationNotes)
	}
}

func TestPreviewServiceCarriesValidationNotesIntoPreviewItems(t *testing.T) {
	destinations := &destinationListStub{
		items: []domain.Destination{
			{
				ID:         "bluesky-main",
				Platform:   domain.PlatformBluesky,
				Name:       "Main Bluesky",
				Template:   "{{message}}",
				ConfigJSON: `{"accountIdentifier":"don.test","credentialKey":"bluesky/main"}`,
			},
		},
	}
	settings := &settingsLoadStub{}
	service := NewPreviewService(destinations, settings, &previewSessionRepositoryStub{})
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	items, err := service.Generate(context.Background(), domain.Announcement{})
	if err != nil {
		t.Fatalf("generate preview: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 preview item, got %d", len(items))
	}

	item := items[0]
	if item.ValidationState != domain.PreviewValidationInvalid {
		t.Fatalf("expected invalid preview state, got %q", item.ValidationState)
	}
	if !slices.Contains(item.ValidationNotes, "Message is required.") {
		t.Fatalf("expected message validation note, got %v", item.ValidationNotes)
	}
	if slices.Contains(item.ValidationNotes, "Stream title is required.") || slices.Contains(item.ValidationNotes, "Stream URL is required.") {
		t.Fatalf("expected unused title and url fields to be ignored, got %v", item.ValidationNotes)
	}
}

func TestPreviewServiceIncludesEnabledEndStreamMessages(t *testing.T) {
	destinations := &destinationListStub{
		items: []domain.Destination{
			{
				ID:         "bluesky-main",
				Platform:   domain.PlatformBluesky,
				Name:       "Main Bluesky",
				Template:   "{{stream_title}} {{stream_url}}",
				ConfigJSON: `{"accountIdentifier":"don.test","credentialKey":"bluesky/main","endStreamEnabled":true,"endStreamTemplate":"Thanks for watching {{stream_title}}"}`,
			},
		},
	}
	settings := &settingsLoadStub{}
	sessions := &previewSessionRepositoryStub{
		items: []domain.ActiveLiveNowSession{{
			DestinationID: "bluesky-main",
			StreamTitle:   "Going Live",
			StreamURL:     "https://example.com/live",
		}},
	}
	service := NewPreviewService(destinations, settings, sessions)
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	items, err := service.Generate(context.Background(), domain.Announcement{
		StreamTitle:    "Going Live",
		StreamURL:      "https://example.com/live",
		DestinationIDs: []string{"bluesky-main"},
	})
	if err != nil {
		t.Fatalf("generate preview: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected go live and end stream previews, got %d: %+v", len(items), items)
	}
	if items[0].PreviewLabel != "Go Live" || items[0].Content != "Going Live https://example.com/live" {
		t.Fatalf("unexpected go live preview: %+v", items[0])
	}
	if items[1].PreviewLabel != "End Stream" || items[1].Content != "Thanks for watching Going Live" {
		t.Fatalf("unexpected end stream preview: %+v", items[1])
	}
}

func TestPreviewServiceIncludesDestinationSetupValidation(t *testing.T) {
	destinations := &destinationListStub{
		items: []domain.Destination{
			{
				ID:         "discord-main",
				Platform:   domain.PlatformDiscord,
				Name:       "Main Discord",
				Template:   "{{stream_title}}",
				ConfigJSON: `{"webhookKey":"not a url"}`,
			},
		},
	}
	settings := &settingsLoadStub{}
	service := NewPreviewService(destinations, settings, &previewSessionRepositoryStub{})
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	items, err := service.Generate(context.Background(), domain.Announcement{
		StreamTitle: "Going Live",
		StreamURL:   "https://example.com/live",
	})
	if err != nil {
		t.Fatalf("generate preview: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 preview item, got %d", len(items))
	}

	item := items[0]
	if item.ValidationState != domain.PreviewValidationInvalid {
		t.Fatalf("expected invalid preview state, got %q", item.ValidationState)
	}
	if len(item.ValidationNotes) != 1 || item.ValidationNotes[0] != "Discord webhook key must be a valid absolute URL" {
		t.Fatalf("expected destination setup validation note, got %v", item.ValidationNotes)
	}
}

func TestPreviewServiceGeneratesEndStreamItemsFromDestinationConfig(t *testing.T) {
	destinations := &destinationListStub{
		items: []domain.Destination{
			{
				ID:         "bluesky-main",
				Platform:   domain.PlatformBluesky,
				Name:       "Main Bluesky",
				Template:   "{{stream_title}}",
				ConfigJSON: `{"accountIdentifier":"don.test","credentialKey":"bluesky/main","endStreamEnabled":true,"endStreamTemplate":"Thanks for watching {{stream_title}} {{stream_url}}"}`,
			},
			{
				ID:         "discord-main",
				Platform:   domain.PlatformDiscord,
				Name:       "Main Discord",
				Template:   "{{stream_title}}",
				ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main","endStreamEnabled":false,"endStreamTemplate":"Bye"}`,
			},
		},
	}
	settings := &settingsLoadStub{}
	sessions := &previewSessionRepositoryStub{
		items: []domain.ActiveLiveNowSession{{
			DestinationID: "bluesky-main",
			StreamTitle:   "Going Live",
			StreamURL:     "https://example.com/live",
		}},
	}
	service := NewPreviewService(destinations, settings, sessions)
	service.clock = fixedClock{now: time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)}

	items, err := service.GenerateEndStream(context.Background(), domain.Announcement{
		DestinationIDs: []string{"bluesky-main", "discord-main"},
	})
	if err != nil {
		t.Fatalf("generate end stream preview: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 preview items, got %d", len(items))
	}
	if items[0].Content != "Thanks for watching Going Live https://example.com/live" {
		t.Fatalf("unexpected end stream content %q", items[0].Content)
	}
	if !slices.Contains(items[1].ValidationNotes, "End Stream message disabled for this destination.") {
		t.Fatalf("expected disabled note, got %v", items[1].ValidationNotes)
	}
}
