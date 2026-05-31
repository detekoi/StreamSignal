package discord

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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

	if err := publisher.Publish(context.Background(), server.URL, "Going Live"); err != nil {
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

func TestPublisherReturnsErrorForNonSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad webhook", http.StatusBadRequest)
	}))
	defer server.Close()

	publisher := NewPublisher(server.Client())

	err := publisher.Publish(context.Background(), server.URL, "Going Live")
	if err == nil {
		t.Fatal("expected publish error")
	}
	if got := err.Error(); got == "" || got == "bad webhook" {
		t.Fatalf("expected contextual error, got %q", got)
	}
}

func TestPublisherReturnsErrorForInvalidURL(t *testing.T) {
	publisher := NewPublisher(nil)

	err := publisher.Publish(context.Background(), "://bad-url", "Going Live")
	if err == nil {
		t.Fatal("expected publish error")
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
