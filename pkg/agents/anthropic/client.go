package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/agents"
	"time"
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
	VaultIDs      []string
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
	c.setHeaders(req, body != nil)

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
		return nil, fmt.Errorf("anthropic API %d: %s", resp.StatusCode, truncate(string(data), 500))
	}
	return data, nil
}

// openStream opens an SSE GET. Caller must close the returned body.
func (c *Client) openStream(ctx context.Context, path string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("build stream request: %w", err)
	}
	c.setHeaders(req, false)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.streamClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("anthropic: stream returned %d: %s", resp.StatusCode, truncate(string(body), 500))
	}
	return resp.Body, nil
}

func (c *Client) setHeaders(req *http.Request, hasBody bool) {
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", apiVersion)
	req.Header.Set("anthropic-beta", managedAgentsBeta)
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
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
