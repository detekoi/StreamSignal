package app

import (
	"context"
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
	service := NewPreviewService(destinations, settings)
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
	service := NewPreviewService(destinations, settings)
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
	if len(item.ValidationNotes) < 2 {
		t.Fatalf("expected validation notes, got %v", item.ValidationNotes)
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
	service := NewPreviewService(destinations, settings)
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
