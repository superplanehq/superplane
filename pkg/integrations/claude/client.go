package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	defaultBaseURL        = "https://api.anthropic.com/v1"
	anthropicVersionValue = "2023-06-01"
	// anthropicFilesBeta is required to upload files and to reference a file_id
	// from a content block on the Messages API.
	anthropicFilesBeta = "files-api-2025-04-14"
)

type Client struct {
	APIKey  string
	BaseURL string
	http    core.HTTPContext
}

// Message represents a Claude API message.
// Content can be a plain string (for simple text) or []ContentBlock
// (for multi-part content with documents, images and text).
type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// ContentBlock represents a content block in a Claude message.
// Used for text, images, and documents.
type ContentBlock struct {
	Type   string              `json:"type"`
	Text   string              `json:"text,omitempty"`
	Source *ContentBlockSource `json:"source,omitempty"`
}

// ContentBlockSource describes the source of an image/document content block.
// Type "text"/"base64" carry MediaType+Data inline; type "file" references a
// Files API file_id.
type ContentBlockSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
	FileID    string `json:"file_id,omitempty"`
}

type CreateMessageRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	System      string    `json:"system,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
}

type MessageContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type MessageUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type CreateMessageResponse struct {
	ID           string           `json:"id"`
	Type         string           `json:"type"`
	Role         string           `json:"role"`
	Content      []MessageContent `json:"content"`
	Model        string           `json:"model"`
	StopReason   string           `json:"stop_reason"`
	StopSequence string           `json:"stop_sequence,omitempty"`
	Usage        MessageUsage     `json:"usage"`
}

type ModelsResponse struct {
	Data []Model `json:"data"`
}

type Model struct {
	ID string `json:"id"`
}

type claudeErrorResponse struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, err
	}

	return &Client{
		APIKey:  string(apiKey),
		BaseURL: defaultBaseURL,
		http:    httpClient,
	}, nil
}

func (c *Client) Verify() error {
	_, err := c.execRequest(http.MethodGet, c.BaseURL+"/models", nil)
	return err
}

func (c *Client) ListModels() ([]Model, error) {
	responseBody, err := c.execRequest(http.MethodGet, c.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}

	var response ModelsResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal models response: %v", err)
	}

	return response.Data, nil
}

func (c *Client) CreateMessage(req CreateMessageRequest) (*CreateMessageResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	beta := ""
	if requestReferencesFiles(req) {
		beta = anthropicFilesBeta
	}

	responseBody, err := c.execRequestWithBeta(http.MethodPost, c.BaseURL+"/messages", bytes.NewBuffer(reqBody), beta)
	if err != nil {
		return nil, err
	}

	var response CreateMessageResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message response: %v", err)
	}

	return &response, nil
}

// requestReferencesFiles reports whether any message content block references a
// Files API file_id (which requires the files beta header).
func requestReferencesFiles(req CreateMessageRequest) bool {
	for _, m := range req.Messages {
		blocks, ok := m.Content.([]ContentBlock)
		if !ok {
			continue
		}
		for _, b := range blocks {
			if b.Source != nil && b.Source.FileID != "" {
				return true
			}
		}
	}
	return false
}

// UploadFile uploads a file to the Anthropic Files API and returns its file_id.
func (c *Client) UploadFile(content io.Reader, filename string) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return "", fmt.Errorf("create multipart file: %w", err)
	}
	if _, err := io.Copy(part, content); err != nil {
		return "", fmt.Errorf("copy file content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/files", &body)
	if err != nil {
		return "", fmt.Errorf("build upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", anthropicVersionValue)
	req.Header.Set("anthropic-beta", anthropicFilesBeta)

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload file: %w", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("read upload response: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("upload file failed (%d): %s", res.StatusCode, string(resBody))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resBody, &result); err != nil {
		return "", fmt.Errorf("decode upload response: %w", err)
	}
	if result.ID == "" {
		return "", fmt.Errorf("upload returned empty file ID")
	}
	return result.ID, nil
}

// DeleteFile removes an uploaded file. Best-effort cleanup; no-op for empty IDs.
func (c *Client) DeleteFile(fileID string) error {
	if fileID == "" {
		return nil
	}
	_, err := c.execRequestWithBeta(http.MethodDelete, c.BaseURL+"/files/"+url.PathEscape(fileID), nil, anthropicFilesBeta)
	return err
}

func (c *Client) execRequest(method, URL string, body io.Reader) ([]byte, error) {
	return c.execRequestWithBeta(method, URL, body, "")
}

func (c *Client) execRequestWithBeta(method, URL string, body io.Reader, beta string) ([]byte, error) {
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", anthropicVersionValue)
	if beta != "" {
		req.Header.Set("anthropic-beta", beta)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		var apiErr claudeErrorResponse
		var errorMessage string

		// Try to parse the official Anthropic error message
		if err := json.Unmarshal(responseBody, &apiErr); err == nil && apiErr.Error.Message != "" {
			errorMessage = apiErr.Error.Message
		} else {
			errorMessage = string(responseBody)
		}

		// Handle 401 specifically
		if res.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("Claude credentials are invalid or expired: %s", errorMessage)
		}

		return nil, fmt.Errorf("request failed (%d): %s", res.StatusCode, errorMessage)
	}
	return responseBody, nil
}
