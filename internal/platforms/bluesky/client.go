package bluesky

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"StreamSignal/internal/domain"
)

const defaultBaseURL = "https://bsky.social"

type Publisher struct {
	baseURL string
	client  *http.Client
	now     func() time.Time
}

type createSessionRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type createSessionResponse struct {
	AccessJWT string `json:"accessJwt"`
	DID       string `json:"did"`
}

type createRecordRequest struct {
	Collection string         `json:"collection"`
	Repo       string         `json:"repo"`
	Record     postRecordBody `json:"record"`
}

type putRecordRequest struct {
	Collection string              `json:"collection"`
	Repo       string              `json:"repo"`
	RKey       string              `json:"rkey"`
	Record     liveNowStatusRecord `json:"record"`
}

type deleteRecordRequest struct {
	Collection string `json:"collection"`
	Repo       string `json:"repo"`
	RKey       string `json:"rkey"`
}

type postRecordBody struct {
	Type      string `json:"$type"`
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt"`
}

type liveNowStatusRecord struct {
	Type            string         `json:"$type"`
	Status          string         `json:"status"`
	Embed           *externalEmbed `json:"embed,omitempty"`
	DurationMinutes int            `json:"durationMinutes,omitempty"`
	CreatedAt       string         `json:"createdAt"`
}

type externalEmbed struct {
	Type     string            `json:"$type"`
	External externalEmbedCard `json:"external"`
}

type externalEmbedCard struct {
	URI         string `json:"uri"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func NewPublisher(baseURL string, client *http.Client) *Publisher {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Publisher{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
		now:     time.Now,
	}
}

func (p *Publisher) PublishPost(ctx context.Context, accountIdentifier string, credentialKey string, content string) error {
	session, err := p.createSession(ctx, accountIdentifier, credentialKey)
	if err != nil {
		return err
	}

	record := createRecordRequest{
		Collection: "app.bsky.feed.post",
		Repo:       session.DID,
		Record: postRecordBody{
			Type:      "app.bsky.feed.post",
			Text:      content,
			CreatedAt: p.now().UTC().Format(time.RFC3339),
		},
	}

	body, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode Bluesky post payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/xrpc/com.atproto.repo.createRecord", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build Bluesky createRecord request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+session.AccessJWT)
	request.Header.Set("Content-Type", "application/json")

	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("send Bluesky createRecord request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Bluesky createRecord request failed: %s%s", response.Status, readResponseSuffix(response.Body))
	}

	return nil
}

func (p *Publisher) VerifyCredentials(ctx context.Context, accountIdentifier string, credentialKey string) error {
	_, err := p.createSession(ctx, accountIdentifier, credentialKey)
	return err
}

func (p *Publisher) SetLiveNow(ctx context.Context, accountIdentifier string, credentialKey string, status domain.BlueskyLiveNowStatus) error {
	session, err := p.createSession(ctx, accountIdentifier, credentialKey)
	if err != nil {
		return err
	}

	record := liveNowStatusRecord{
		Type:      "app.bsky.actor.status",
		Status:    "app.bsky.actor.status#live",
		CreatedAt: p.now().UTC().Format(time.RFC3339),
	}
	record.Embed = &externalEmbed{
		Type: "app.bsky.embed.external",
		External: externalEmbedCard{
			URI:         status.URL,
			Title:       status.Title,
			Description: status.Description,
		},
	}
	if status.DurationMinutes > 0 {
		record.DurationMinutes = status.DurationMinutes
	}

	payload := putRecordRequest{
		Collection: "app.bsky.actor.status",
		Repo:       session.DID,
		RKey:       "self",
		Record:     record,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode Bluesky Live Now payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/xrpc/com.atproto.repo.putRecord", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build Bluesky putRecord request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+session.AccessJWT)
	request.Header.Set("Content-Type", "application/json")

	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("send Bluesky putRecord request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Bluesky putRecord request failed: %s%s", response.Status, readResponseSuffix(response.Body))
	}

	return nil
}

func (p *Publisher) ClearLiveNow(ctx context.Context, accountIdentifier string, credentialKey string) error {
	session, err := p.createSession(ctx, accountIdentifier, credentialKey)
	if err != nil {
		return err
	}

	payload := deleteRecordRequest{
		Collection: "app.bsky.actor.status",
		Repo:       session.DID,
		RKey:       "self",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode Bluesky Live Now delete payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/xrpc/com.atproto.repo.deleteRecord", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build Bluesky deleteRecord request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+session.AccessJWT)
	request.Header.Set("Content-Type", "application/json")

	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("send Bluesky deleteRecord request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Bluesky deleteRecord request failed: %s%s", response.Status, readResponseSuffix(response.Body))
	}

	return nil
}

func (p *Publisher) createSession(ctx context.Context, accountIdentifier string, credentialKey string) (createSessionResponse, error) {
	payload := createSessionRequest{
		Identifier: strings.TrimSpace(accountIdentifier),
		Password:   strings.TrimSpace(credentialKey),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return createSessionResponse{}, fmt.Errorf("encode Bluesky session payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/xrpc/com.atproto.server.createSession", bytes.NewReader(body))
	if err != nil {
		return createSessionResponse{}, fmt.Errorf("build Bluesky createSession request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := p.client.Do(request)
	if err != nil {
		return createSessionResponse{}, fmt.Errorf("send Bluesky createSession request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return createSessionResponse{}, fmt.Errorf("Bluesky createSession request failed: %s%s", response.Status, readResponseSuffix(response.Body))
	}

	var session createSessionResponse
	if err := json.NewDecoder(response.Body).Decode(&session); err != nil {
		return createSessionResponse{}, fmt.Errorf("decode Bluesky createSession response: %w", err)
	}
	if session.AccessJWT == "" || session.DID == "" {
		return createSessionResponse{}, fmt.Errorf("Bluesky createSession response was missing required fields")
	}

	return session, nil
}

func readResponseSuffix(body io.Reader) string {
	responseBody, _ := io.ReadAll(io.LimitReader(body, 512))
	message := strings.TrimSpace(string(responseBody))
	if message == "" {
		return ""
	}
	return ": " + message
}
