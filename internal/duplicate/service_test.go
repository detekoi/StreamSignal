package duplicate

import (
	"context"
	"testing"
	"time"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/ports"
)

type historyStub struct {
	records map[string][]ports.PostHistoryRecord
}

func (s *historyStub) Record(context.Context, string, string, string, time.Time) error {
	return nil
}

func (s *historyStub) FindRecentByDestination(_ context.Context, destinationID string, _ time.Time) ([]ports.PostHistoryRecord, error) {
	return append([]ports.PostHistoryRecord(nil), s.records[destinationID]...), nil
}

func TestCheckReturnsWarningForMatchingHash(t *testing.T) {
	now := time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)
	service := NewService(&historyStub{
		records: map[string][]ports.PostHistoryRecord{
			"discord-main": {{
				DestinationID: "discord-main",
				ContentHash:   HashContent("Going Live"),
				PostedAt:      now.Add(-5 * time.Minute),
			}},
		},
	})

	warning, err := service.Check(context.Background(), domain.AppSettings{
		DuplicateProtectionEnabled: true,
		DuplicateWindowMinutes:     10,
	}, map[domain.Destination]string{
		{ID: "discord-main"}: "Going Live",
	}, now)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if warning == nil {
		t.Fatal("expected duplicate warning")
	}
}

func TestCheckReturnsNilWhenProtectionDisabled(t *testing.T) {
	service := NewService(&historyStub{})
	warning, err := service.Check(context.Background(), domain.DefaultAppSettings(), nil, time.Now().UTC())
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if warning != nil {
		t.Fatalf("expected no warning, got %+v", warning)
	}
}
