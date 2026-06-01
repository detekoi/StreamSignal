package discord

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"StreamSignal/internal/domain"
)

const maxEmbedImageBytes = 8_000_000

type Publisher struct {
	client *http.Client
}

type messagePayload struct {
	Content     string              `json:"content"`
	Embeds      []embedPayload      `json:"embeds,omitempty"`
	Attachments []attachmentPayload `json:"attachments,omitempty"`
}

type embedPayload struct {
	Image *embedImagePayload `json:"image,omitempty"`
}

type embedImagePayload struct {
	URL string `json:"url"`
}

type attachmentPayload struct {
	ID       int    `json:"id"`
	Filename string `json:"filename"`
}

type decodedImage struct {
	contentType string
	data        []byte
	filename    string
}

func NewPublisher(client *http.Client) *Publisher {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Publisher{client: client}
}

func (p *Publisher) Publish(ctx context.Context, webhookURL string, content string, metadata domain.DiscordPostMetadata) error {
	thumbnailDataURL := strings.TrimSpace(metadata.ThumbnailDataURL)
	if thumbnailDataURL != "" {
		image, err := decodeImageDataURL(thumbnailDataURL)
		if err != nil {
			return err
		}
		return p.publishMultipart(ctx, webhookURL, content, image)
	}

	payload := messagePayload{Content: content}
	if thumbnailURL := strings.TrimSpace(metadata.ThumbnailURL); thumbnailURL != "" {
		if err := validateHTTPImageURL(thumbnailURL); err != nil {
			return err
		}
		payload.Embeds = []embedPayload{{Image: &embedImagePayload{URL: thumbnailURL}}}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode Discord payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(webhookURL), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build Discord webhook request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	return p.send(request)
}

func (p *Publisher) publishMultipart(ctx context.Context, webhookURL string, content string, image decodedImage) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	payload := messagePayload{
		Content: content,
		Embeds: []embedPayload{{
			Image: &embedImagePayload{URL: "attachment://" + image.filename},
		}},
		Attachments: []attachmentPayload{{
			ID:       0,
			Filename: image.filename,
		}},
	}
	payloadBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode Discord payload: %w", err)
	}
	if err := writer.WriteField("payload_json", string(payloadBody)); err != nil {
		return fmt.Errorf("build Discord multipart payload: %w", err)
	}

	part, err := writer.CreateFormFile("files[0]", image.filename)
	if err != nil {
		return fmt.Errorf("build Discord image upload: %w", err)
	}
	if _, err := part.Write(image.data); err != nil {
		return fmt.Errorf("write Discord image upload: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finalize Discord image upload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(webhookURL), &body)
	if err != nil {
		return fmt.Errorf("build Discord webhook request: %w", err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())

	return p.send(request)
}

func (p *Publisher) send(request *http.Request) error {
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

func validateHTTPImageURL(imageURL string) error {
	if !isHTTPURL(imageURL) {
		return fmt.Errorf("Discord card thumbnail URL must be a valid HTTP or HTTPS URL")
	}
	return nil
}

func isHTTPURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed != nil && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func decodeImageDataURL(dataURL string) (decodedImage, error) {
	const marker = ";base64,"
	if !strings.HasPrefix(dataURL, "data:") {
		return decodedImage{}, fmt.Errorf("Discord card thumbnail upload must be a data URL")
	}
	index := strings.Index(dataURL, marker)
	if index < 0 {
		return decodedImage{}, fmt.Errorf("Discord card thumbnail upload must be base64 encoded")
	}
	contentType := strings.TrimSpace(strings.TrimPrefix(dataURL[:index], "data:"))
	if !strings.HasPrefix(contentType, "image/") {
		return decodedImage{}, fmt.Errorf("Discord card thumbnail must be an image")
	}
	encoded := dataURL[index+len(marker):]
	if len(encoded) > base64.StdEncoding.EncodedLen(maxEmbedImageBytes)+4 {
		return decodedImage{}, fmt.Errorf("Discord card thumbnail must be 8 MB or smaller")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return decodedImage{}, fmt.Errorf("decode Discord card thumbnail upload: %w", err)
	}
	if len(data) > maxEmbedImageBytes {
		return decodedImage{}, fmt.Errorf("Discord card thumbnail must be 8 MB or smaller")
	}
	return decodedImage{
		contentType: contentType,
		data:        data,
		filename:    "streamsignal-card-thumbnail" + extensionForContentType(contentType),
	}, nil
}

func extensionForContentType(contentType string) string {
	switch strings.ToLower(contentType) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		extension := filepath.Ext(strings.TrimPrefix(contentType, "image/"))
		if extension != "" {
			return extension
		}
		return ".img"
	}
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
