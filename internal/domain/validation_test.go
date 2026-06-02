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

func TestValidateAnnouncementForTemplateOnlyRequiresUsedFields(t *testing.T) {
	notes := ValidateAnnouncementForTemplate(Announcement{
		Message: "Discord smoke test",
	}, "{{message}}")

	if len(notes) != 0 {
		t.Fatalf("expected no notes for unused title and url fields, got %v", notes)
	}
}

func TestValidateAnnouncementForTemplateRequiresReferencedOptionalFields(t *testing.T) {
	notes := ValidateAnnouncementForTemplate(Announcement{}, "{{category}} {{message}} {{hashtags}}")

	expected := []string{"Category is required.", "Message is required.", "Hashtags are required."}
	if len(notes) != len(expected) {
		t.Fatalf("expected %d notes, got %d: %v", len(expected), len(notes), notes)
	}
	for i, note := range expected {
		if notes[i] != note {
			t.Fatalf("expected note %d to be %q, got %q", i, note, notes[i])
		}
	}
}

func TestValidateAnnouncementForTemplateIgnoresUnusedInvalidURL(t *testing.T) {
	notes := ValidateAnnouncementForTemplate(Announcement{
		Message:   "Discord smoke test",
		StreamURL: "not-a-url",
	}, "{{message}}")

	if len(notes) != 0 {
		t.Fatalf("expected unused invalid url to be ignored, got %v", notes)
	}
}

func TestValidateAnnouncementForTemplateRejectsReferencedInvalidURL(t *testing.T) {
	notes := ValidateAnnouncementForTemplate(Announcement{
		StreamTitle: "Going Live",
		StreamURL:   "not-a-url",
	}, "{{stream_title}} {{stream_url}}")

	if len(notes) != 1 || !strings.Contains(notes[0], "valid absolute URL") {
		t.Fatalf("expected invalid referenced url note, got %v", notes)
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
