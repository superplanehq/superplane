// Package anthropic implements agents.Provider against Anthropic's
// managed-agents API.
package anthropic

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/agents"
	agenttools "github.com/superplanehq/superplane/pkg/agents/agent_tools"
)

const ProviderName = "anthropic"

type Provider struct {
	agentID       string
	environmentID string
	resources     []agents.FileResource
	client        *Client
}

func New(cfg Config) (*Provider, error) {
	if strings.TrimSpace(cfg.AgentID) == "" {
		return nil, fmt.Errorf("anthropic: AgentID is required")
	}
	if strings.TrimSpace(cfg.EnvironmentID) == "" {
		return nil, fmt.Errorf("anthropic: EnvironmentID is required")
	}
	client, err := newClient(cfg)
	if err != nil {
		return nil, err
	}
	return &Provider{
		agentID:       cfg.AgentID,
		environmentID: cfg.EnvironmentID,
		resources:     cfg.Resources,
		client:        client,
	}, nil
}

func (p *Provider) Name() string { return ProviderName }

func (p *Provider) ToolSchemaRevision() string {
	return agenttools.SchemaRevision()
}

func (p *Provider) CreateSession(ctx context.Context, opts agents.CreateSessionOptions) (*agents.CreateSessionResult, error) {
	body := map[string]any{
		"agent":          p.agentID,
		"environment_id": p.environmentID,
	}
	if opts.Title != "" {
		body["title"] = opts.Title
	}
	if len(opts.VaultIDs) > 0 {
		body["vault_ids"] = opts.VaultIDs
	}
	// Mount reference files
	resources := opts.Resources
	if len(p.resources) > 0 && len(resources) == 0 {
		resources = p.resources
	}
	if len(resources) > 0 {
		fileResources := make([]map[string]string, len(resources))
		for i, r := range resources {
			fileResources[i] = map[string]string{
				"type":       "file",
				"file_id":    r.FileID,
				"mount_path": r.MountPath,
			}
		}
		body["resources"] = fileResources
	}
	data, err := p.client.executeHTTP(ctx, http.MethodPost, "/sessions", body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: create session: %w", err)
	}

	var resp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("anthropic: decode session: %w", err)
	}
	if resp.ID == "" {
		return nil, fmt.Errorf("anthropic: provider returned empty session id")
	}
	return &agents.CreateSessionResult{ProviderSessionID: resp.ID}, nil
}

func (p *Provider) SendMessage(ctx context.Context, providerSessionID, message string, opts agents.SendMessageOptions) error {
	if providerSessionID == "" {
		return fmt.Errorf("anthropic: provider session id is required")
	}

	content := []map[string]any{
		{"type": "text", "text": withPreamble(message, opts.ContextPreamble)},
	}
	for _, image := range opts.Images {
		content = append(content, map[string]any{
			"type": "image",
			"source": map[string]string{
				"type":       "base64",
				"media_type": image.MediaType,
				"data":       image.Data,
			},
		})
	}

	body := map[string]any{
		"events": []map[string]any{
			{
				"type":    "user.message",
				"content": content,
			},
		},
	}
	if _, err := p.client.executeHTTP(ctx, http.MethodPost, "/sessions/"+providerSessionID+"/events", body); err != nil {
		if isSessionAwaitingToolResults(err) {
			return fmt.Errorf("%w: %w", agents.ErrSessionBusy, err)
		}
		if isProviderSessionUnavailable(err) {
			return fmt.Errorf("%w: %w", agents.ErrProviderSessionUnavailable, err)
		}
		if isClientPayloadError(err) {
			return fmt.Errorf("%w: %w", agents.ErrInvalidRequest, err)
		}
		return fmt.Errorf("anthropic: send message: %w", err)
	}
	return nil
}

func (p *Provider) InterruptSession(ctx context.Context, providerSessionID string) error {
	if providerSessionID == "" {
		return fmt.Errorf("anthropic: provider session id is required")
	}
	body := map[string]any{
		"events": []map[string]any{
			{"type": "user.interrupt"},
		},
	}
	if _, err := p.client.executeHTTP(ctx, http.MethodPost, "/sessions/"+providerSessionID+"/events", body); err != nil {
		if isProviderSessionUnavailable(err) {
			return fmt.Errorf("%w: %w", agents.ErrProviderSessionUnavailable, err)
		}
		return fmt.Errorf("anthropic: interrupt session: %w", err)
	}
	return nil
}

func (p *Provider) DefineOutcome(ctx context.Context, providerSessionID string, opts agents.DefineOutcomeOptions) error {
	if providerSessionID == "" {
		return fmt.Errorf("anthropic: provider session id is required")
	}

	event := map[string]any{
		"type":        "user.define_outcome",
		"description": withPreamble(opts.Description, opts.ContextPreamble),
		"rubric":      map[string]string{"type": "text", "content": opts.Rubric},
	}
	if opts.MaxIterations > 0 {
		event["max_iterations"] = opts.MaxIterations
	}

	body := map[string]any{
		"events": []map[string]any{event},
	}
	if _, err := p.client.executeHTTP(ctx, http.MethodPost, "/sessions/"+providerSessionID+"/events", body); err != nil {
		if isSessionAwaitingToolResults(err) {
			return fmt.Errorf("%w: %w", agents.ErrSessionBusy, err)
		}
		if isProviderSessionUnavailable(err) {
			return fmt.Errorf("%w: %w", agents.ErrProviderSessionUnavailable, err)
		}
		return fmt.Errorf("anthropic: define outcome: %w", err)
	}
	return nil
}

func (p *Provider) SendCustomToolResults(ctx context.Context, providerSessionID string, results []agents.CustomToolResult) error {
	if providerSessionID == "" {
		return fmt.Errorf("anthropic: provider session id is required")
	}
	if len(results) == 0 {
		return nil
	}

	events := make([]map[string]any, 0, len(results))
	for _, result := range results {
		if result.CustomToolUseID == "" {
			return fmt.Errorf("anthropic: custom tool use id is required")
		}
		event := map[string]any{
			"type":               "user.custom_tool_result",
			"custom_tool_use_id": result.CustomToolUseID,
			"content": []map[string]string{
				{"type": "text", "text": result.Content},
			},
		}
		if result.IsError {
			event["is_error"] = true
		}
		events = append(events, event)
	}

	body := map[string]any{"events": events}
	fields := customToolResultsRequestLogFields(providerSessionID, results)
	log.WithFields(fields).Info("anthropic: sending custom tool results")
	startedAt := time.Now()
	if _, err := p.client.executeHTTP(ctx, http.MethodPost, "/sessions/"+providerSessionID+"/events", body); err != nil {
		fields["elapsed_ms"] = time.Since(startedAt).Milliseconds()
		log.WithError(err).WithFields(fields).Warn("anthropic: failed to send custom tool results")
		return fmt.Errorf("anthropic: send custom tool results: %w", err)
	}
	fields["elapsed_ms"] = time.Since(startedAt).Milliseconds()
	log.WithFields(fields).Info("anthropic: sent custom tool results")
	return nil
}

func customToolResultsRequestLogFields(providerSessionID string, results []agents.CustomToolResult) log.Fields {
	errorCount := 0
	contentBytes := 0
	resultIDs := make([]string, 0, len(results))
	for _, result := range results {
		contentBytes += len(result.Content)
		resultIDs = append(resultIDs, result.CustomToolUseID)
		if result.IsError {
			errorCount++
		}
	}

	return log.Fields{
		"provider_session_id":       providerSessionID,
		"custom_tool_result_count":  len(results),
		"custom_tool_result_ids":    resultIDs,
		"custom_tool_error_count":   errorCount,
		"custom_tool_content_bytes": contentBytes,
	}
}

func isSessionAwaitingToolResults(err error) bool {
	var apiErr *apiError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		return false
	}
	return strings.Contains(apiErr.Message, "waiting on responses to events")
}

func isProviderSessionUnavailable(err error) bool {
	var apiErr *apiError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.StatusCode == http.StatusNotFound || apiErr.StatusCode == http.StatusGone {
		return true
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		return false
	}

	message := strings.ToLower(apiErr.Message)
	if !strings.Contains(message, "session") {
		return false
	}

	sessionID := sessionIDFromProviderPath(apiErr.Path)
	if sessionID != "" && strings.Contains(message, strings.ToLower(sessionID)) {
		return strings.Contains(message, "not found") ||
			strings.Contains(message, "does not exist") ||
			strings.Contains(message, "deleted") ||
			strings.Contains(message, "archived")
	}

	return strings.Contains(message, "session does not exist") ||
		strings.Contains(message, "session has been deleted") ||
		strings.Contains(message, "session was deleted") ||
		strings.Contains(message, "session has been archived") ||
		strings.Contains(message, "session was archived")
}

func isClientPayloadError(err error) bool {
	var apiErr *apiError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.StatusCode == http.StatusRequestEntityTooLarge {
		return true
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		return false
	}

	message := strings.ToLower(apiErr.Message)
	return strings.Contains(message, "image") ||
		strings.Contains(message, "payload") ||
		strings.Contains(message, "too large") ||
		strings.Contains(message, "request body") ||
		strings.Contains(message, "body size")
}

func (p *Provider) StreamEvents(ctx context.Context, providerSessionID string, onEvent func(agents.ProviderEvent) error) error {
	if providerSessionID == "" {
		return fmt.Errorf("anthropic: provider session id is required")
	}

	body, err := p.client.openStream(ctx, "/sessions/"+providerSessionID+"/events/stream")
	if err != nil {
		if isProviderSessionUnavailable(err) {
			return fmt.Errorf("%w: %w", agents.ErrProviderSessionUnavailable, err)
		}
		return fmt.Errorf("anthropic: open stream: %w", err)
	}
	defer body.Close()
	return forwardSSE(ctx, body, onEvent)
}

func (p *Provider) DeleteSession(ctx context.Context, providerSessionID string) error {
	if providerSessionID == "" {
		return fmt.Errorf("anthropic: provider session id is required")
	}

	if _, err := p.client.executeHTTP(ctx, http.MethodDelete, "/sessions/"+url.PathEscape(providerSessionID), nil); err != nil {
		return fmt.Errorf("anthropic: delete session: %w", err)
	}

	return nil
}

func (p *Provider) ArchiveSession(ctx context.Context, providerSessionID string) error {
	if providerSessionID == "" {
		return fmt.Errorf("anthropic: provider session id is required")
	}

	if _, err := p.client.executeHTTP(ctx, http.MethodPost, "/sessions/"+url.PathEscape(providerSessionID)+"/archive", nil); err != nil {
		if isProviderSessionUnavailable(err) {
			return fmt.Errorf("%w: %w", agents.ErrProviderSessionUnavailable, err)
		}
		return fmt.Errorf("anthropic: archive session: %w", err)
	}

	return nil
}

func sessionIDFromProviderPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 || parts[0] != "sessions" {
		return ""
	}
	return parts[1]
}

func withPreamble(message, preamble string) string {
	if preamble == "" {
		return message
	}
	return preamble + "\n\n" + message
}

func (p *Provider) RetrieveSessionUsage(ctx context.Context, providerSessionID string) (*agents.TokenUsage, error) {
	data, err := p.client.executeHTTP(ctx, http.MethodGet, "/sessions/"+url.PathEscape(providerSessionID), nil)
	if err != nil {
		return nil, err
	}

	var session struct {
		Usage *anthropicUsage `json:"usage,omitempty"`
	}
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("decode session usage: %w", err)
	}

	return tokenUsage(session.Usage), nil
}

func forwardSSE(ctx context.Context, body io.Reader, onEvent func(agents.ProviderEvent) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		event, ok := parseSSEData(scanner.Text())
		if !ok {
			continue
		}
		if err := onEvent(event); err != nil {
			return err
		}
		if isTerminalEvent(event.Type) {
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("anthropic: stream read: %w", err)
	}
	return nil
}

func parseSSEData(line string) (agents.ProviderEvent, bool) {
	if !strings.HasPrefix(line, "data: ") {
		return agents.ProviderEvent{}, false
	}
	payload := strings.TrimPrefix(line, "data: ")
	if payload == "" {
		return agents.ProviderEvent{}, false
	}
	var raw anthropicEvent
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		log.WithError(err).Debug("anthropic: skipping malformed sse event")
		return agents.ProviderEvent{}, false
	}
	return mapEvent(raw)
}

func isTerminalEvent(t agents.ProviderEventType) bool {
	return t == agents.ProviderEventTurnCompleted ||
		t == agents.ProviderEventSessionFailed
}
