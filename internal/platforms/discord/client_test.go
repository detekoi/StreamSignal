package discord

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"StreamSignal/internal/domain"
)

func TestPublisherSendsWebhookPayload(t *testing.T) {
	var gotMethod string
	var gotContentType string
	var gotPayload map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	publisher := NewPublisher(server.Client())

	if err := publisher.Publish(context.Background(), server.URL, "Going Live", domain.DiscordPostMetadata{}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %s", gotMethod)
	}
	if gotContentType != "application/json" {
		t.Fatalf("expected application/json, got %s", gotContentType)
	}
	if gotPayload["content"] != "Going Live" {
		t.Fatalf("expected content payload, got %+v", gotPayload)
	}
}

func TestPublisherSendsWebhookPayloadWithImageEmbed(t *testing.T) {
	var gotPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	publisher := NewPublisher(server.Client())

	err := publisher.Publish(context.Background(), server.URL, "Going Live", domain.DiscordPostMetadata{
		ThumbnailURL: "https://example.com/card.png",
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	embeds := gotPayload["embeds"].([]any)
	embed := embeds[0].(map[string]any)
	if _, ok := embed["url"]; ok {
		t.Fatalf("expected no StreamSignal-added embed URL, got %+v", gotPayload)
	}
	if _, ok := embed["title"]; ok {
		t.Fatalf("expected no StreamSignal-added embed title, got %+v", gotPayload)
	}
	image := embed["image"].(map[string]any)
	if image["url"] != "https://example.com/card.png" {
		t.Fatalf("expected embed image URL, got %+v", gotPayload)
	}
}

func TestPublisherSendsWebhookPayloadWithUploadedImageEmbed(t *testing.T) {
	var gotContentType string
	var gotBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		gotBody = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	publisher := NewPublisher(server.Client())

	err := publisher.Publish(context.Background(), server.URL, "Going Live", domain.DiscordPostMetadata{
		ThumbnailDataURL: "data:image/png;base64,ZmFrZS1wbmc=",
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	if !strings.HasPrefix(gotContentType, "multipart/form-data;") {
		t.Fatalf("expected multipart form data, got %q", gotContentType)
	}
	for _, expected := range []string{
		`"content":"Going Live"`,
		`"url":"attachment://streamsignal-card-thumbnail.png"`,
		`name="files[0]"; filename="streamsignal-card-thumbnail.png"`,
		"fake-png",
	} {
		if !strings.Contains(gotBody, expected) {
			t.Fatalf("expected multipart body to contain %q, got %s", expected, gotBody)
		}
	}
}

func TestPublisherReturnsErrorForNonSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad webhook", http.StatusBadRequest)
	}))
	defer server.Close()

	publisher := NewPublisher(server.Client())

	err := publisher.Publish(context.Background(), server.URL, "Going Live", domain.DiscordPostMetadata{})
	if err == nil {
		t.Fatal("expected publish error")
	}
	if got := err.Error(); got == "" || got == "bad webhook" {
		t.Fatalf("expected contextual error, got %q", got)
	}
}

func TestPublisherReturnsErrorForInvalidURL(t *testing.T) {
	publisher := NewPublisher(nil)

	err := publisher.Publish(context.Background(), "://bad-url", "Going Live", domain.DiscordPostMetadata{})
	if err == nil {
		t.Fatal("expected publish error")
	}
}

func TestPublisherRejectsInvalidImageURL(t *testing.T) {
	publisher := NewPublisher(nil)

	err := publisher.Publish(context.Background(), "https://example.com/webhook", "Going Live", domain.DiscordPostMetadata{
		ThumbnailURL: "file:///tmp/card.png",
	})
	if err == nil || err.Error() != "Discord card thumbnail URL must be a valid HTTP or HTTPS URL" {
		t.Fatalf("expected invalid image URL error, got %v", err)
	}
}

func TestPublisherVerifiesWebhook(t *testing.T) {
	var gotMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	publisher := NewPublisher(server.Client())

	if err := publisher.VerifyWebhook(context.Background(), server.URL); err != nil {
		t.Fatalf("verify webhook: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("expected GET, got %s", gotMethod)
	}
}

func TestPublisherVerifyWebhookReturnsErrorForNonSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "missing webhook", http.StatusNotFound)
	}))
	defer server.Close()

	publisher := NewPublisher(server.Client())

	err := publisher.VerifyWebhook(context.Background(), server.URL)
	if err == nil {
		t.Fatal("expected verify error")
	}
}
