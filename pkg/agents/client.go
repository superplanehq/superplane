package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.anthropic.com/v1"

// Client communicates with the Anthropic Managed Agents API.
type Client struct {
	apiKey     string
	agentID    string
	envID      string
	httpClient *http.Client
	baseURL    string
}

func NewClient(apiKey, agentID, envID string) *Client {
	return &Client{
		apiKey:  apiKey,
		agentID: agentID,
		envID:   envID,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// CreateSession creates a new Anthropic managed agent session.
func (c *Client) CreateSession(ctx context.Context) (*Session, error) {
	body := map[string]any{
		"agent": c.agentID,
	}
	if c.envID != "" {
		body["environment"] = c.envID
	}

	resp, err := c.do(ctx, "POST", "/sessions", body)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	var session Session
	if err := json.Unmarshal(resp, &session); err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}
	return &session, nil
}

// GetSession retrieves the current state of a session.
func (c *Client) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	resp, err := c.do(ctx, "GET", "/sessions/"+sessionID, nil)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	var session Session
	if err := json.Unmarshal(resp, &session); err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}
	return &session, nil
}

// SendMessage sends a user message to the session.
func (c *Client) SendMessage(ctx context.Context, sessionID, message string) error {
	body := map[string]any{
		"type": "user_message",
		"content": []map[string]string{
			{"type": "text", "text": message},
		},
	}

	_, err := c.do(ctx, "POST", "/sessions/"+sessionID+"/events", body)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
}

// ListEvents retrieves events from a session.
func (c *Client) ListEvents(ctx context.Context, sessionID string, limit int) (*EventList, error) {
	path := fmt.Sprintf("/sessions/%s/events?limit=%d", sessionID, limit)
	resp, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	var events EventList
	if err := json.Unmarshal(resp, &events); err != nil {
		return nil, fmt.Errorf("decode events: %w", err)
	}
	return &events, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2025-01-01")
	req.Header.Set("anthropic-beta", "managed-agents-2026-04-01")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}
