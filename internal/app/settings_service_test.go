package app

import (
	"context"
	"errors"
	"testing"

	"StreamSignal/internal/domain"
)

type settingsRepositoryStub struct {
	item    domain.AppSettings
	loadErr error
	saveErr error
}

func (s *settingsRepositoryStub) Load(context.Context) (domain.AppSettings, error) {
	if s.loadErr != nil {
		return domain.AppSettings{}, s.loadErr
	}
	return s.item, nil
}

func (s *settingsRepositoryStub) Save(_ context.Context, settings domain.AppSettings) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.item = settings
	return nil
}

func TestSettingsServiceLoadDelegatesToRepository(t *testing.T) {
	expected := domain.DefaultAppSettings()
	expected.DefaultStreamURL = "https://example.com/live"

	repository := &settingsRepositoryStub{item: expected}
	service := NewSettingsService(repository)

	actual, err := service.Load(context.Background())
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if actual != expected {
		t.Fatalf("expected settings %+v, got %+v", expected, actual)
	}
}

func TestSettingsServiceSaveValidatesDuplicateWindow(t *testing.T) {
	repository := &settingsRepositoryStub{}
	service := NewSettingsService(repository)

	_, err := service.Save(context.Background(), domain.AppSettings{
		DuplicateProtectionEnabled: true,
		DuplicateWindowMinutes:     0,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSettingsServiceSavePassesRepositoryError(t *testing.T) {
	repository := &settingsRepositoryStub{saveErr: errors.New("boom")}
	service := NewSettingsService(repository)

	_, err := service.Save(context.Background(), domain.DefaultAppSettings())
	if !errors.Is(err, repository.saveErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestSettingsServiceSaveReturnsPersistedSettings(t *testing.T) {
	repository := &settingsRepositoryStub{}
	service := NewSettingsService(repository)
	expected := domain.AppSettings{
		DefaultStreamURL:           "https://example.com/live",
		DefaultHashtags:            "#vtuber",
		DuplicateProtectionEnabled: true,
		DuplicateWindowMinutes:     12,
	}

	actual, err := service.Save(context.Background(), expected)
	if err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if actual != expected {
		t.Fatalf("expected settings %+v, got %+v", expected, actual)
	}
	if repository.item != expected {
		t.Fatalf("expected repository settings %+v, got %+v", expected, repository.item)
	}
}
