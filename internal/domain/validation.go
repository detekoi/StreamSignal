package domain

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	PreviewValidationValid   = "VALID"
	PreviewValidationInvalid = "INVALID"
)

var platformCharacterLimits = map[DestinationPlatform]int{
	PlatformDiscord:  2000,
	PlatformBluesky:  300,
	PlatformMastodon: 500,
}

func NormalizeAnnouncement(input Announcement, settings AppSettings) Announcement {
	normalized := input
	if strings.TrimSpace(normalized.StreamURL) == "" {
		normalized.StreamURL = strings.TrimSpace(settings.DefaultStreamURL)
	}
	if strings.TrimSpace(normalized.Hashtags) == "" {
		normalized.Hashtags = strings.TrimSpace(settings.DefaultHashtags)
	}
	return normalized
}

func ValidateAnnouncement(input Announcement) []string {
	var notes []string

	if strings.TrimSpace(input.StreamTitle) == "" {
		notes = append(notes, "Stream title is required.")
	}

	if strings.TrimSpace(input.StreamURL) == "" {
		notes = append(notes, "Stream URL is required.")
	} else if _, err := url.ParseRequestURI(strings.TrimSpace(input.StreamURL)); err != nil {
		notes = append(notes, "Stream URL must be a valid absolute URL.")
	}

	return notes
}

func ValidatePreviewContent(platform DestinationPlatform, content string) []string {
	var notes []string
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		notes = append(notes, "Rendered content is empty.")
	}

	limit, ok := platformCharacterLimits[platform]
	if ok && len(content) > limit {
		notes = append(notes, fmt.Sprintf("Rendered content exceeds the %d character limit for %s.", limit, platform))
	}

	return notes
}

func PreviewValidationState(notes []string) string {
	if len(notes) == 0 {
		return PreviewValidationValid
	}
	return PreviewValidationInvalid
}
