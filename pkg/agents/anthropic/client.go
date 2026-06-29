package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/agents"
)

const (
	defaultBaseURL    = "https://api.anthropic.com/v1"
	apiVersion        = "2023-06-01"
	managedAgentsBeta = "managed-agents-2026-04-01,files-api-2025-04-14"
)

type Config struct {
	APIKey        string
	AgentID       string
	EnvironmentID string
	Resources     []agents.FileResource
	BaseURL       string // overridable for tests
	HTTPClient    *http.Client
}

// Client is the HTTP transport for the managed-agents API. It knows about
// authentication and JSON encoding; it does not know about agents.Provider
// concepts.
type Client struct {
	apiKey       string
	baseURL      string
	httpClient   *http.Client
	streamClient *http.Client // SSE connections must not time out
}

func newClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("anthropic: APIKey is required")
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{
		apiKey:       cfg.APIKey,
		baseURL:      strings.TrimRight(baseURL, "/"),
		httpClient:   httpClient,
		streamClient: &http.Client{},
	}, nil
}

// executeHTTP issues a JSON request and returns the decoded body bytes.
func (c *Client) executeHTTP(ctx context.Context, method, path string, body any) ([]byte, error) {
	reader, err := encodeBody(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	contentType := ""
	if body != nil {
		contentType = "application/json"
	}
	c.setHeaders(req, contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, &apiError{
			StatusCode: resp.StatusCode,
			Path:       path,
			Message:    truncate(string(data), 500),
		}
	}
	return data, nil
}

// openStream opens an SSE GET. Caller must close the returned body.
func (c *Client) openStream(ctx context.Context, path string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build stream request: %w", err)
	}
	c.setHeaders(req, "")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.streamClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &apiError{
			StatusCode: resp.StatusCode,
			Path:       path,
			Message:    truncate(string(body), 500),
		}
	}
	return resp.Body, nil
}

type fileMetadata struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
}

type apiError struct {
	StatusCode int
	Path       string
	Message    string
}

type agentMetadata struct {
	System  string          `json:"system"`
	Version int             `json:"version"`
	Tools   json.RawMessage `json:"tools,omitempty"`
}

func (e *apiError) Error() string {
	return fmt.Sprintf("anthropic API %d: %s", e.StatusCode, e.Message)
}

func (c *Client) getAgent(ctx context.Context, agentID string) (agentMetadata, error) {
	data, err := c.executeHTTP(ctx, http.MethodGet, "/agents/"+url.PathEscape(agentID), nil)
	if err != nil {
		return agentMetadata{}, err
	}

	var agent agentMetadata
	if err := json.Unmarshal(data, &agent); err != nil {
		return agentMetadata{}, fmt.Errorf("decode agent: %w", err)
	}

	return agent, nil
}

func (c *Client) updateAgentSystemPrompt(ctx context.Context, agentID string, version int, prompt string) (agentMetadata, error) {
	body := map[string]any{
		"system":  prompt,
		"version": version,
		"tools":   defaultAgentTools(),
	}
	data, err := c.executeHTTP(ctx, http.MethodPost, "/agents/"+url.PathEscape(agentID), body)
	if err != nil {
		return agentMetadata{}, err
	}

	var agent agentMetadata
	if err := json.Unmarshal(data, &agent); err != nil {
		return agentMetadata{}, fmt.Errorf("decode agent: %w", err)
	}

	return agent, nil
}

func (c *Client) listFiles(ctx context.Context) ([]fileMetadata, error) {
	var all []fileMetadata
	afterID := ""

	for {
		query := url.Values{}
		query.Set("limit", "1000")
		if afterID != "" {
			query.Set("after_id", afterID)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/files?"+query.Encode(), nil)
		if err != nil {
			return nil, fmt.Errorf("build files request: %w", err)
		}
		c.setHeaders(req, "")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		var payload struct {
			Data    []fileMetadata `json:"data"`
			HasMore bool           `json:"has_more"`
			LastID  string         `json:"last_id"`
		}

		err = decodeJSONResponse(resp, &payload)
		if err != nil {
			return nil, err
		}

		all = append(all, payload.Data...)
		if !payload.HasMore || payload.LastID == "" {
			return all, nil
		}
		afterID = payload.LastID
	}
}

func (c *Client) uploadFileContent(ctx context.Context, content []byte, filename string) (fileMetadata, error) {
	return c.uploadFileReader(ctx, bytes.NewReader(content), filename)
}

func (c *Client) uploadFileReader(ctx context.Context, file io.Reader, filename string) (fileMetadata, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return fileMetadata{}, fmt.Errorf("create multipart file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fileMetadata{}, fmt.Errorf("copy source file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fileMetadata{}, fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/files", &body)
	if err != nil {
		return fileMetadata{}, fmt.Errorf("build upload request: %w", err)
	}
	c.setHeaders(req, writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fileMetadata{}, err
	}

	var metadata fileMetadata
	if err := decodeJSONResponse(resp, &metadata); err != nil {
		return fileMetadata{}, err
	}
	return metadata, nil
}

func decodeJSONResponse(resp *http.Response, out any) error {
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode >= 400 {
		return &apiError{
			StatusCode: resp.StatusCode,
			Message:    truncate(string(data), 500),
		}
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode body: %w", err)
	}
	return nil
}

func (c *Client) setHeaders(req *http.Request, contentType string) {
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", apiVersion)
	req.Header.Set("anthropic-beta", managedAgentsBeta)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
}

func encodeBody(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}
	return bytes.NewReader(buf), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
