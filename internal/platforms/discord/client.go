package discord

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

type messagePayload struct {
	Content string `json:"content"`
}

func NewPublisher(client *http.Client) *Publisher {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Publisher{client: client}
}

func (p *Publisher) Publish(ctx context.Context, webhookURL string, content string) error {
	payload := messagePayload{Content: content}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode Discord payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(webhookURL), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build Discord webhook request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("send Discord webhook request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		message := strings.TrimSpace(string(responseBody))
		if message != "" {
			return fmt.Errorf("Discord webhook request failed: %s: %s", response.Status, message)
		}
		return fmt.Errorf("Discord webhook request failed: %s", response.Status)
	}

	return nil
}

func (p *Publisher) VerifyWebhook(ctx context.Context, webhookURL string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimSpace(webhookURL), nil)
	if err != nil {
		return fmt.Errorf("build Discord webhook request: %w", err)
	}

	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("send Discord webhook request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		message := strings.TrimSpace(string(responseBody))
		if message != "" {
			return fmt.Errorf("Discord webhook verification failed: %s: %s", response.Status, message)
		}
		return fmt.Errorf("Discord webhook verification failed: %s", response.Status)
	}

	return nil
}
