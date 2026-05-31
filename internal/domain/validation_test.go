package domain

import (
	"strings"
	"testing"
)

func TestNormalizeAnnouncementUsesDefaults(t *testing.T) {
	input := Announcement{
		StreamTitle: "Going Live",
	}
	settings := AppSettings{
		DefaultStreamURL: "https://example.com/live",
		DefaultHashtags:  "#default",
	}

	normalized := NormalizeAnnouncement(input, settings)

	if normalized.StreamURL != settings.DefaultStreamURL {
		t.Fatalf("expected default stream url %q, got %q", settings.DefaultStreamURL, normalized.StreamURL)
	}
	if normalized.Hashtags != settings.DefaultHashtags {
		t.Fatalf("expected default hashtags %q, got %q", settings.DefaultHashtags, normalized.Hashtags)
	}
}

func TestValidateAnnouncementAllowsDefaultedURL(t *testing.T) {
	input := Announcement{
		StreamTitle: "Going Live",
		StreamURL:   "https://example.com/live",
	}

	notes := ValidateAnnouncement(input)
	if len(notes) != 0 {
		t.Fatalf("expected no announcement validation notes, got %v", notes)
	}
}

func TestValidateAnnouncementReportsMissingRequiredFields(t *testing.T) {
	notes := ValidateAnnouncement(Announcement{})
	if len(notes) != 2 {
		t.Fatalf("expected 2 validation notes, got %d: %v", len(notes), notes)
	}
}

func TestValidateAnnouncementRejectsInvalidURL(t *testing.T) {
	notes := ValidateAnnouncement(Announcement{
		StreamTitle: "Going Live",
		StreamURL:   "not-a-url",
	})

	if len(notes) != 1 || !strings.Contains(notes[0], "valid absolute URL") {
		t.Fatalf("expected invalid url note, got %v", notes)
	}
}

func TestValidatePreviewContentChecksCharacterLimits(t *testing.T) {
	notes := ValidatePreviewContent(PlatformBluesky, strings.Repeat("x", 301))
	if len(notes) == 0 {
		t.Fatal("expected character limit validation note")
	}
}

func TestPreviewValidationState(t *testing.T) {
	if state := PreviewValidationState(nil); state != PreviewValidationValid {
		t.Fatalf("expected valid state, got %q", state)
	}
	if state := PreviewValidationState([]string{"warning"}); state != PreviewValidationInvalid {
		t.Fatalf("expected invalid state, got %q", state)
	}
}
