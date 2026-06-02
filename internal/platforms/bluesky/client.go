package bluesky

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/platforms/netguard"
)

const defaultBaseURL = "https://bsky.social"
const maxExternalThumbBytes = 1_000_000
const maxAdditionalImageBytes = 1_000_000

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

type uploadBlobResponse struct {
	Blob blobRef `json:"blob"`
}

type blobRef struct {
	Type     string   `json:"$type,omitempty"`
	Ref      blobLink `json:"ref"`
	MimeType string   `json:"mimeType"`
	Size     int      `json:"size"`
}

type blobLink struct {
	Link string `json:"$link"`
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
	Type      string          `json:"$type"`
	Text      string          `json:"text"`
	CreatedAt string          `json:"createdAt"`
	Facets    []richTextFacet `json:"facets,omitempty"`
	Embed     any             `json:"embed,omitempty"`
}

type richTextFacet struct {
	Index    byteSliceIndex    `json:"index"`
	Features []richTextFeature `json:"features"`
}

type byteSliceIndex struct {
	ByteStart int `json:"byteStart"`
	ByteEnd   int `json:"byteEnd"`
}

type richTextFeature struct {
	Type string `json:"$type"`
	URI  string `json:"uri"`
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
	URI         string   `json:"uri"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Thumb       *blobRef `json:"thumb,omitempty"`
}

type imageEmbed struct {
	Type   string            `json:"$type"`
	Images []imageEmbedImage `json:"images"`
}

type imageEmbedImage struct {
	Image *blobRef `json:"image"`
	Alt   string   `json:"alt"`
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

func (p *Publisher) PublishPost(ctx context.Context, accountIdentifier string, credentialKey string, content string, metadata domain.BlueskyPostMetadata) error {
	session, err := p.createSession(ctx, accountIdentifier, credentialKey)
	if err != nil {
		return err
	}

	postRecord := postRecordBody{
		Type:      "app.bsky.feed.post",
		Text:      content,
		CreatedAt: p.now().UTC().Format(time.RFC3339),
	}
	if err := p.applyPostLinkMetadata(ctx, session.AccessJWT, &postRecord, metadata); err != nil {
		return err
	}
	if postRecord.Embed == nil {
		if err := p.applyPostImageMetadata(ctx, session.AccessJWT, &postRecord, metadata); err != nil {
			return err
		}
	}

	record := createRecordRequest{
		Collection: "app.bsky.feed.post",
		Repo:       session.DID,
		Record:     postRecord,
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
		Identifier: normalizeIdentifier(accountIdentifier),
		Password:   normalizeAppPassword(credentialKey),
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

func normalizeIdentifier(value string) string {
	identifier := strings.TrimSpace(value)
	identifier = strings.TrimPrefix(identifier, "@")
	if strings.HasPrefix(identifier, "https://bsky.app/profile/") {
		identifier = strings.TrimPrefix(identifier, "https://bsky.app/profile/")
		identifier = strings.Trim(identifier, "/")
	}
	return identifier
}

func normalizeAppPassword(value string) string {
	password := strings.NewReplacer(
		" ", "",
		"\t", "",
		"\n", "",
		"\r", "",
		"\u00a0", "",
		"\u2010", "-",
		"\u2011", "-",
		"\u2012", "-",
		"\u2013", "-",
		"\u2014", "-",
	).Replace(strings.TrimSpace(value))
	return password
}

func (p *Publisher) applyPostLinkMetadata(ctx context.Context, accessJWT string, record *postRecordBody, metadata domain.BlueskyPostMetadata) error {
	streamURL := strings.TrimSpace(metadata.StreamURL)
	if streamURL == "" || !strings.Contains(record.Text, streamURL) {
		return nil
	}

	byteStart := strings.Index(record.Text, streamURL)
	if byteStart < 0 {
		return nil
	}
	byteEnd := byteStart + len(streamURL)
	record.Facets = append(record.Facets, richTextFacet{
		Index: byteSliceIndex{
			ByteStart: byteStart,
			ByteEnd:   byteEnd,
		},
		Features: []richTextFeature{{
			Type: "app.bsky.richtext.facet#link",
			URI:  streamURL,
		}},
	})

	title := strings.TrimSpace(metadata.StreamTitle)
	if title == "" {
		title = streamURL
	}
	card := externalEmbedCard{
		URI:         streamURL,
		Title:       title,
		Description: strings.TrimSpace(metadata.Description),
	}
	if thumbnailDataURL := strings.TrimSpace(metadata.ThumbnailDataURL); thumbnailDataURL != "" {
		contentType, data, err := decodeCardThumbnailDataURL(thumbnailDataURL)
		if err != nil {
			return err
		}
		thumb, err := p.uploadExternalThumb(ctx, accessJWT, contentType, data)
		if err != nil {
			return err
		}
		card.Thumb = thumb
	} else if thumbnailURL := strings.TrimSpace(metadata.ThumbnailURL); thumbnailURL != "" {
		thumb, err := p.fetchAndUploadExternalThumb(ctx, accessJWT, thumbnailURL)
		if err != nil {
			return err
		}
		card.Thumb = thumb
	}

	record.Embed = &externalEmbed{
		Type:     "app.bsky.embed.external",
		External: card,
	}
	return nil
}

func (p *Publisher) applyPostImageMetadata(ctx context.Context, accessJWT string, record *postRecordBody, metadata domain.BlueskyPostMetadata) error {
	if additionalImageDataURL := strings.TrimSpace(metadata.AdditionalImageDataURL); additionalImageDataURL != "" {
		contentType, data, err := decodeAdditionalImageDataURL(additionalImageDataURL)
		if err != nil {
			return err
		}
		image, err := p.uploadAdditionalImage(ctx, accessJWT, contentType, data)
		if err != nil {
			return err
		}
		record.Embed = &imageEmbed{
			Type:   "app.bsky.embed.images",
			Images: []imageEmbedImage{{Image: image, Alt: ""}},
		}
		return nil
	}

	if additionalImageURL := strings.TrimSpace(metadata.AdditionalImageURL); additionalImageURL != "" {
		image, err := p.fetchAndUploadAdditionalImage(ctx, accessJWT, additionalImageURL)
		if err != nil {
			return err
		}
		record.Embed = &imageEmbed{
			Type:   "app.bsky.embed.images",
			Images: []imageEmbedImage{{Image: image, Alt: ""}},
		}
	}
	return nil
}

func decodeCardThumbnailDataURL(dataURL string) (string, []byte, error) {
	return decodeImageDataURL(dataURL, "card thumbnail", maxExternalThumbBytes, "1 MB")
}

func decodeAdditionalImageDataURL(dataURL string) (string, []byte, error) {
	return decodeImageDataURL(dataURL, "additional image", maxAdditionalImageBytes, "1 MB")
}

func decodeImageDataURL(dataURL string, purpose string, maxBytes int, maxBytesLabel string) (string, []byte, error) {
	const marker = ";base64,"
	if !strings.HasPrefix(dataURL, "data:") {
		return "", nil, fmt.Errorf("Bluesky %s upload must be a data URL", purpose)
	}
	index := strings.Index(dataURL, marker)
	if index < 0 {
		return "", nil, fmt.Errorf("Bluesky %s upload must be base64 encoded", purpose)
	}
	contentType := strings.TrimSpace(strings.TrimPrefix(dataURL[:index], "data:"))
	if !strings.HasPrefix(contentType, "image/") {
		return "", nil, fmt.Errorf("Bluesky %s must be an image", purpose)
	}
	encoded := dataURL[index+len(marker):]
	if len(encoded) > base64.StdEncoding.EncodedLen(maxBytes)+4 {
		return "", nil, fmt.Errorf("Bluesky %s must be %s or smaller", purpose, maxBytesLabel)
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, fmt.Errorf("decode Bluesky %s upload: %w", purpose, err)
	}
	if len(data) > maxBytes {
		return "", nil, fmt.Errorf("Bluesky %s must be %s or smaller", purpose, maxBytesLabel)
	}
	return contentType, data, nil
}

func (p *Publisher) fetchAndUploadExternalThumb(ctx context.Context, accessJWT string, thumbnailURL string) (*blobRef, error) {
	if err := netguard.ValidatePublicHTTPURL(ctx, thumbnailURL, "Bluesky card thumbnail"); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, thumbnailURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build Bluesky card thumbnail request: %w", err)
	}
	response, err := p.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch Bluesky card thumbnail: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch Bluesky card thumbnail failed: %s%s", response.Status, readResponseSuffix(response.Body))
	}

	contentType := response.Header.Get("Content-Type")
	if index := strings.Index(contentType, ";"); index >= 0 {
		contentType = contentType[:index]
	}
	contentType = strings.TrimSpace(contentType)
	if !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("Bluesky card thumbnail must be an image")
	}

	data, err := io.ReadAll(io.LimitReader(response.Body, maxExternalThumbBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read Bluesky card thumbnail: %w", err)
	}
	if len(data) > maxExternalThumbBytes {
		return nil, fmt.Errorf("Bluesky card thumbnail must be 1 MB or smaller")
	}

	return p.uploadExternalThumb(ctx, accessJWT, contentType, data)
}

func (p *Publisher) uploadExternalThumb(ctx context.Context, accessJWT string, contentType string, data []byte) (*blobRef, error) {
	return p.uploadBlob(ctx, accessJWT, contentType, data, "card thumbnail")
}

func (p *Publisher) fetchAndUploadAdditionalImage(ctx context.Context, accessJWT string, imageURL string) (*blobRef, error) {
	if err := netguard.ValidatePublicHTTPURL(ctx, imageURL, "Bluesky additional image"); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build Bluesky additional image request: %w", err)
	}
	response, err := p.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch Bluesky additional image: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch Bluesky additional image failed: %s%s", response.Status, readResponseSuffix(response.Body))
	}

	contentType := response.Header.Get("Content-Type")
	if index := strings.Index(contentType, ";"); index >= 0 {
		contentType = contentType[:index]
	}
	contentType = strings.TrimSpace(contentType)
	if !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("Bluesky additional image must be an image")
	}

	data, err := io.ReadAll(io.LimitReader(response.Body, maxAdditionalImageBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read Bluesky additional image: %w", err)
	}
	if len(data) > maxAdditionalImageBytes {
		return nil, fmt.Errorf("Bluesky additional image must be 1 MB or smaller")
	}

	return p.uploadAdditionalImage(ctx, accessJWT, contentType, data)
}

func (p *Publisher) uploadAdditionalImage(ctx context.Context, accessJWT string, contentType string, data []byte) (*blobRef, error) {
	return p.uploadBlob(ctx, accessJWT, contentType, data, "additional image")
}

func (p *Publisher) uploadBlob(ctx context.Context, accessJWT string, contentType string, data []byte, purpose string) (*blobRef, error) {
	upload, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/xrpc/com.atproto.repo.uploadBlob", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("build Bluesky %s upload request: %w", purpose, err)
	}
	upload.Header.Set("Authorization", "Bearer "+accessJWT)
	upload.Header.Set("Content-Type", contentType)

	uploadResponse, err := p.client.Do(upload)
	if err != nil {
		return nil, fmt.Errorf("upload Bluesky %s: %w", purpose, err)
	}
	defer uploadResponse.Body.Close()

	if uploadResponse.StatusCode < 200 || uploadResponse.StatusCode >= 300 {
		return nil, fmt.Errorf("upload Bluesky %s failed: %s%s", purpose, uploadResponse.Status, readResponseSuffix(uploadResponse.Body))
	}

	var result uploadBlobResponse
	if err := json.NewDecoder(uploadResponse.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode Bluesky %s upload response: %w", purpose, err)
	}
	if result.Blob.Ref.Link == "" || result.Blob.MimeType == "" || result.Blob.Size <= 0 {
		return nil, fmt.Errorf("Bluesky %s upload response was missing required fields", purpose)
	}
	return &result.Blob, nil
}

func readResponseSuffix(body io.Reader) string {
	responseBody, _ := io.ReadAll(io.LimitReader(body, 512))
	message := strings.TrimSpace(string(responseBody))
	if message == "" {
		return ""
	}
	return ": " + message
}
