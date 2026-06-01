package bluesky

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"StreamSignal/internal/domain"
)

func TestPublisherCreatesSessionThenPublishesPost(t *testing.T) {
	var createSessionPayload map[string]string
	var createRecordPayload map[string]any
	var authHeader string
	var uploadAuthHeader string
	var uploadContentType string
	var uploadedThumb string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/com.atproto.server.createSession":
			if err := json.NewDecoder(r.Body).Decode(&createSessionPayload); err != nil {
				t.Fatalf("decode createSession payload: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"accessJwt":"jwt-token","did":"did:plc:test"}`))
		case "/thumbnail.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("fake-png"))
		case "/xrpc/com.atproto.repo.uploadBlob":
			uploadAuthHeader = r.Header.Get("Authorization")
			uploadContentType = r.Header.Get("Content-Type")
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read uploaded thumb: %v", err)
			}
			uploadedThumb = string(body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"blob":{"$type":"blob","ref":{"$link":"thumb-cid"},"mimeType":"image/png","size":8}}`))
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

	content := "Going Live\nhttps://example.com/live"
	if err := publisher.PublishPost(context.Background(), "don.test", "app-password", content, domain.BlueskyPostMetadata{
		StreamURL:    "https://example.com/live",
		StreamTitle:  "Going Live",
		Description:  "Come hang out",
		ThumbnailURL: server.URL + "/thumbnail.png",
	}); err != nil {
		t.Fatalf("publish post: %v", err)
	}

	if createSessionPayload["identifier"] != "don.test" || createSessionPayload["password"] != "app-password" {
		t.Fatalf("unexpected createSession payload: %+v", createSessionPayload)
	}
	if authHeader != "Bearer jwt-token" {
		t.Fatalf("expected bearer token, got %q", authHeader)
	}
	if uploadAuthHeader != "Bearer jwt-token" || uploadContentType != "image/png" || uploadedThumb != "fake-png" {
		t.Fatalf("unexpected thumbnail upload: auth=%q content-type=%q body=%q", uploadAuthHeader, uploadContentType, uploadedThumb)
	}
	if createRecordPayload["collection"] != "app.bsky.feed.post" || createRecordPayload["repo"] != "did:plc:test" {
		t.Fatalf("unexpected createRecord payload: %+v", createRecordPayload)
	}
	record, ok := createRecordPayload["record"].(map[string]any)
	if !ok {
		t.Fatalf("expected record object, got %+v", createRecordPayload["record"])
	}
	if record["text"] != content || record["createdAt"] != "2026-05-31T19:00:00Z" || record["$type"] != "app.bsky.feed.post" {
		t.Fatalf("unexpected record payload: %+v", record)
	}
	facets, ok := record["facets"].([]any)
	if !ok || len(facets) != 1 {
		t.Fatalf("expected one link facet, got %+v", record["facets"])
	}
	facet := facets[0].(map[string]any)
	index := facet["index"].(map[string]any)
	if index["byteStart"] != float64(11) || index["byteEnd"] != float64(35) {
		t.Fatalf("unexpected link facet index: %+v", index)
	}
	features := facet["features"].([]any)
	feature := features[0].(map[string]any)
	if feature["$type"] != "app.bsky.richtext.facet#link" || feature["uri"] != "https://example.com/live" {
		t.Fatalf("unexpected link facet feature: %+v", feature)
	}
	embed := record["embed"].(map[string]any)
	external := embed["external"].(map[string]any)
	if embed["$type"] != "app.bsky.embed.external" || external["uri"] != "https://example.com/live" || external["title"] != "Going Live" || external["description"] != "Come hang out" {
		t.Fatalf("unexpected external embed: %+v", embed)
	}
	thumb := external["thumb"].(map[string]any)
	ref := thumb["ref"].(map[string]any)
	if ref["$link"] != "thumb-cid" || thumb["mimeType"] != "image/png" || thumb["size"] != float64(8) {
		t.Fatalf("unexpected external thumb: %+v", thumb)
	}
}

func TestPublisherUploadsDataURLThumbnailForPostCard(t *testing.T) {
	var createRecordPayload map[string]any
	var uploadedThumb string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/com.atproto.server.createSession":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"accessJwt":"jwt-token","did":"did:plc:test"}`))
		case "/xrpc/com.atproto.repo.uploadBlob":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read uploaded thumb: %v", err)
			}
			uploadedThumb = string(body)
			if r.Header.Get("Content-Type") != "image/png" {
				t.Fatalf("expected image/png upload, got %q", r.Header.Get("Content-Type"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"blob":{"$type":"blob","ref":{"$link":"uploaded-thumb-cid"},"mimeType":"image/png","size":8}}`))
		case "/xrpc/com.atproto.repo.createRecord":
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
	content := "Going Live\nhttps://example.com/live"
	err := publisher.PublishPost(context.Background(), "don.test", "app-password", content, domain.BlueskyPostMetadata{
		StreamURL:        "https://example.com/live",
		StreamTitle:      "Going Live",
		ThumbnailDataURL: "data:image/png;base64,ZmFrZS1wbmc=",
	})
	if err != nil {
		t.Fatalf("publish post: %v", err)
	}
	if uploadedThumb != "fake-png" {
		t.Fatalf("expected decoded thumbnail upload, got %q", uploadedThumb)
	}
	record := createRecordPayload["record"].(map[string]any)
	embed := record["embed"].(map[string]any)
	external := embed["external"].(map[string]any)
	thumb := external["thumb"].(map[string]any)
	ref := thumb["ref"].(map[string]any)
	if ref["$link"] != "uploaded-thumb-cid" {
		t.Fatalf("unexpected data URL thumb ref: %+v", thumb)
	}
}

func TestPublisherRejectsOversizedDataURLThumbnailBeforeDecode(t *testing.T) {
	encoded := strings.Repeat("A", base64EncodedLimit()+8)
	_, _, err := decodeImageDataURL("data:image/png;base64," + encoded)
	if err == nil || err.Error() != "Bluesky card thumbnail must be 1 MB or smaller" {
		t.Fatalf("expected oversized thumbnail error, got %v", err)
	}
}

func TestPublisherRejectsNonHTTPThumbnailURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/com.atproto.server.createSession":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"accessJwt":"jwt-token","did":"did:plc:test"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	publisher := NewPublisher(server.URL, server.Client())
	err := publisher.PublishPost(context.Background(), "don.test", "app-password", "Going Live\nhttps://example.com/live", domain.BlueskyPostMetadata{
		StreamURL:    "https://example.com/live",
		StreamTitle:  "Going Live",
		ThumbnailURL: "file:///tmp/thumb.png",
	})
	if err == nil || err.Error() != "Bluesky card thumbnail URL must be a valid HTTP or HTTPS URL" {
		t.Fatalf("expected invalid thumbnail URL error, got %v", err)
	}
}

func base64EncodedLimit() int {
	return ((maxExternalThumbBytes + 2) / 3) * 4
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

	err := publisher.PublishPost(context.Background(), "don.test", "bad-password", "Going Live", domain.BlueskyPostMetadata{})
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

	err := publisher.PublishPost(context.Background(), "don.test", "app-password", "Going Live", domain.BlueskyPostMetadata{})
	if err == nil {
		t.Fatal("expected publish error")
	}
}

func TestPublisherVerifiesCredentials(t *testing.T) {
	var createSessionPayload map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.server.createSession" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&createSessionPayload); err != nil {
			t.Fatalf("decode createSession payload: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accessJwt":"jwt-token","did":"did:plc:test"}`))
	}))
	defer server.Close()

	publisher := NewPublisher(server.URL, server.Client())

	if err := publisher.VerifyCredentials(context.Background(), " @don.test ", " app\u2011pass word "); err != nil {
		t.Fatalf("verify credentials: %v", err)
	}
	if createSessionPayload["identifier"] != "don.test" || createSessionPayload["password"] != "app-password" {
		t.Fatalf("unexpected createSession payload: %+v", createSessionPayload)
	}
}
