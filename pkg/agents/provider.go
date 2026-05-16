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
	ProviderEventAssistantMessage ProviderEventType = "assistant_message"
	ProviderEventToolUseStarted   ProviderEventType = "tool_use_started"
	ProviderEventToolUseFinished  ProviderEventType = "tool_use_finished"
	ProviderEventTurnCompleted    ProviderEventType = "turn_completed"
	ProviderEventSessionFailed    ProviderEventType = "session_failed"
)

type ProviderEvent struct {
	ProviderEventID string
	Type            ProviderEventType
	Text            string
	ToolName        string
	ToolCallID      string
	// ToolInput is a human-readable rendering of the tool's invocation
	// (e.g. the shell command for bash, or compact JSON for other tools).
	ToolInput    string
	ErrorMessage string
}

type CreateSessionOptions struct {
	InitialContext string
	Title          string
}

type CreateSessionResult struct {
	ProviderSessionID string
}

// SendMessageOptions.ContextPreamble is prepended to the user's message so
// providers that need caller context inline (e.g. a CLI token on first turn)
// receive it without a separate system message.
type SendMessageOptions struct {
	ContextPreamble string
}

type Provider interface {
	Name() string
	CreateSession(ctx context.Context, opts CreateSessionOptions) (*CreateSessionResult, error)
	SendMessage(ctx context.Context, providerSessionID, message string, opts SendMessageOptions) error
	InterruptSession(ctx context.Context, providerSessionID string) error
	// StreamEvents blocks until the provider closes the stream, ctx is
	// cancelled, or onEvent errors. Implementations must not call onEvent
	// after returning.
	StreamEvents(ctx context.Context, providerSessionID string, onEvent func(ProviderEvent) error) error
}

type ProviderSessionCleaner interface {
	Name() string
	DeleteSession(ctx context.Context, providerSessionID string) error
}

var ErrSessionAlreadyTerminated = errors.New("agent session already terminated")
