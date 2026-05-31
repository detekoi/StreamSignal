package domain

import "time"

type Announcement struct {
	StreamTitle    string    `json:"streamTitle"`
	StreamURL      string    `json:"streamURL"`
	Category       string    `json:"category"`
	Message        string    `json:"message"`
	Hashtags       string    `json:"hashtags"`
	DestinationIDs []string  `json:"destinationIDs"`
	CreatedAt      time.Time `json:"createdAt"`
}

type DestinationPlatform string

const (
	PlatformDiscord  DestinationPlatform = "discord"
	PlatformBluesky  DestinationPlatform = "bluesky"
	PlatformMastodon DestinationPlatform = "mastodon"
)

type Destination struct {
	ID         string              `json:"id"`
	Platform   DestinationPlatform `json:"platform"`
	Name       string              `json:"name"`
	Enabled    bool                `json:"enabled"`
	Template   string              `json:"template"`
	ConfigJSON string              `json:"configJSON"`
	CreatedAt  time.Time           `json:"createdAt"`
	UpdatedAt  time.Time           `json:"updatedAt"`
}

type AppSettings struct {
	TestModeEnabled              bool   `json:"testModeEnabled"`
	TestDiscordWebhookKey        string `json:"testDiscordWebhookKey"`
	TestBlueskyAccountIdentifier string `json:"testBlueskyAccountIdentifier"`
	TestBlueskyCredentialKey     string `json:"testBlueskyCredentialKey"`
	TestMastodonCredentialKey    string `json:"testMastodonCredentialKey"`
	TestMastodonInstanceURL      string `json:"testMastodonInstanceURL"`
	DefaultStreamURL             string `json:"defaultStreamURL"`
	DefaultHashtags              string `json:"defaultHashtags"`
	DuplicateProtectionEnabled   bool   `json:"duplicateProtectionEnabled"`
	DuplicateWindowMinutes       int    `json:"duplicateWindowMinutes"`
	EndStreamPostEnabled         bool   `json:"endStreamPostEnabled"`
	EndStreamTemplate            string `json:"endStreamTemplate"`
}

func DefaultAppSettings() AppSettings {
	return AppSettings{
		TestModeEnabled:              false,
		TestDiscordWebhookKey:        "",
		TestBlueskyAccountIdentifier: "",
		TestBlueskyCredentialKey:     "",
		TestMastodonCredentialKey:    "",
		TestMastodonInstanceURL:      "",
		DefaultStreamURL:             "",
		DefaultHashtags:              "",
		DuplicateProtectionEnabled:   true,
		DuplicateWindowMinutes:       10,
		EndStreamPostEnabled:         false,
		EndStreamTemplate:            "",
	}
}

type LogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Destination string    `json:"destination"`
	Action      string    `json:"action"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
}

type CredentialCheckResult struct {
	Platform string `json:"platform"`
	State    string `json:"state"`
	Message  string `json:"message"`
}

type PreviewItem struct {
	DestinationID   string              `json:"destinationID"`
	DestinationName string              `json:"destinationName"`
	Platform        DestinationPlatform `json:"platform"`
	Content         string              `json:"content"`
	CharacterCount  int                 `json:"characterCount"`
	ValidationState string              `json:"validationState"`
	ValidationNotes []string            `json:"validationNotes"`
}

type PostHistoryRecord struct {
	ID            int64     `json:"id"`
	DestinationID string    `json:"destinationID"`
	ContentHash   string    `json:"contentHash"`
	RenderedText  string    `json:"renderedText"`
	PostedAt      time.Time `json:"postedAt"`
}

type BlueskyLiveNowStatus struct {
	URL             string `json:"url"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	DurationMinutes int    `json:"durationMinutes"`
}

type ActiveLiveNowSession struct {
	DestinationID     string    `json:"destinationID"`
	DestinationName   string    `json:"destinationName"`
	Platform          string    `json:"platform"`
	AccountIdentifier string    `json:"accountIdentifier"`
	CredentialKey     string    `json:"credentialKey"`
	StreamURL         string    `json:"streamURL"`
	StreamTitle       string    `json:"streamTitle"`
	StartedAt         time.Time `json:"startedAt"`
}
