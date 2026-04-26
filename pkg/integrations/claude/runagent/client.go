package runagent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	defaultBaseURL             = "https://api.anthropic.com/v1"
	anthropicVersionValue      = "2023-06-01"
	anthropicBetaManagedAgents = "managed-agents-2026-04-01"
	sessionEventsPageLimit     = "20"
)

type Client struct {
	APIKey  string
	BaseURL string
	http    core.HTTPContext
}

type claudeErrorResponse struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// CreateManagedSessionRequest is the body for POST /v1/sessions.
type CreateManagedSessionRequest struct {
	// Agent is the agent ID string, or the ID used with AgentVersion for a specific version.
	Agent         string
	AgentVersion  *int
	EnvironmentID string
	VaultIDs      []string
}

// ManagedSession is a subset of the session resource returned by the API.
type ManagedSession struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type ManagedSessionEvent struct {
	Type    string                       `json:"type"`
	Content []ManagedSessionContentBlock `json:"content"`
}

type ManagedSessionContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
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

type listSessionEventsResponse struct {
	Data     []ManagedSessionEvent `json:"data"`
	NextPage string                `json:"next_page"`
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

// buildCreateSessionBody maps CreateManagedSessionRequest to JSON.
func buildCreateSessionBody(req CreateManagedSessionRequest) (createManagedSessionBody, error) {
	if req.EnvironmentID == "" {
		return createManagedSessionBody{}, fmt.Errorf("environmentId is required")
	}

	agentID := strings.TrimSpace(req.Agent)
	if agentID == "" {
		return createManagedSessionBody{}, fmt.Errorf("agent is required")
	}

	var agent any = agentID
	if req.AgentVersion != nil {
		agent = map[string]any{
			"type":    "agent",
			"id":      agentID,
			"version": *req.AgentVersion,
		}
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

func (c *Client) listManagedSessionEventsPage(sessionID, page string) ([]ManagedSessionEvent, string, error) {
	if sessionID == "" {
		return nil, "", fmt.Errorf("session id is required")
	}

	params := url.Values{}
	params.Set("limit", sessionEventsPageLimit)
	params.Set("order", "desc")
	if page != "" {
		params.Set("page", page)
	}

	URL := c.BaseURL + "/sessions/" + url.PathEscape(sessionID) + "/events?" + params.Encode()
	responseBody, err := c.execRequestWithBeta(http.MethodGet, URL, nil, anthropicBetaManagedAgents)
	if err != nil {
		return nil, "", err
	}

	var out listSessionEventsResponse
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal session events: %w", err)
	}
	return out.Data, out.NextPage, nil
}

func (c *Client) GetLastManagedSessionAgentMessage(sessionID string) (string, []ManagedSessionEvent, error) {
	seen := []ManagedSessionEvent{}
	page := ""
	for {
		events, nextPage, err := c.listManagedSessionEventsPage(sessionID, page)
		if err != nil {
			return "", seen, err
		}
		seen = append(seen, events...)

		message := lastAgentMessageFromEvents(events)
		if message != "" || nextPage == "" {
			return message, seen, nil
		}

		page = nextPage
	}
}

func (c *Client) GetLastManagedSessionAgentMessageWithRetry(sessionID string, attempts int, delay time.Duration) (string, []ManagedSessionEvent, error) {
	if attempts < 1 {
		attempts = 1
	}

	var events []ManagedSessionEvent
	for i := 0; i < attempts; i++ {
		var err error
		message, events, err := c.GetLastManagedSessionAgentMessage(sessionID)
		if err != nil {
			return "", events, err
		}

		if message != "" || i == attempts-1 {
			return message, events, nil
		}

		time.Sleep(delay)
	}
	return "", events, nil
}

func lastAgentMessageFromEvents(events []ManagedSessionEvent) string {
	for _, event := range events {
		if event.Type != "agent.message" && event.Type != "assistant.message" {
			continue
		}

		parts := []string{}
		for _, block := range event.Content {
			if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
				parts = append(parts, block.Text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	return ""
}

func managedSessionEventTypes(events []ManagedSessionEvent) string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, event.Type)
	}
	return strings.Join(types, ", ")
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
		if err := json.Unmarshal(responseBody, &apiErr); err == nil && apiErr.Error.Message != "" {
			errorMessage = apiErr.Error.Message
		} else {
			errorMessage = string(responseBody)
		}

		if res.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("Claude credentials are invalid or expired: %s", errorMessage)
		}

		return nil, fmt.Errorf("request failed (%d): %s", res.StatusCode, errorMessage)
	}
	return responseBody, nil
}
