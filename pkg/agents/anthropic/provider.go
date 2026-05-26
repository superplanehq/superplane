// Package anthropic implements agents.Provider against Anthropic's
// managed-agents API.
package anthropic

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/agents"
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

	body := map[string]any{
		"events": []map[string]any{
			{
				"type":    "user.message",
				"content": []map[string]string{{"type": "text", "text": withPreamble(message, opts.ContextPreamble)}},
			},
		},
	}
	if _, err := p.client.executeHTTP(ctx, http.MethodPost, "/sessions/"+providerSessionID+"/events", body); err != nil {
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
		return fmt.Errorf("anthropic: define outcome: %w", err)
	}
	return nil
}

func (p *Provider) StreamEvents(ctx context.Context, providerSessionID string, onEvent func(agents.ProviderEvent) error) error {
	if providerSessionID == "" {
		return fmt.Errorf("anthropic: provider session id is required")
	}

	body, err := p.client.openStream(ctx, "/sessions/"+providerSessionID+"/events/stream")
	if err != nil {
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

func withPreamble(message, preamble string) string {
	if preamble == "" {
		return message
	}
	return preamble + "\n\n" + message
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
	return t == agents.ProviderEventTurnCompleted || t == agents.ProviderEventSessionFailed
}
