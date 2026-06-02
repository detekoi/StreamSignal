package secure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/ports"
)

const secretRefPrefix = "secret://"

type DestinationRepository struct {
	inner   ports.DestinationRepository
	secrets ports.SecretStore
}

type SettingsRepository struct {
	inner   ports.SettingsRepository
	secrets ports.SecretStore
}

type LiveNowSessionRepository struct {
	inner   ports.LiveNowSessionRepository
	secrets ports.SecretStore
}

type discordDestinationConfig struct {
	Environment          string `json:"environment,omitempty"`
	ServerName           string `json:"serverName,omitempty"`
	ChannelName          string `json:"channelName,omitempty"`
	WebhookKey           string `json:"webhookKey,omitempty"`
	CardThumbnailURL     string `json:"cardThumbnailURL,omitempty"`
	CardThumbnailDataURL string `json:"cardThumbnailDataURL,omitempty"`
	EndStreamEnabled     bool   `json:"endStreamEnabled,omitempty"`
	EndStreamTemplate    string `json:"endStreamTemplate,omitempty"`
}

type blueskyDestinationConfig struct {
	Environment            string `json:"environment,omitempty"`
	AccountIdentifier      string `json:"accountIdentifier,omitempty"`
	CredentialKey          string `json:"credentialKey,omitempty"`
	LiveStatusTemplate     string `json:"liveStatusTemplate,omitempty"`
	LiveNowDurationMinutes int    `json:"liveNowDurationMinutes,omitempty"`
	CardThumbnailURL       string `json:"cardThumbnailURL,omitempty"`
	CardThumbnailDataURL   string `json:"cardThumbnailDataURL,omitempty"`
	AdditionalImageURL     string `json:"additionalImageURL,omitempty"`
	AdditionalImageDataURL string `json:"additionalImageDataURL,omitempty"`
	EndStreamEnabled       bool   `json:"endStreamEnabled,omitempty"`
	EndStreamTemplate      string `json:"endStreamTemplate,omitempty"`
}

type mastodonDestinationConfig struct {
	Environment            string `json:"environment,omitempty"`
	AccountIdentifier      string `json:"accountIdentifier,omitempty"`
	InstanceURL            string `json:"instanceURL,omitempty"`
	CredentialKey          string `json:"credentialKey,omitempty"`
	AdditionalImageURL     string `json:"additionalImageURL,omitempty"`
	AdditionalImageDataURL string `json:"additionalImageDataURL,omitempty"`
	EndStreamEnabled       bool   `json:"endStreamEnabled,omitempty"`
	EndStreamTemplate      string `json:"endStreamTemplate,omitempty"`
}

func NewDestinationRepository(inner ports.DestinationRepository, secrets ports.SecretStore) *DestinationRepository {
	return &DestinationRepository{inner: inner, secrets: secrets}
}

func NewSettingsRepository(inner ports.SettingsRepository, secrets ports.SecretStore) *SettingsRepository {
	return &SettingsRepository{inner: inner, secrets: secrets}
}

func NewLiveNowSessionRepository(inner ports.LiveNowSessionRepository, secrets ports.SecretStore) *LiveNowSessionRepository {
	return &LiveNowSessionRepository{inner: inner, secrets: secrets}
}

func (r *DestinationRepository) List(ctx context.Context) ([]domain.Destination, error) {
	items, err := r.inner.List(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = make([]domain.Destination, 0)
	}

	for index := range items {
		if err := r.hydrateDestination(ctx, &items[index]); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (r *DestinationRepository) Get(ctx context.Context, id string) (domain.Destination, error) {
	item, err := r.inner.Get(ctx, id)
	if err != nil {
		return domain.Destination{}, err
	}
	if err := r.hydrateDestination(ctx, &item); err != nil {
		return domain.Destination{}, err
	}
	return item, nil
}

func (r *DestinationRepository) Save(ctx context.Context, destination domain.Destination) error {
	item, err := r.dehydrateDestination(ctx, destination)
	if err != nil {
		return err
	}
	return r.inner.Save(ctx, item)
}

func (r *DestinationRepository) Delete(ctx context.Context, id string) error {
	for _, field := range []string{"discord-webhook", "bluesky-credential", "mastodon-credential"} {
		_ = r.secrets.Delete(ctx, destinationSecretKey(id, field))
	}
	return r.inner.Delete(ctx, id)
}

func (r *SettingsRepository) Load(ctx context.Context) (domain.AppSettings, error) {
	return r.inner.Load(ctx)
}

func (r *SettingsRepository) Save(ctx context.Context, settings domain.AppSettings) error {
	return r.inner.Save(ctx, settings)
}

func (r *LiveNowSessionRepository) Upsert(ctx context.Context, session domain.ActiveLiveNowSession) error {
	item := session

	key := liveNowSecretKey(session.DestinationID)
	if strings.TrimSpace(item.CredentialKey) == "" {
		_ = r.secrets.Delete(ctx, key)
		item.CredentialKey = ""
	} else if !isSecretRef(item.CredentialKey) {
		if err := r.secrets.Put(ctx, key, item.CredentialKey); err != nil {
			return fmt.Errorf("store Live Now session secret: %w", err)
		}
		item.CredentialKey = toSecretRef(key)
	}

	return r.inner.Upsert(ctx, item)
}

func (r *LiveNowSessionRepository) List(ctx context.Context) ([]domain.ActiveLiveNowSession, error) {
	items, err := r.inner.List(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = make([]domain.ActiveLiveNowSession, 0)
	}

	for index := range items {
		if strings.TrimSpace(items[index].CredentialKey) != "" && !isSecretRef(items[index].CredentialKey) {
			if err := r.Upsert(ctx, items[index]); err != nil {
				return nil, err
			}
		}
		if isSecretRef(items[index].CredentialKey) {
			value, err := r.secrets.Get(ctx, fromSecretRef(items[index].CredentialKey))
			if err != nil {
				items[index].CredentialKey = ""
				continue
			}
			items[index].CredentialKey = value
		}
	}
	return items, nil
}

func (r *LiveNowSessionRepository) Delete(ctx context.Context, destinationID string) error {
	_ = r.secrets.Delete(ctx, liveNowSecretKey(destinationID))
	return r.inner.Delete(ctx, destinationID)
}

func (r *DestinationRepository) hydrateDestination(ctx context.Context, destination *domain.Destination) error {
	switch destination.Platform {
	case domain.PlatformDiscord:
		var config discordDestinationConfig
		if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
			return nil
		}
		if strings.TrimSpace(config.WebhookKey) != "" && !isSecretRef(config.WebhookKey) {
			migrated, err := r.dehydrateDestination(ctx, *destination)
			if err != nil {
				return err
			}
			if err := r.inner.Save(ctx, migrated); err != nil {
				return err
			}
		}
		value, err := r.resolveSecretValue(ctx, config.WebhookKey)
		if err != nil {
			value = strings.TrimSpace(config.WebhookKey)
		}
		config.WebhookKey = value
		return rewriteDestinationConfig(destination, config)
	case domain.PlatformBluesky:
		var config blueskyDestinationConfig
		if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
			return nil
		}
		if strings.TrimSpace(config.CredentialKey) != "" && !isSecretRef(config.CredentialKey) {
			migrated, err := r.dehydrateDestination(ctx, *destination)
			if err != nil {
				return err
			}
			if err := r.inner.Save(ctx, migrated); err != nil {
				return err
			}
		}
		value, err := r.resolveSecretValue(ctx, config.CredentialKey)
		if err != nil {
			value = strings.TrimSpace(config.CredentialKey)
		}
		config.CredentialKey = value
		return rewriteDestinationConfig(destination, config)
	case domain.PlatformMastodon:
		var config mastodonDestinationConfig
		if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
			return nil
		}
		if strings.TrimSpace(config.CredentialKey) != "" && !isSecretRef(config.CredentialKey) {
			migrated, err := r.dehydrateDestination(ctx, *destination)
			if err != nil {
				return err
			}
			if err := r.inner.Save(ctx, migrated); err != nil {
				return err
			}
		}
		value, err := r.resolveSecretValue(ctx, config.CredentialKey)
		if err != nil {
			value = strings.TrimSpace(config.CredentialKey)
		}
		config.CredentialKey = value
		return rewriteDestinationConfig(destination, config)
	default:
		return nil
	}
}

func (r *DestinationRepository) dehydrateDestination(ctx context.Context, destination domain.Destination) (domain.Destination, error) {
	item := destination

	switch destination.Platform {
	case domain.PlatformDiscord:
		var config discordDestinationConfig
		if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
			return destination, nil
		}
		value, err := r.persistSecretValue(ctx, destinationSecretKey(destination.ID, "discord-webhook"), config.WebhookKey)
		if err != nil {
			return domain.Destination{}, err
		}
		config.WebhookKey = value
		if err := rewriteDestinationConfig(&item, config); err != nil {
			return domain.Destination{}, err
		}
	case domain.PlatformBluesky:
		var config blueskyDestinationConfig
		if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
			return destination, nil
		}
		value, err := r.persistSecretValue(ctx, destinationSecretKey(destination.ID, "bluesky-credential"), config.CredentialKey)
		if err != nil {
			return domain.Destination{}, err
		}
		config.CredentialKey = value
		if err := rewriteDestinationConfig(&item, config); err != nil {
			return domain.Destination{}, err
		}
	case domain.PlatformMastodon:
		var config mastodonDestinationConfig
		if err := json.Unmarshal([]byte(destination.ConfigJSON), &config); err != nil {
			return destination, nil
		}
		value, err := r.persistSecretValue(ctx, destinationSecretKey(destination.ID, "mastodon-credential"), config.CredentialKey)
		if err != nil {
			return domain.Destination{}, err
		}
		config.CredentialKey = value
		if err := rewriteDestinationConfig(&item, config); err != nil {
			return domain.Destination{}, err
		}
	}

	return item, nil
}

func (r *DestinationRepository) resolveSecretValue(ctx context.Context, value string) (string, error) {
	if !isSecretRef(value) {
		return value, nil
	}
	return r.secrets.Get(ctx, fromSecretRef(value))
}

func (r *DestinationRepository) persistSecretValue(ctx context.Context, key string, value string) (string, error) {
	return persistSecretValue(ctx, r.secrets, key, value)
}

func persistSecretValue(ctx context.Context, store ports.SecretStore, key, value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		_ = store.Delete(ctx, key)
		return "", nil
	}
	if isSecretRef(value) {
		return value, nil
	}
	if err := store.Put(ctx, key, value); err != nil {
		return "", fmt.Errorf("store secret %q: %w", key, err)
	}
	return toSecretRef(key), nil
}

func rewriteDestinationConfig(destination *domain.Destination, config any) error {
	body, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("encode destination config: %w", err)
	}
	destination.ConfigJSON = string(body)
	return nil
}

func toSecretRef(key string) string {
	return secretRefPrefix + key
}

func fromSecretRef(value string) string {
	return strings.TrimPrefix(value, secretRefPrefix)
}

func isSecretRef(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), secretRefPrefix)
}

func destinationSecretKey(destinationID, field string) string {
	return fmt.Sprintf("streamsignal/destinations/%s/%s", destinationID, field)
}

func liveNowSecretKey(destinationID string) string {
	return fmt.Sprintf("streamsignal/live-now/%s/credential", destinationID)
}
