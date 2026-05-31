package mastodon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Publisher struct {
	client *http.Client
}

type statusPayload struct {
	Status string `json:"status"`
}

func NewPublisher(client *http.Client) *Publisher {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Publisher{client: client}
}

func (p *Publisher) PublishPost(ctx context.Context, credentialKey string, instanceURL string, content string) error {
	payload := statusPayload{Status: content}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode Mastodon payload: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(strings.TrimSpace(instanceURL), "/")+"/api/v1/statuses",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build Mastodon status request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(credentialKey))
	request.Header.Set("Content-Type", "application/json")

	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("send Mastodon status request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		message := strings.TrimSpace(string(responseBody))
		if message != "" {
			return fmt.Errorf("Mastodon status request failed: %s: %s", response.Status, message)
		}
		return fmt.Errorf("Mastodon status request failed: %s", response.Status)
	}

	return nil
}

func (p *Publisher) VerifyCredentials(ctx context.Context, credentialKey string, instanceURL string) error {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		strings.TrimRight(strings.TrimSpace(instanceURL), "/")+"/api/v1/accounts/verify_credentials",
		nil,
	)
	if err != nil {
		return fmt.Errorf("build Mastodon verify request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(credentialKey))

	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("send Mastodon verify request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		message := strings.TrimSpace(string(responseBody))
		if message != "" {
			return fmt.Errorf("Mastodon verify request failed: %s: %s", response.Status, message)
		}
		return fmt.Errorf("Mastodon verify request failed: %s", response.Status)
	}

	return nil
}
