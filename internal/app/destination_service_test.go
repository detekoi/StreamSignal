package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"StreamSignal/internal/domain"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type destinationRepositoryStub struct {
	listItems []domain.Destination
	items     map[string]domain.Destination
	saveErr   error
	deleteErr error
}

func newDestinationRepositoryStub() *destinationRepositoryStub {
	return &destinationRepositoryStub{
		items: make(map[string]domain.Destination),
	}
}

func (s *destinationRepositoryStub) List(context.Context) ([]domain.Destination, error) {
	if s.listItems != nil {
		return append([]domain.Destination(nil), s.listItems...), nil
	}

	items := make([]domain.Destination, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func (s *destinationRepositoryStub) Get(_ context.Context, id string) (domain.Destination, error) {
	item, ok := s.items[id]
	if !ok {
		return domain.Destination{}, sql.ErrNoRows
	}
	return item, nil
}

func (s *destinationRepositoryStub) Save(_ context.Context, destination domain.Destination) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.items[destination.ID] = destination
	return nil
}

func (s *destinationRepositoryStub) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	delete(s.items, id)
	return nil
}

func TestDestinationServiceSaveAssignsIDAndTimestamps(t *testing.T) {
	repository := newDestinationRepositoryStub()
	service := NewDestinationService(repository)
	now := time.Date(2026, 5, 31, 6, 0, 0, 0, time.UTC)
	service.clock = fixedClock{now: now}

	saved, err := service.Save(context.Background(), domain.Destination{
		Platform:   domain.PlatformDiscord,
		Name:       "Main Discord",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main"}`,
	})
	if err != nil {
		t.Fatalf("save destination: %v", err)
	}

	if saved.ID == "" {
		t.Fatal("expected generated destination id")
	}
	if !saved.CreatedAt.Equal(now) {
		t.Fatalf("expected created at %v, got %v", now, saved.CreatedAt)
	}
	if !saved.UpdatedAt.Equal(now) {
		t.Fatalf("expected updated at %v, got %v", now, saved.UpdatedAt)
	}
}

func TestDestinationServiceSavePreservesCreatedAtForExistingRecord(t *testing.T) {
	repository := newDestinationRepositoryStub()
	createdAt := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	repository.items["discord-main"] = domain.Destination{
		ID:         "discord-main",
		Platform:   domain.PlatformDiscord,
		Name:       "Main Discord",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main"}`,
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
	}

	service := NewDestinationService(repository)
	now := time.Date(2026, 5, 31, 6, 0, 0, 0, time.UTC)
	service.clock = fixedClock{now: now}

	saved, err := service.Save(context.Background(), domain.Destination{
		ID:         "discord-main",
		Platform:   domain.PlatformDiscord,
		Name:       "Main Discord Updated",
		Enabled:    false,
		Template:   "{{stream_title}} {{message}}",
		ConfigJSON: `{"webhookKey":"https://discord.com/api/webhooks/123/main"}`,
	})
	if err != nil {
		t.Fatalf("save destination: %v", err)
	}

	if !saved.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected created at %v, got %v", createdAt, saved.CreatedAt)
	}
	if !saved.UpdatedAt.Equal(now) {
		t.Fatalf("expected updated at %v, got %v", now, saved.UpdatedAt)
	}
}

func TestDestinationServiceSaveValidatesRequiredFields(t *testing.T) {
	repository := newDestinationRepositoryStub()
	service := NewDestinationService(repository)

	_, err := service.Save(context.Background(), domain.Destination{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestDestinationServiceSaveValidatesDiscordWebhookURL(t *testing.T) {
	repository := newDestinationRepositoryStub()
	service := NewDestinationService(repository)

	_, err := service.Save(context.Background(), domain.Destination{
		Platform:   domain.PlatformDiscord,
		Name:       "Main Discord",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"webhookKey":"not-a-url"}`,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestDestinationServiceSaveValidatesBlueskyConfig(t *testing.T) {
	repository := newDestinationRepositoryStub()
	service := NewDestinationService(repository)

	_, err := service.Save(context.Background(), domain.Destination{
		Platform:   domain.PlatformBluesky,
		Name:       "Main Bluesky",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"credentialKey":"bluesky/main"}`,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestDestinationServiceSaveValidatesMastodonInstanceURL(t *testing.T) {
	repository := newDestinationRepositoryStub()
	service := NewDestinationService(repository)

	_, err := service.Save(context.Background(), domain.Destination{
		Platform:   domain.PlatformMastodon,
		Name:       "Main Mastodon",
		Enabled:    true,
		Template:   "{{stream_title}}",
		ConfigJSON: `{"credentialKey":"mastodon/main","instanceURL":"bad-url"}`,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestDestinationServiceDeleteRequiresID(t *testing.T) {
	repository := newDestinationRepositoryStub()
	service := NewDestinationService(repository)

	err := service.Delete(context.Background(), " ")
	if err == nil {
		t.Fatal("expected delete validation error")
	}
}

func TestDestinationServiceDeletePassesRepositoryError(t *testing.T) {
	repository := newDestinationRepositoryStub()
	repository.deleteErr = errors.New("boom")
	service := NewDestinationService(repository)

	err := service.Delete(context.Background(), "discord-main")
	if !errors.Is(err, repository.deleteErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}
