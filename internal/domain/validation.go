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
	return ValidateAnnouncementForTemplate(input, "{{stream_title}}{{stream_url}}")
}

func ValidateAnnouncementForTemplate(input Announcement, template string) []string {
	var notes []string

	if templateUses(template, "{{stream_title}}") && strings.TrimSpace(input.StreamTitle) == "" {
		notes = append(notes, "Stream title is required.")
	}

	if templateUses(template, "{{stream_url}}") && strings.TrimSpace(input.StreamURL) == "" {
		notes = append(notes, "Stream URL is required.")
	} else if templateUses(template, "{{stream_url}}") && !isAbsoluteHTTPURL(strings.TrimSpace(input.StreamURL)) {
		notes = append(notes, "Stream URL must be a valid absolute URL.")
	}

	if templateUses(template, "{{category}}") && strings.TrimSpace(input.Category) == "" {
		notes = append(notes, "Category is required.")
	}

	if templateUses(template, "{{message}}") && strings.TrimSpace(input.Message) == "" {
		notes = append(notes, "Message is required.")
	}

	if templateUses(template, "{{hashtags}}") && strings.TrimSpace(input.Hashtags) == "" {
		notes = append(notes, "Hashtags are required.")
	}

	return notes
}

func templateUses(template string, token string) bool {
	return strings.Contains(template, token)
}

func isAbsoluteHTTPURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
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
