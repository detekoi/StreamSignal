package templates

import (
	"strings"
	"time"

	"StreamSignal/internal/domain"
)

func Render(template string, announcement domain.Announcement, platform domain.DestinationPlatform, now time.Time) string {
	replacements := map[string]string{
		"{{stream_title}}": announcement.StreamTitle,
		"{{stream_url}}":   announcement.StreamURL,
		"{{category}}":     announcement.Category,
		"{{message}}":      announcement.Message,
		"{{hashtags}}":     announcement.Hashtags,
		"{{date}}":         now.Format("2006-01-02"),
		"{{time}}":         now.Format("15:04"),
		"{{platform}}":     string(platform),
	}

	rendered := template
	for key, value := range replacements {
		rendered = strings.ReplaceAll(rendered, key, value)
	}

	return rendered
}
