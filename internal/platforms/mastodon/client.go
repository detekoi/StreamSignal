package mastodon

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/platforms/netguard"
)

const maxAdditionalImageBytes = 8_000_000

type Publisher struct {
	client *http.Client
}

type statusPayload struct {
	Status   string   `json:"status"`
	MediaIDs []string `json:"media_ids,omitempty"`
}

type mediaUploadResponse struct {
	ID string `json:"id"`
}

func NewPublisher(client *http.Client) *Publisher {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Publisher{client: client}
}

func (p *Publisher) PublishPost(ctx context.Context, credentialKey string, instanceURL string, content string, metadata domain.MastodonPostMetadata) error {
	normalizedInstanceURL := strings.TrimRight(strings.TrimSpace(instanceURL), "/")
	trimmedCredentialKey := strings.TrimSpace(credentialKey)
	mediaIDs, err := p.uploadAdditionalImage(ctx, trimmedCredentialKey, normalizedInstanceURL, metadata)
	if err != nil {
		return err
	}

	payload := statusPayload{Status: content, MediaIDs: mediaIDs}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode Mastodon payload: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		normalizedInstanceURL+"/api/v1/statuses",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build Mastodon status request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+trimmedCredentialKey)
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

func (p *Publisher) uploadAdditionalImage(ctx context.Context, credentialKey string, instanceURL string, metadata domain.MastodonPostMetadata) ([]string, error) {
	if additionalImageDataURL := strings.TrimSpace(metadata.AdditionalImageDataURL); additionalImageDataURL != "" {
		contentType, data, err := decodeImageDataURL(additionalImageDataURL)
		if err != nil {
			return nil, err
		}
		mediaID, err := p.uploadMedia(ctx, credentialKey, instanceURL, contentType, data)
		if err != nil {
			return nil, err
		}
		return []string{mediaID}, nil
	}

	if additionalImageURL := strings.TrimSpace(metadata.AdditionalImageURL); additionalImageURL != "" {
		contentType, data, err := p.fetchAdditionalImage(ctx, additionalImageURL)
		if err != nil {
			return nil, err
		}
		mediaID, err := p.uploadMedia(ctx, credentialKey, instanceURL, contentType, data)
		if err != nil {
			return nil, err
		}
		return []string{mediaID}, nil
	}

	return nil, nil
}

func decodeImageDataURL(dataURL string) (string, []byte, error) {
	const marker = ";base64,"
	if !strings.HasPrefix(dataURL, "data:") {
		return "", nil, fmt.Errorf("Mastodon additional image upload must be a data URL")
	}
	index := strings.Index(dataURL, marker)
	if index < 0 {
		return "", nil, fmt.Errorf("Mastodon additional image upload must be base64 encoded")
	}
	contentType := strings.TrimSpace(strings.TrimPrefix(dataURL[:index], "data:"))
	if !strings.HasPrefix(contentType, "image/") {
		return "", nil, fmt.Errorf("Mastodon additional image must be an image")
	}
	encoded := dataURL[index+len(marker):]
	if len(encoded) > base64.StdEncoding.EncodedLen(maxAdditionalImageBytes)+4 {
		return "", nil, fmt.Errorf("Mastodon additional image must be 8 MB or smaller")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, fmt.Errorf("decode Mastodon additional image upload: %w", err)
	}
	if len(data) > maxAdditionalImageBytes {
		return "", nil, fmt.Errorf("Mastodon additional image must be 8 MB or smaller")
	}
	return contentType, data, nil
}

func (p *Publisher) fetchAdditionalImage(ctx context.Context, imageURL string) (string, []byte, error) {
	if err := netguard.ValidatePublicHTTPURL(ctx, imageURL, "Mastodon additional image"); err != nil {
		return "", nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return "", nil, fmt.Errorf("build Mastodon additional image request: %w", err)
	}
	response, err := p.client.Do(request)
	if err != nil {
		return "", nil, fmt.Errorf("fetch Mastodon additional image: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		message := strings.TrimSpace(string(responseBody))
		if message != "" {
			return "", nil, fmt.Errorf("fetch Mastodon additional image failed: %s: %s", response.Status, message)
		}
		return "", nil, fmt.Errorf("fetch Mastodon additional image failed: %s", response.Status)
	}

	contentType := response.Header.Get("Content-Type")
	if index := strings.Index(contentType, ";"); index >= 0 {
		contentType = contentType[:index]
	}
	contentType = strings.TrimSpace(contentType)
	if !strings.HasPrefix(contentType, "image/") {
		return "", nil, fmt.Errorf("Mastodon additional image must be an image")
	}

	data, err := io.ReadAll(io.LimitReader(response.Body, maxAdditionalImageBytes+1))
	if err != nil {
		return "", nil, fmt.Errorf("read Mastodon additional image: %w", err)
	}
	if len(data) > maxAdditionalImageBytes {
		return "", nil, fmt.Errorf("Mastodon additional image must be 8 MB or smaller")
	}
	return contentType, data, nil
}

func (p *Publisher) uploadMedia(ctx context.Context, credentialKey string, instanceURL string, contentType string, data []byte) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="streamsignal-additional-image`+extensionForContentType(contentType)+`"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return "", fmt.Errorf("build Mastodon media upload form: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return "", fmt.Errorf("write Mastodon media upload form: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("finalize Mastodon media upload form: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, instanceURL+"/api/v1/media", &body)
	if err != nil {
		return "", fmt.Errorf("build Mastodon media upload request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+credentialKey)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	response, err := p.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("send Mastodon media upload request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		message := strings.TrimSpace(string(responseBody))
		if message != "" {
			return "", fmt.Errorf("Mastodon media upload failed: %s: %s", response.Status, message)
		}
		return "", fmt.Errorf("Mastodon media upload failed: %s", response.Status)
	}

	var result mediaUploadResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode Mastodon media upload response: %w", err)
	}
	if strings.TrimSpace(result.ID) == "" {
		return "", fmt.Errorf("Mastodon media upload response was missing an ID")
	}
	return result.ID, nil
}

func extensionForContentType(contentType string) string {
	switch strings.ToLower(contentType) {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".png"
	}
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
