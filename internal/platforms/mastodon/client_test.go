package mastodon

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

	if err := publisher.PublishPost(context.Background(), "mastodon-token", server.URL, "Going Live", domain.MastodonPostMetadata{}); err != nil {
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

	err := publisher.PublishPost(context.Background(), "mastodon-token", server.URL, "Going Live", domain.MastodonPostMetadata{})
	if err == nil {
		t.Fatal("expected publish error")
	}
}

func TestPublisherReturnsErrorForInvalidURL(t *testing.T) {
	publisher := NewPublisher(nil)

	err := publisher.PublishPost(context.Background(), "mastodon-token", "://bad-url", "Going Live", domain.MastodonPostMetadata{})
	if err == nil {
		t.Fatal("expected publish error")
	}
}

func TestPublisherUploadsAdditionalImageBeforeStatus(t *testing.T) {
	var uploadedImage string
	var gotPayload struct {
		Status   string   `json:"status"`
		MediaIDs []string `json:"media_ids"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/media":
			if r.Method != http.MethodPost {
				t.Fatalf("expected media POST, got %s", r.Method)
			}
			if r.Header.Get("Authorization") != "Bearer mastodon-token" {
				t.Fatalf("expected bearer token, got %q", r.Header.Get("Authorization"))
			}
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
				t.Fatalf("expected multipart upload, got %q", r.Header.Get("Content-Type"))
			}
			if err := r.ParseMultipartForm(maxAdditionalImageBytes); err != nil {
				t.Fatalf("parse multipart: %v", err)
			}
			file, _, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("read uploaded file: %v", err)
			}
			defer file.Close()
			body, err := io.ReadAll(file)
			if err != nil {
				t.Fatalf("read uploaded image: %v", err)
			}
			uploadedImage = string(body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"media-1"}`))
		case "/api/v1/statuses":
			if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
				t.Fatalf("decode status payload: %v", err)
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	publisher := NewPublisher(server.Client())
	err := publisher.PublishPost(context.Background(), "mastodon-token", server.URL, "Going Live", domain.MastodonPostMetadata{
		AdditionalImageDataURL: "data:image/png;base64,ZmFrZS1wbmc=",
	})
	if err != nil {
		t.Fatalf("publish post: %v", err)
	}
	if uploadedImage != "fake-png" {
		t.Fatalf("expected decoded image upload, got %q", uploadedImage)
	}
	if gotPayload.Status != "Going Live" || len(gotPayload.MediaIDs) != 1 || gotPayload.MediaIDs[0] != "media-1" {
		t.Fatalf("unexpected status payload: %+v", gotPayload)
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
