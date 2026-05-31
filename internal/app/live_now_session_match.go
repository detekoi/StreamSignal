package app

import (
	"encoding/json"
	"strings"

	"StreamSignal/internal/domain"
)

func trackedSessionMatchesDestination(destination domain.Destination, session domain.ActiveLiveNowSession) bool {
	if destination.Platform != domain.PlatformBluesky {
		return false
	}
	if strings.TrimSpace(session.AccountIdentifier) == "" || strings.TrimSpace(session.CredentialKey) == "" {
		return false
	}

	var config blueskyDestinationConfig
	if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
		return false
	}

	return strings.TrimSpace(config.AccountIdentifier) != "" &&
		strings.TrimSpace(config.CredentialKey) != "" &&
		!strings.HasPrefix(strings.TrimSpace(config.CredentialKey), "secret://") &&
		strings.EqualFold(strings.TrimSpace(config.AccountIdentifier), strings.TrimSpace(session.AccountIdentifier)) &&
		strings.TrimSpace(config.CredentialKey) == strings.TrimSpace(session.CredentialKey)
}
