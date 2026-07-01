package messages

import (
	"encoding/json"
	"time"
)

const (
	// AgentStreamRequestedRoutingKey: SendAgentChatMessage -> AgentStreamWorker.
	AgentStreamRequestedRoutingKey = "agent-stream-requested"
	// AgentSessionEventRoutingKey: AgentStreamWorker -> EventDistributer -> websocket clients.
	AgentSessionEventRoutingKey = "agent-session-event"
)

// AgentStreamRequest carries only the session id; the worker re-reads state
// from the DB so we never publish tokens or message content over the queue.
type AgentStreamRequest struct {
	SessionID      string `json:"session_id"`
	OrganizationID string `json:"organization_id"`
	UserID         string `json:"user_id"`
	LockRetryCount int    `json:"lock_retry_count,omitempty"`
}

func PublishAgentStreamRequested(req AgentStreamRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return Publish(CanvasExchange, AgentStreamRequestedRoutingKey, body)
}

func PublishAgentSessionEvent(payload AgentSessionEventMessage) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return Publish(CanvasExchange, AgentSessionEventRoutingKey, body)
}

// AgentSessionEventMessage is the on-wire shape forwarded verbatim to
// websocket clients. Field names match the camelCase convention the rest of
// the canvas WS protocol uses.
type AgentSessionEventMessage struct {
	SessionID string         `json:"sessionId"`
	Event     string         `json:"event"`
	MessageID string         `json:"messageId,omitempty"`
	Message   *AgentMessage  `json:"message,omitempty"`
	Status    string         `json:"status,omitempty"`
	Error     string         `json:"error,omitempty"`
	Extra     map[string]any `json:"extra,omitempty"`
}

// AgentMessage uses time.Time so encoding/json renders RFC 3339 strings.
// timestamppb.Timestamp would serialise as {"seconds":...,"nanos":...},
// which is not what websocket clients expect.
type AgentMessage struct {
	ID         string     `json:"id"`
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCallID string     `json:"toolCallId,omitempty"`
	ToolName   string     `json:"toolName,omitempty"`
	ToolStatus string     `json:"toolStatus,omitempty"`
	CreatedAt  *time.Time `json:"createdAt,omitempty"`
}

// AgentOutcomeEventMessage carries outcome/grader lifecycle events.
type AgentOutcomeEventMessage struct {
	SessionID string `json:"sessionId"`
	Event     string `json:"event"`
	Iteration int    `json:"iteration"`
	Passed    bool   `json:"passed,omitempty"`
	Feedback  string `json:"feedback,omitempty"`
}
