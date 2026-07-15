// Package agents defines the provider interface SuperPlane uses to talk to
// managed-agent backends, and a service layer that persists sessions and
// routes streamed events through the event distributor.
package agents

import (
	"context"
	"errors"
)

type ProviderEventType string

const (
	ProviderEventAssistantMessage          ProviderEventType = "assistant_message"
	ProviderEventToolUseStarted            ProviderEventType = "tool_use_started"
	ProviderEventToolUseFinished           ProviderEventType = "tool_use_finished"
	ProviderEventCustomToolUseStarted      ProviderEventType = "custom_tool_use_started"
	ProviderEventCustomToolResultsRequired ProviderEventType = "custom_tool_results_required"
	ProviderEventTurnCompleted             ProviderEventType = "turn_completed"
	ProviderEventSessionFailed             ProviderEventType = "session_failed"
	// Recoverable provider error; the session keeps running.
	ProviderEventSessionNotice          ProviderEventType = "session_notice"
	ProviderEventOutcomeEvaluation      ProviderEventType = "outcome_evaluation"
	ProviderEventOutcomeEvaluationStart ProviderEventType = "outcome_evaluation_start"
	ProviderEventThreadMessageSent      ProviderEventType = "thread_message_sent"
	ProviderEventThreadMessageReceived  ProviderEventType = "thread_message_received"
)

type ProviderEvent struct {
	ProviderEventID string
	Type            ProviderEventType
	Text            string
	Model           string
	ToolName        string
	ToolCallID      string
	// ToolInput is a human-readable rendering of the tool's invocation
	// (e.g. the shell command for bash, or compact JSON for other tools).
	ToolInput          string
	ErrorMessage       string
	OutcomeResult      *OutcomeEvaluation
	CustomToolUse      *CustomToolUse
	CustomToolEventIDs []string

	// Multi-agent thread fields
	AgentName string
	ThreadID  string

	Usage *TokenUsage
}

type TokenUsage struct {
	InputTokens      int64
	OutputTokens     int64
	TotalTokens      int64
	CacheReadTokens  int64
	CacheWriteTokens int64
}

func (u TokenUsage) HasUsage() bool {
	return u.TotalTokens > 0
}

type CustomToolUse struct {
	ID    string
	Name  string
	Input string
}

type CustomToolResult struct {
	CustomToolUseID string
	Content         string
	IsError         bool
}

type CustomToolInputSchema struct {
	Type        string                           `json:"type"`
	Description string                           `json:"description,omitempty"`
	Enum        []string                         `json:"enum,omitempty"`
	Properties  map[string]CustomToolInputSchema `json:"properties,omitempty"`
	Items       *CustomToolInputSchema           `json:"items,omitempty"`
	Required    []string                         `json:"required,omitempty"`
}

func (s CustomToolInputSchema) Map() map[string]any {
	result := map[string]any{"type": s.Type}
	if s.Description != "" {
		result["description"] = s.Description
	}
	if len(s.Enum) > 0 {
		result["enum"] = append([]string(nil), s.Enum...)
	}
	if len(s.Properties) > 0 {
		properties := make(map[string]any, len(s.Properties))
		for name, property := range s.Properties {
			properties[name] = property.Map()
		}
		result["properties"] = properties
	}
	if s.Items != nil {
		result["items"] = s.Items.Map()
	}
	if len(s.Required) > 0 {
		result["required"] = append([]string(nil), s.Required...)
	}
	return result
}

type CustomToolExecutor interface {
	ExecuteCustomTool(ctx context.Context, session AgentSessionContext, toolUse CustomToolUse) CustomToolResult
}

type CustomToolResultSender interface {
	SendCustomToolResults(ctx context.Context, providerSessionID string, results []CustomToolResult) error
}

type OutcomeEvaluation struct {
	Iteration   int
	Result      string // "satisfied", "needs_revision", "max_iterations_reached", "failed", "interrupted"
	Explanation string // grader's prose verdict
}

type AgentSessionContext struct {
	SessionID         string
	ProviderSessionID string
	OrganizationID    string
	UserID            string
	CanvasID          string
}

type FileResource struct {
	FileID    string
	MountPath string
}

type CreateSessionOptions struct {
	InitialContext string
	Title          string
	VaultIDs       []string
	Resources      []FileResource
}

type DefineOutcomeOptions struct {
	// Description is the user-visible goal the provider should work toward.
	Description string
	// Rubric is the grader-facing checklist evaluated after each iteration.
	Rubric string
	// MaxIterations caps the provider's autonomous build/evaluate loop.
	MaxIterations int
	// ContextPreamble is prepended to the description so provider-managed
	// autonomous loops get the same refreshed session context as normal turns.
	ContextPreamble string
}

type CreateSessionResult struct {
	ProviderSessionID string
}

type MessageImage struct {
	MediaType string
	Data      string
}

type SendMessageRequestOptions struct {
	Mode                      string
	AutoLayoutOnUpdateEnabled bool
}

type DefineOutcomeRequestOptions struct {
	AutoLayoutOnUpdateEnabled bool
}

// SendMessageOptions.ContextPreamble is prepended to the user's message so
// providers that need caller context inline (e.g. the canvas/session
// identifiers) receive it without a separate system message.
type SendMessageOptions struct {
	ContextPreamble string
	Images          []MessageImage
}

type Provider interface {
	Name() string
	CreateSession(ctx context.Context, opts CreateSessionOptions) (*CreateSessionResult, error)
	SendMessage(ctx context.Context, providerSessionID, message string, opts SendMessageOptions) error
	InterruptSession(ctx context.Context, providerSessionID string) error
	// DefineOutcome starts a rubric-driven execution loop on the provider side.
	DefineOutcome(ctx context.Context, providerSessionID string, opts DefineOutcomeOptions) error
	// StreamEvents blocks until the provider closes the stream, ctx is
	// cancelled, or onEvent errors. Implementations must not call onEvent
	// after returning.
	StreamEvents(ctx context.Context, providerSessionID string, onEvent func(ProviderEvent) error) error
}

type ProviderSessionCleaner interface {
	Name() string
	DeleteSession(ctx context.Context, providerSessionID string) error
}

type ProviderSessionArchiver interface {
	Name() string
	ArchiveSession(ctx context.Context, providerSessionID string) error
}

type ProviderSessionUsageRetriever interface {
	RetrieveSessionUsage(ctx context.Context, providerSessionID string) (*TokenUsage, error)
}

type ProviderToolSchemaRevisioner interface {
	Name() string
	ToolSchemaRevision() string
}

var ErrSessionAlreadyTerminated = errors.New("agent session already terminated")
var ErrSessionBusy = errors.New("agent session is still processing")
var ErrProviderSessionUnavailable = errors.New("provider session is unavailable")
var ErrInvalidRequest = errors.New("invalid agent request")
