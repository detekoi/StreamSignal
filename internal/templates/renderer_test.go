package templates

import (
	"testing"
	"time"

	"StreamSignal/internal/domain"
)

func TestRenderReplacesSupportedVariables(t *testing.T) {
	announcement := domain.Announcement{
		StreamTitle: "Late Night Variety",
		StreamURL:   "https://example.com/live",
		Category:    "Variety",
		Message:     "Come hang out",
		Hashtags:    "#stream #vtuber",
	}
	now := time.Date(2026, 5, 31, 21, 45, 0, 0, time.UTC)

	rendered := Render("{{stream_title}}|{{stream_url}}|{{category}}|{{message}}|{{hashtags}}|{{date}}|{{time}}|{{platform}}", announcement, domain.PlatformBluesky, now)
	expected := "Late Night Variety|https://example.com/live|Variety|Come hang out|#stream #vtuber|2026-05-31|21:45|bluesky"

	if rendered != expected {
		t.Fatalf("expected %q, got %q", expected, rendered)
	}
}

func TestRenderLeavesUnknownVariablesUnchanged(t *testing.T) {
	rendered := Render("{{stream_title}} {{unknown}}", domain.Announcement{StreamTitle: "Signal"}, domain.PlatformDiscord, time.Now().UTC())
	if rendered != "Signal {{unknown}}" {
		t.Fatalf("expected unknown variable to remain unchanged, got %q", rendered)
	}
}

func TestRenderUsesEmptyStringForMissingOptionalValues(t *testing.T) {
	rendered := Render("{{message}}/{{hashtags}}", domain.Announcement{}, domain.PlatformDiscord, time.Now().UTC())
	if rendered != "/" {
		t.Fatalf("expected empty optional values to render as empty strings, got %q", rendered)
	}
}
