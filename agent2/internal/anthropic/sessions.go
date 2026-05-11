package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// CreateSession creates a new managed agent session.
func (c *Client) CreateSession(ctx context.Context, req CreateSessionRequest) (*Session, error) {
	data, err := c.do(ctx, "POST", "/sessions", req)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	return &session, nil
}

// GetSession retrieves a session by ID.
func (c *Client) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	data, err := c.do(ctx, "GET", "/sessions/"+sessionID, nil)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	return &session, nil
}

// SendMessage sends a user message to a session.
func (c *Client) SendMessage(ctx context.Context, sessionID string, message string) error {
	req := SendEventRequest{
		Events: []UserEvent{
			{
				Type: "user.message",
				Content: []Content{
					{Type: "text", Text: message},
				},
			},
		},
	}

	_, err := c.do(ctx, "POST", "/sessions/"+sessionID+"/events", req)
	return err
}

// ListEvents retrieves events from a session.
func (c *Client) ListEvents(ctx context.Context, sessionID string, limit int) (*EventsList, error) {
	path := fmt.Sprintf("/sessions/%s/events?limit=%d", sessionID, limit)
	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var events EventsList
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("unmarshal events: %w", err)
	}

	return &events, nil
}

// WaitForIdle polls a session until it reaches idle status or context cancels.
func (c *Client) WaitForIdle(ctx context.Context, sessionID string, pollInterval time.Duration) (*Session, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		session, err := c.GetSession(ctx, sessionID)
		if err != nil {
			return nil, err
		}

		if session.Status == "idle" && session.Usage.OutputTokens > 0 {
			return session, nil
		}

		if session.Status == "failed" {
			return session, fmt.Errorf("session failed")
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}
