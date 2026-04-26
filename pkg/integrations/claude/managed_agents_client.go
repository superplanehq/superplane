package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Managed session API (Claude Managed Agents). See
// https://platform.claude.com/docs/en/managed-agents/sessions

// CreateManagedSessionRequest is the body for POST /v1/sessions.
type CreateManagedSessionRequest struct {
	// Agent is the agent ID string, or use AgentPin for a specific version.
	Agent string
	// AgentID and AgentVersion pin the session to a version when both are set.
	AgentID       string
	AgentVersion  *int
	EnvironmentID string
	VaultIDs      []string
}

// ManagedSession is a subset of the session resource returned by the API.
type ManagedSession struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// createManagedSessionBody is the JSON body for session creation.
type createManagedSessionBody struct {
	Agent         any      `json:"agent"`
	EnvironmentID string   `json:"environment_id"`
	VaultIDs      []string `json:"vault_ids,omitempty"`
}

type userMessageTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type userMessageEvent struct {
	Type    string                 `json:"type"`
	Content []userMessageTextBlock `json:"content"`
}

// sendSessionEventsRequest wraps events for POST .../sessions/{id}/events.
type sendSessionEventsRequest struct {
	Events []userMessageEvent `json:"events"`
}

// buildCreateSessionBody maps CreateManagedSessionRequest to JSON.
func buildCreateSessionBody(req CreateManagedSessionRequest) (createManagedSessionBody, error) {
	if req.EnvironmentID == "" {
		return createManagedSessionBody{}, fmt.Errorf("environmentId is required")
	}

	var agent any
	switch {
	case req.AgentVersion != nil:
		aid := req.AgentID
		if aid == "" {
			aid = req.Agent
		}
		if strings.TrimSpace(aid) == "" {
			return createManagedSessionBody{}, fmt.Errorf("agent is required when version is set")
		}
		agent = map[string]any{
			"type":    "agent",
			"id":      strings.TrimSpace(aid),
			"version": *req.AgentVersion,
		}
	case strings.TrimSpace(req.Agent) != "":
		agent = strings.TrimSpace(req.Agent)
	default:
		return createManagedSessionBody{}, fmt.Errorf("agent is required")
	}

	return createManagedSessionBody{
		Agent:         agent,
		EnvironmentID: req.EnvironmentID,
		VaultIDs:      nonEmptyStrings(req.VaultIDs),
	}, nil
}

func nonEmptyStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// CreateManagedSession creates a Managed Agents session.
func (c *Client) CreateManagedSession(req CreateManagedSessionRequest) (*ManagedSession, error) {
	body, err := buildCreateSessionBody(req)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session request: %w", err)
	}
	responseBody, err := c.execRequestWithBeta(http.MethodPost, c.BaseURL+"/sessions", bytes.NewBuffer(b), anthropicBetaManagedAgents)
	if err != nil {
		return nil, err
	}
	var out ManagedSession
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session response: %w", err)
	}
	return &out, nil
}

// GetManagedSession retrieves a session by ID (GET /v1/sessions/{id}).
func (c *Client) GetManagedSession(sessionID string) (*ManagedSession, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session id is required")
	}
	URL := c.BaseURL + "/sessions/" + url.PathEscape(sessionID)
	responseBody, err := c.execRequestWithBeta(http.MethodGet, URL, nil, anthropicBetaManagedAgents)
	if err != nil {
		return nil, err
	}
	var out ManagedSession
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}
	return &out, nil
}

// SendManagedSessionUserMessage appends a user.message event to the session.
// The events endpoint uses ?beta=true per the Managed Agents API.
func (c *Client) SendManagedSessionUserMessage(sessionID, text string) error {
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	if text == "" {
		return fmt.Errorf("message text is required")
	}
	URL := c.BaseURL + "/sessions/" + url.PathEscape(sessionID) + "/events?beta=true"
	payload := sendSessionEventsRequest{
		Events: []userMessageEvent{{
			Type: "user.message",
			Content: []userMessageTextBlock{{
				Type: "text",
				Text: text,
			}},
		}},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}
	_, err = c.execRequestWithBeta(http.MethodPost, URL, bytes.NewBuffer(b), anthropicBetaManagedAgents)
	return err
}

// SendManagedSessionInterrupt sends a user.interrupt event (stop agent mid-execution).
func (c *Client) SendManagedSessionInterrupt(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	URL := c.BaseURL + "/sessions/" + url.PathEscape(sessionID) + "/events?beta=true"
	payload := map[string]any{
		"events": []map[string]any{
			{"type": "user.interrupt"},
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal interrupt: %w", err)
	}
	_, err = c.execRequestWithBeta(http.MethodPost, URL, bytes.NewBuffer(b), anthropicBetaManagedAgents)
	return err
}

// DeleteManagedSession removes a session (DELETE /v1/sessions/{id}).
// The API does not allow deleting a running session without interrupting first.
func (c *Client) DeleteManagedSession(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	URL := c.BaseURL + "/sessions/" + url.PathEscape(sessionID)
	_, err := c.execRequestWithBeta(http.MethodDelete, URL, nil, anthropicBetaManagedAgents)
	return err
}
