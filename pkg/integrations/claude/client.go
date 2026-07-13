package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path/filepath"
	"strings"

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
	APIKey   string
	AdminKey string
	BaseURL  string
	http     core.HTTPContext
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
	Model        string        `json:"model"`
	Messages     []Message     `json:"messages"`
	System       string        `json:"system,omitempty"`
	MaxTokens    int           `json:"max_tokens,omitempty"`
	Temperature  *float64      `json:"temperature,omitempty"`
	OutputConfig *OutputConfig `json:"output_config,omitempty"`
	Tools        []any         `json:"tools,omitempty"`
}

// OutputConfig configures Claude's response format (structured outputs).
// See https://platform.claude.com/docs/en/build-with-claude/structured-outputs
type OutputConfig struct {
	Format *OutputFormat `json:"format,omitempty"`
}

// OutputFormat constrains the final text response to a JSON schema.
type OutputFormat struct {
	Type   string `json:"type"`   // always "json_schema"
	Schema any    `json:"schema"` // the JSON Schema object
}

type MessageContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// Content carries the nested payload of tool-result blocks (e.g. code
	// execution results referencing generated file_ids). Kept raw so unknown
	// block shapes round-trip untouched.
	Content json.RawMessage `json:"content,omitempty"`
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

// FileMetadata is a file stored in the Anthropic Files API.
type FileMetadata struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Filename     string `json:"filename"`
	MimeType     string `json:"mime_type"`
	SizeBytes    int64  `json:"size_bytes"`
	CreatedAt    string `json:"created_at"`
	Downloadable bool   `json:"downloadable"`
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

	adminKey, _ := ctx.GetConfig("adminKey")

	return &Client{
		APIKey:   string(apiKey),
		AdminKey: string(adminKey),
		BaseURL:  defaultBaseURL,
		http:     httpClient,
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

	// The files beta is needed both to reference uploaded file_ids and to
	// fetch the files that server tools (e.g. code execution) generate.
	beta := ""
	if requestReferencesFiles(req) || len(req.Tools) > 0 {
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

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

// createFormFile is like multipart.Writer.CreateFormFile but lets the caller set
// the part's Content-Type. The stdlib helper hardcodes application/octet-stream,
// which causes the provider to store (and later reject) the file with the wrong
// media type instead of the detected one.
func createFormFile(w *multipart.Writer, fieldname, filename, contentType string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
		quoteEscaper.Replace(fieldname), quoteEscaper.Replace(filename)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	h.Set("Content-Type", contentType)
	return w.CreatePart(h)
}

// UploadFile uploads a file to the Anthropic Files API and returns its file_id.
func (c *Client) UploadFile(content io.Reader, filename, contentType string) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := createFormFile(writer, "file", filepath.Base(filename), contentType)
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

// GetFileMetadata retrieves a single file's metadata (GET /v1/files/{id}).
func (c *Client) GetFileMetadata(fileID string) (*FileMetadata, error) {
	if fileID == "" {
		return nil, fmt.Errorf("file id is required")
	}

	responseBody, err := c.execRequestWithBeta(http.MethodGet, c.BaseURL+"/files/"+url.PathEscape(fileID), nil, anthropicFilesBeta)
	if err != nil {
		return nil, err
	}

	var file FileMetadata
	if err := json.Unmarshal(responseBody, &file); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file metadata: %v", err)
	}
	return &file, nil
}

// DownloadFile fetches a file's raw content (GET /v1/files/{id}/content).
// Only files with downloadable=true (created by code execution, skills, or
// agent sessions) can be downloaded; the API rejects user-uploaded files.
func (c *Client) DownloadFile(fileID string) ([]byte, error) {
	if fileID == "" {
		return nil, fmt.Errorf("file id is required")
	}
	return c.execRequestWithBeta(http.MethodGet, c.FileContentURL(fileID), nil, anthropicFilesBeta)
}

// FileContentURL returns the programmatic download link for a file. Requests
// to it require the API key headers, including the files beta.
func (c *Client) FileContentURL(fileID string) string {
	return c.BaseURL + "/files/" + url.PathEscape(fileID) + "/content"
}

func (c *Client) execRequest(method, URL string, body io.Reader) ([]byte, error) {
	return c.execRequestWithBeta(method, URL, body, "")
}

func (c *Client) execRequestWithBeta(method, URL string, body io.Reader, beta string) ([]byte, error) {
	return c.doRequest(method, URL, body, c.APIKey, beta)
}

func (c *Client) execAdminRequest(method, URL string, body io.Reader) ([]byte, error) {
	return c.doRequest(method, URL, body, c.AdminKey, "")
}

func (c *Client) doRequest(method, URL string, body io.Reader, apiKey, beta string) ([]byte, error) {
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("x-api-key", apiKey)
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

// BatchRequestParams mirrors CreateMessageRequest but is used specifically for
// the per-item "params" object of a Message Batches request.
type BatchRequestParams struct {
	Model        string        `json:"model"`
	Messages     []Message     `json:"messages"`
	System       string        `json:"system,omitempty"`
	MaxTokens    int           `json:"max_tokens,omitempty"`
	Temperature  *float64      `json:"temperature,omitempty"`
	OutputConfig *OutputConfig `json:"output_config,omitempty"`
}

// CreateMessageBatchRequestItem is a single entry in a POST /v1/messages/batches request.
type CreateMessageBatchRequestItem struct {
	CustomID string             `json:"custom_id"`
	Params   BatchRequestParams `json:"params"`
}

// createMessageBatchBody is the JSON body for POST /v1/messages/batches.
type createMessageBatchBody struct {
	Requests []CreateMessageBatchRequestItem `json:"requests"`
}

// MessageBatchRequestCounts summarizes the processing status of every request in a batch.
type MessageBatchRequestCounts struct {
	Processing int `json:"processing"`
	Succeeded  int `json:"succeeded"`
	Errored    int `json:"errored"`
	Canceled   int `json:"canceled"`
	Expired    int `json:"expired"`
}

// MessageBatch is the resource returned by the Message Batches API.
type MessageBatch struct {
	ID               string                    `json:"id"`
	Type             string                    `json:"type"`
	ProcessingStatus string                    `json:"processing_status"`
	RequestCounts    MessageBatchRequestCounts `json:"request_counts"`
	CreatedAt        string                    `json:"created_at,omitempty"`
	EndedAt          string                    `json:"ended_at,omitempty"`
	ExpiresAt        string                    `json:"expires_at,omitempty"`
	ResultsURL       string                    `json:"results_url,omitempty"`
}

// MessageBatchResultError is the error object of an "errored" batch result.
type MessageBatchResultError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// messageBatchResultBody is the "result" object nested in each batch results line.
type messageBatchResultBody struct {
	Type    string                   `json:"type"` // succeeded | errored | canceled | expired
	Message *CreateMessageResponse   `json:"message,omitempty"`
	Error   *MessageBatchResultError `json:"error,omitempty"`
}

// MessageBatchResult is a single line of the batch results JSONL stream.
type MessageBatchResult struct {
	CustomID string                 `json:"custom_id"`
	Result   messageBatchResultBody `json:"result"`
}

const messageBatchesPath = "/messages/batches"

// CreateMessageBatch submits a batch of Messages API requests for asynchronous processing.
func (c *Client) CreateMessageBatch(items []CreateMessageBatchRequestItem) (*MessageBatch, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("at least one request is required")
	}

	body, err := json.Marshal(createMessageBatchBody{Requests: items})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch request: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, c.BaseURL+messageBatchesPath, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	var batch MessageBatch
	if err := json.Unmarshal(responseBody, &batch); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch response: %w", err)
	}
	return &batch, nil
}

// GetMessageBatch retrieves the current status of a batch (GET /v1/messages/batches/{id}).
func (c *Client) GetMessageBatch(batchID string) (*MessageBatch, error) {
	if batchID == "" {
		return nil, fmt.Errorf("batch id is required")
	}

	responseBody, err := c.execRequest(http.MethodGet, c.BaseURL+messageBatchesPath+"/"+url.PathEscape(batchID), nil)
	if err != nil {
		return nil, err
	}

	var batch MessageBatch
	if err := json.Unmarshal(responseBody, &batch); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch: %w", err)
	}
	return &batch, nil
}

// CancelMessageBatch requests cancellation of a still-processing batch
// (POST /v1/messages/batches/{id}/cancel). Already-completed requests are unaffected.
func (c *Client) CancelMessageBatch(batchID string) error {
	if batchID == "" {
		return fmt.Errorf("batch id is required")
	}
	_, err := c.execRequest(http.MethodPost, c.BaseURL+messageBatchesPath+"/"+url.PathEscape(batchID)+"/cancel", nil)
	return err
}

// GetMessageBatchResults streams and parses the batch's results file
// (GET /v1/messages/batches/{id}/results), which is a JSONL document — one JSON
// object per line, in no particular order. Only callable once processing_status
// is "ended".
func (c *Client) GetMessageBatchResults(batchID string) ([]MessageBatchResult, error) {
	if batchID == "" {
		return nil, fmt.Errorf("batch id is required")
	}

	responseBody, err := c.execRequest(http.MethodGet, c.BaseURL+messageBatchesPath+"/"+url.PathEscape(batchID)+"/results", nil)
	if err != nil {
		return nil, err
	}

	var results []MessageBatchResult
	scanner := bufio.NewScanner(bytes.NewReader(responseBody))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var result MessageBatchResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal batch result line: %w", err)
		}
		results = append(results, result)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read batch results: %w", err)
	}
	return results, nil
}
