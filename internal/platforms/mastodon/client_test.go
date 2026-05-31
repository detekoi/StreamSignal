package mastodon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublisherSendsStatusPayload(t *testing.T) {
	var gotMethod string
	var gotAuth string
	var gotPayload map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/statuses" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	publisher := NewPublisher(server.Client())

	if err := publisher.PublishPost(context.Background(), "mastodon-token", server.URL, "Going Live"); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %s", gotMethod)
	}
	if gotAuth != "Bearer mastodon-token" {
		t.Fatalf("expected bearer token, got %q", gotAuth)
	}
	if gotPayload["status"] != "Going Live" {
		t.Fatalf("expected status payload, got %+v", gotPayload)
	}
}

func TestPublisherReturnsErrorForNonSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "posting blocked", http.StatusUnauthorized)
	}))
	defer server.Close()

	publisher := NewPublisher(server.Client())

	err := publisher.PublishPost(context.Background(), "mastodon-token", server.URL, "Going Live")
	if err == nil {
		t.Fatal("expected publish error")
	}
}

func TestPublisherReturnsErrorForInvalidURL(t *testing.T) {
	publisher := NewPublisher(nil)

	err := publisher.PublishPost(context.Background(), "mastodon-token", "://bad-url", "Going Live")
	if err == nil {
		t.Fatal("expected publish error")
	}
}

func TestPublisherVerifiesCredentials(t *testing.T) {
	var gotMethod string
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/accounts/verify_credentials" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	publisher := NewPublisher(server.Client())

	if err := publisher.VerifyCredentials(context.Background(), "mastodon-token", server.URL); err != nil {
		t.Fatalf("verify credentials: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Fatalf("expected GET, got %s", gotMethod)
	}
	if gotAuth != "Bearer mastodon-token" {
		t.Fatalf("expected bearer token, got %q", gotAuth)
	}
}
