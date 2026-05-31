package bluesky

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"StreamSignal/internal/domain"
)

func TestPublisherCreatesSessionThenPublishesPost(t *testing.T) {
	var createSessionPayload map[string]string
	var createRecordPayload map[string]any
	var authHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/com.atproto.server.createSession":
			if err := json.NewDecoder(r.Body).Decode(&createSessionPayload); err != nil {
				t.Fatalf("decode createSession payload: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"accessJwt":"jwt-token","did":"did:plc:test"}`))
		case "/xrpc/com.atproto.repo.createRecord":
			authHeader = r.Header.Get("Authorization")
			if err := json.NewDecoder(r.Body).Decode(&createRecordPayload); err != nil {
				t.Fatalf("decode createRecord payload: %v", err)
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	publisher := NewPublisher(server.URL, server.Client())
	publisher.now = func() time.Time {
		return time.Date(2026, 5, 31, 19, 0, 0, 0, time.UTC)
	}

	if err := publisher.PublishPost(context.Background(), "don.test", "app-password", "Going Live"); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	if createSessionPayload["identifier"] != "don.test" || createSessionPayload["password"] != "app-password" {
		t.Fatalf("unexpected createSession payload: %+v", createSessionPayload)
	}
	if authHeader != "Bearer jwt-token" {
		t.Fatalf("expected bearer token, got %q", authHeader)
	}
	if createRecordPayload["collection"] != "app.bsky.feed.post" || createRecordPayload["repo"] != "did:plc:test" {
		t.Fatalf("unexpected createRecord payload: %+v", createRecordPayload)
	}
	record, ok := createRecordPayload["record"].(map[string]any)
	if !ok {
		t.Fatalf("expected record object, got %+v", createRecordPayload["record"])
	}
	if record["text"] != "Going Live" || record["createdAt"] != "2026-05-31T19:00:00Z" || record["$type"] != "app.bsky.feed.post" {
		t.Fatalf("unexpected record payload: %+v", record)
	}
}

func TestPublisherCreatesSessionThenSetsLiveNow(t *testing.T) {
	var payload map[string]any
	var authHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/com.atproto.server.createSession":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"accessJwt":"jwt-token","did":"did:plc:test"}`))
		case "/xrpc/com.atproto.repo.putRecord":
			authHeader = r.Header.Get("Authorization")
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode putRecord payload: %v", err)
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	publisher := NewPublisher(server.URL, server.Client())
	publisher.now = func() time.Time {
		return time.Date(2026, 5, 31, 19, 30, 0, 0, time.UTC)
	}

	err := publisher.SetLiveNow(context.Background(), "don.test", "app-password", domain.BlueskyLiveNowStatus{
		URL:             "https://twitch.tv/don",
		Title:           "Don is live",
		Description:     "Come hang out",
		DurationMinutes: 120,
	})
	if err != nil {
		t.Fatalf("set live now: %v", err)
	}
	if authHeader != "Bearer jwt-token" {
		t.Fatalf("expected bearer token, got %q", authHeader)
	}
	if payload["collection"] != "app.bsky.actor.status" || payload["repo"] != "did:plc:test" || payload["rkey"] != "self" {
		t.Fatalf("unexpected putRecord payload: %+v", payload)
	}
	record, ok := payload["record"].(map[string]any)
	if !ok {
		t.Fatalf("expected record object, got %+v", payload["record"])
	}
	if record["$type"] != "app.bsky.actor.status" || record["status"] != "app.bsky.actor.status#live" || record["createdAt"] != "2026-05-31T19:30:00Z" {
		t.Fatalf("unexpected record payload: %+v", record)
	}
	if record["durationMinutes"] != float64(120) {
		t.Fatalf("expected durationMinutes 120, got %+v", record["durationMinutes"])
	}
	embed, ok := record["embed"].(map[string]any)
	if !ok {
		t.Fatalf("expected embed object, got %+v", record["embed"])
	}
	if embed["$type"] != "app.bsky.embed.external" {
		t.Fatalf("unexpected embed type: %+v", embed)
	}
	external, ok := embed["external"].(map[string]any)
	if !ok {
		t.Fatalf("expected external card object, got %+v", embed["external"])
	}
	if external["uri"] != "https://twitch.tv/don" || external["title"] != "Don is live" || external["description"] != "Come hang out" {
		t.Fatalf("unexpected external card: %+v", external)
	}
}

func TestPublisherCreatesSessionThenClearsLiveNow(t *testing.T) {
	var payload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/com.atproto.server.createSession":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"accessJwt":"jwt-token","did":"did:plc:test"}`))
		case "/xrpc/com.atproto.repo.deleteRecord":
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode deleteRecord payload: %v", err)
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	publisher := NewPublisher(server.URL, server.Client())

	if err := publisher.ClearLiveNow(context.Background(), "don.test", "app-password"); err != nil {
		t.Fatalf("clear live now: %v", err)
	}
	if payload["collection"] != "app.bsky.actor.status" || payload["repo"] != "did:plc:test" || payload["rkey"] != "self" {
		t.Fatalf("unexpected deleteRecord payload: %+v", payload)
	}
}

func TestPublisherReturnsSessionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "invalid password", http.StatusUnauthorized)
	}))
	defer server.Close()

	publisher := NewPublisher(server.URL, server.Client())

	err := publisher.PublishPost(context.Background(), "don.test", "bad-password", "Going Live")
	if err == nil {
		t.Fatal("expected publish error")
	}
}

func TestPublisherReturnsCreateRecordError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/com.atproto.server.createSession":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"accessJwt":"jwt-token","did":"did:plc:test"}`))
		case "/xrpc/com.atproto.repo.createRecord":
			http.Error(w, "posting disabled", http.StatusBadRequest)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	publisher := NewPublisher(server.URL, server.Client())

	err := publisher.PublishPost(context.Background(), "don.test", "app-password", "Going Live")
	if err == nil {
		t.Fatal("expected publish error")
	}
}

func TestPublisherVerifiesCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.server.createSession" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accessJwt":"jwt-token","did":"did:plc:test"}`))
	}))
	defer server.Close()

	publisher := NewPublisher(server.URL, server.Client())

	if err := publisher.VerifyCredentials(context.Background(), "don.test", "app-password"); err != nil {
		t.Fatalf("verify credentials: %v", err)
	}
}
