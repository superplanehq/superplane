// Package native implements a SuperPlane-owned agent loop behind agents.Provider.
package native

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
	agenttools "github.com/superplanehq/superplane/pkg/agents/agent_tools"
	"github.com/superplanehq/superplane/pkg/agents/native/llm"
)

const ProviderName = "native"

const (
	defaultMaxSteps        = 12
	defaultMaxToolCalls    = 20
	defaultMaxContextChars = 120_000
	doomLoopLimit          = 3
)

var (
	errMaxStepsReached    = errors.New("native agent reached max steps")
	errMaxToolCalls       = errors.New("native agent requested too many tool calls")
	errContextBudgetEmpty = errors.New("native agent context budget is too small")
	errDoomLoop           = errors.New("native agent repeated the same tool call too many times")
)

type Config struct {
	Client          llm.Client
	Model           string
	MaxSteps        int
	MaxToolCalls    int
	MaxContextChars int
	SystemPrompt    string
	Store           Store
}

type Provider struct {
	client          llm.Client
	model           string
	maxSteps        int
	maxToolCalls    int
	maxContextChars int
	systemPrompt    string
	store           Store

	mu       sync.Mutex
	sessions map[string]*sessionState
}

var (
	_ agents.Provider                     = (*Provider)(nil)
	_ agents.CustomToolResultSender       = (*Provider)(nil)
	_ agents.ProviderSessionCleaner       = (*Provider)(nil)
	_ agents.ProviderToolSchemaRevisioner = (*Provider)(nil)
)

func New(cfg Config) (*Provider, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("native: LLM client is required")
	}
	maxSteps := cfg.MaxSteps
	if maxSteps <= 0 {
		maxSteps = defaultMaxSteps
	}
	maxToolCalls := cfg.MaxToolCalls
	if maxToolCalls <= 0 {
		maxToolCalls = defaultMaxToolCalls
	}
	maxContextChars := cfg.MaxContextChars
	if maxContextChars <= 0 {
		maxContextChars = defaultMaxContextChars
	}
	store := cfg.Store
	if store == nil {
		store = newMemoryStore()
	}
	return &Provider{
		client:          cfg.Client,
		model:           strings.TrimSpace(cfg.Model),
		maxSteps:        maxSteps,
		maxToolCalls:    maxToolCalls,
		maxContextChars: maxContextChars,
		systemPrompt:    nativeSystemPrompt(cfg.SystemPrompt),
		store:           store,
		sessions:        map[string]*sessionState{},
	}, nil
}

func (p *Provider) Name() string { return ProviderName }

func (p *Provider) ToolSchemaRevision() string {
	return agenttools.SchemaRevision()
}

func (p *Provider) CreateSession(ctx context.Context, _ agents.CreateSessionOptions) (*agents.CreateSessionResult, error) {
	providerSessionID := "native-" + uuid.NewString()
	session := newSessionState(providerSessionID, p.systemPrompt)
	if err := p.store.Save(ctx, session.snapshot()); err != nil {
		return nil, fmt.Errorf("native: create durable session: %w", err)
	}
	p.mu.Lock()
	p.sessions[providerSessionID] = session
	p.mu.Unlock()
	return &agents.CreateSessionResult{ProviderSessionID: providerSessionID}, nil
}

func (p *Provider) SendMessage(ctx context.Context, providerSessionID, message string, opts agents.SendMessageOptions) error {
	session, err := p.session(providerSessionID)
	if err != nil {
		return err
	}
	if err := session.enqueueUserMessage(withPreamble(message, opts.ContextPreamble)); err != nil {
		return err
	}
	return p.saveSession(ctx, session)
}

func (p *Provider) DefineOutcome(ctx context.Context, providerSessionID string, opts agents.DefineOutcomeOptions) error {
	session, err := p.session(providerSessionID)
	if err != nil {
		return err
	}

	prompt := withPreamble(outcomePrompt(opts), opts.ContextPreamble)
	if err := session.enqueueUserMessage(prompt); err != nil {
		return err
	}
	return p.saveSession(ctx, session)
}

func (p *Provider) InterruptSession(ctx context.Context, providerSessionID string) error {
	session, err := p.session(providerSessionID)
	if err != nil {
		return err
	}
	session.interrupt()
	return p.saveSession(ctx, session)
}

func (p *Provider) DeleteSession(ctx context.Context, providerSessionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.sessions[providerSessionID]; !ok {
		err := p.store.Delete(ctx, providerSessionID)
		if errors.Is(err, errNativeSessionNotFound) {
			return fmt.Errorf("%w: native session %q", agents.ErrProviderSessionUnavailable, providerSessionID)
		}
		return err
	}
	delete(p.sessions, providerSessionID)
	return p.store.Delete(ctx, providerSessionID)
}

func (p *Provider) StreamEvents(ctx context.Context, providerSessionID string, onEvent func(agents.ProviderEvent) error) error {
	session, err := p.session(providerSessionID)
	if err != nil {
		return err
	}

	for {
		turn, ok, err := session.nextTurn(p.maxSteps)
		if err != nil || !ok {
			return err
		}
		if err := p.saveSession(ctx, session); err != nil {
			return err
		}

		result, err := p.streamStep(ctx, session, turn, onEvent)
		if err != nil {
			return err
		}
		if len(result.toolCalls) == 0 {
			session.completeTurn(result.assistantBlocks)
			if err := p.saveSession(ctx, session); err != nil {
				return err
			}
			return onEvent(agents.ProviderEvent{Type: agents.ProviderEventTurnCompleted})
		}

		if err := session.pauseForToolResults(result.assistantBlocks, result.toolCalls); err != nil {
			return err
		}
		if err := p.saveSession(ctx, session); err != nil {
			return err
		}
		return onEvent(agents.ProviderEvent{
			Type:               agents.ProviderEventCustomToolResultsRequired,
			CustomToolEventIDs: toolCallIDs(result.toolCalls),
		})
	}
}

func (p *Provider) SendCustomToolResults(ctx context.Context, providerSessionID string, results []agents.CustomToolResult) error {
	session, err := p.session(providerSessionID)
	if err != nil {
		return err
	}
	if err := session.appendToolResults(results); err != nil {
		return err
	}
	return p.saveSession(ctx, session)
}

func (p *Provider) streamStep(
	ctx context.Context,
	session *sessionState,
	turn turnSnapshot,
	onEvent func(agents.ProviderEvent) error,
) (stepResult, error) {
	result := stepResult{}
	history, err := boundedHistory(turn.history, p.maxContextChars)
	if err != nil {
		return stepResult{}, err
	}
	req := llm.StreamRequest{
		SessionID: session.id,
		Model:     p.model,
		Messages:  history,
		Tools:     toolDefinitions(),
	}

	var text strings.Builder
	flushText := func() error {
		if text.Len() == 0 {
			return nil
		}

		content := text.String()
		text.Reset()
		result.assistantBlocks = append(result.assistantBlocks, llm.Block{Type: llm.BlockTypeText, Text: content})
		return onEvent(agents.ProviderEvent{
			ProviderEventID: uuid.NewString(),
			Type:            agents.ProviderEventAssistantMessage,
			Text:            content,
		})
	}

	err = p.client.Stream(ctx, req, func(event llm.StreamEvent) error {
		switch event.Type {
		case llm.StreamEventTextDelta:
			if event.Text == "" {
				return nil
			}
			text.WriteString(event.Text)
			return nil
		case llm.StreamEventToolCall:
			if event.ToolCall == nil {
				return nil
			}
			if err := flushText(); err != nil {
				return err
			}
			toolCall := normalizeToolCall(*event.ToolCall)
			if len(result.toolCalls) >= p.maxToolCalls {
				return fmt.Errorf("%w: limit %d", errMaxToolCalls, p.maxToolCalls)
			}
			if err := session.checkRepeatedToolCall(toolCall); err != nil {
				return err
			}
			result.toolCalls = append(result.toolCalls, toolCall)
			toolCallCopy := toolCall
			result.assistantBlocks = append(result.assistantBlocks, llm.Block{
				Type:     llm.BlockTypeToolUse,
				ToolCall: &toolCallCopy,
			})
			return onEvent(agents.ProviderEvent{
				ProviderEventID: toolCall.ID,
				Type:            agents.ProviderEventCustomToolUseStarted,
				ToolName:        toolCall.Name,
				ToolCallID:      toolCall.ID,
				ToolInput:       toolCall.Input,
				CustomToolUse: &agents.CustomToolUse{
					ID:    toolCall.ID,
					Name:  toolCall.Name,
					Input: toolCall.Input,
				},
			})
		default:
			return nil
		}
	})
	if err != nil {
		return result, err
	}
	return result, flushText()
}

func (p *Provider) session(providerSessionID string) (*sessionState, error) {
	if strings.TrimSpace(providerSessionID) == "" {
		return nil, fmt.Errorf("native: provider session id is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	session, ok := p.sessions[providerSessionID]
	if ok {
		return session, nil
	}

	snapshot, err := p.store.Load(context.Background(), providerSessionID)
	if errors.Is(err, errNativeSessionNotFound) {
		return nil, fmt.Errorf("%w: native session %q", agents.ErrProviderSessionUnavailable, providerSessionID)
	}
	if err != nil {
		return nil, err
	}
	session = sessionFromSnapshot(*snapshot)
	p.sessions[providerSessionID] = session
	return session, nil
}

func (p *Provider) saveSession(ctx context.Context, session *sessionState) error {
	if err := p.store.Save(ctx, session.snapshot()); err != nil {
		return fmt.Errorf("native: save session: %w", err)
	}
	return nil
}

type stepResult struct {
	assistantBlocks []llm.Block
	toolCalls       []llm.ToolCall
}

func toolDefinitions() []llm.ToolDefinition {
	definitions := agenttools.DefaultDefinitions()
	tools := make([]llm.ToolDefinition, 0, len(definitions))
	for _, definition := range definitions {
		tools = append(tools, llm.ToolDefinition{
			Name:        definition.Name(),
			Description: definition.Description(),
			InputSchema: definition.InputSchema().Map(),
		})
	}
	return tools
}

func toolCallIDs(toolCalls []llm.ToolCall) []string {
	ids := make([]string, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		ids = append(ids, toolCall.ID)
	}
	return ids
}

func normalizeToolCall(toolCall llm.ToolCall) llm.ToolCall {
	if toolCall.ID == "" {
		toolCall.ID = uuid.NewString()
	}
	return toolCall
}

func withPreamble(message, preamble string) string {
	if preamble == "" {
		return message
	}
	return preamble + "\n\n" + message
}

func nativeSystemPrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt != "" {
		return prompt
	}
	return DefaultAgentPrompt()
}

func outcomePrompt(opts agents.DefineOutcomeOptions) string {
	parts := []string{
		"[Outcome request]",
		"Work toward this goal:",
		opts.Description,
	}
	if opts.Rubric != "" {
		parts = append(parts, "Evaluation rubric:", opts.Rubric)
	}
	if opts.MaxIterations > 0 {
		parts = append(parts, fmt.Sprintf("Maximum iterations: %d", opts.MaxIterations))
	}
	return strings.Join(parts, "\n")
}
