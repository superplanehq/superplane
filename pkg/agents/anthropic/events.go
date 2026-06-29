package anthropic

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/superplanehq/superplane/pkg/agents"
)

type anthropicEvent struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Model      string                  `json:"model,omitempty"`
	Name       string                  `json:"name"`
	ToolName   string                  `json:"tool_name,omitempty"`
	ToolUseID  string                  `json:"tool_use_id,omitempty"`
	Input      json.RawMessage         `json:"input,omitempty"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason *anthropicStopReason    `json:"stop_reason,omitempty"`
	Usage      *anthropicUsage         `json:"usage,omitempty"`
	Error      *struct {
		Message string `json:"message"`
	} `json:"error"`
	// Outcome evaluation fields (from Anthropic SSE stream)
	Iteration   int    `json:"iteration,omitempty"`
	Result      string `json:"result,omitempty"`      // "satisfied", "needs_revision", etc.
	Explanation string `json:"explanation,omitempty"` // grader's prose verdict

	// Multi-agent thread fields
	AgentName           string `json:"agent_name,omitempty"`
	FromAgentName       string `json:"from_agent_name,omitempty"`
	ToAgentName         string `json:"to_agent_name,omitempty"`
	SessionThreadID     string `json:"session_thread_id,omitempty"`
	FromSessionThreadID string `json:"from_session_thread_id,omitempty"`
	ToSessionThreadID   string `json:"to_session_thread_id,omitempty"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type anthropicStopReason struct {
	Type     string   `json:"type"`
	EventIDs []string `json:"event_ids"`
}

type anthropicUsage struct {
	InputTokens              int64  `json:"input_tokens,omitempty"`
	OutputTokens             int64  `json:"output_tokens,omitempty"`
	TotalTokens              int64  `json:"total_tokens,omitempty"`
	CacheReadTokens          int64  `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens         int64  `json:"cache_write_tokens,omitempty"`
	CacheReadInputTokens     int64  `json:"cache_read_input_tokens,omitempty"`
	CacheCreationInputTokens int64  `json:"cache_creation_input_tokens,omitempty"`
	ServerToolUseInputTokens int64  `json:"server_tool_use_input_tokens,omitempty"`
	ServiceTier              string `json:"service_tier,omitempty"`
	CacheCreation            struct {
		Ephemeral5mInputTokens int64 `json:"ephemeral_5m_input_tokens,omitempty"`
		Ephemeral1hInputTokens int64 `json:"ephemeral_1h_input_tokens,omitempty"`
	} `json:"cache_creation,omitempty"`
}

func mapEvent(raw anthropicEvent) (agents.ProviderEvent, bool) {
	switch raw.Type {
	case "agent.message":
		return assistantMessageEvent(raw), true
	case "agent.tool_use":
		return toolEvent(raw, agents.ProviderEventToolUseStarted), true
	case "agent.tool_result":
		return toolEvent(raw, agents.ProviderEventToolUseFinished), true
	case "agent.custom_tool_use":
		return customToolUseEvent(raw), true
	case "session.status_idle":
		return idleEvent(raw)
	case "session.status_terminated":
		return sessionFailedEvent(raw), true
	case "session.error":
		// Recoverable error: surface a notice but keep streaming. Only
		// status_terminated is terminal.
		return sessionNoticeEvent(raw), true
	case "span.outcome_evaluation_start":
		return outcomeEvaluationStartEvent(raw), true
	case "agent.thread_message_sent":
		return threadMessageEvent(raw, agents.ProviderEventThreadMessageSent, raw.ToAgentName, raw.ToSessionThreadID), true
	case "agent.thread_message_received":
		return threadMessageEvent(raw, agents.ProviderEventThreadMessageReceived, raw.FromAgentName, raw.FromSessionThreadID), true
	case "span.outcome_evaluation_end":
		return outcomeEvaluationEndEvent(raw), true
	}

	return agents.ProviderEvent{}, false
}

func assistantMessageEvent(raw anthropicEvent) agents.ProviderEvent {
	return agents.ProviderEvent{
		ProviderEventID: raw.ID,
		Type:            agents.ProviderEventAssistantMessage,
		Text:            redactSensitive(joinText(raw.Content)),
	}
}

func toolEvent(raw anthropicEvent, eventType agents.ProviderEventType) agents.ProviderEvent {
	return agents.ProviderEvent{
		ProviderEventID: toolUseID(raw),
		Type:            eventType,
		ToolName:        providerToolName(raw),
		ToolCallID:      toolUseID(raw),
		ToolInput:       redactSensitive(renderToolInput(raw.Input)),
	}
}

func customToolUseEvent(raw anthropicEvent) agents.ProviderEvent {
	id := customToolUseID(raw)
	input := strings.TrimSpace(string(raw.Input))
	return agents.ProviderEvent{
		ProviderEventID: id,
		Type:            agents.ProviderEventCustomToolUseStarted,
		ToolName:        providerToolName(raw),
		ToolCallID:      id,
		ToolInput:       redactSensitive(input),
		CustomToolUse: &agents.CustomToolUse{
			ID:    id,
			Name:  providerToolName(raw),
			Input: input,
		},
	}
}

func idleEvent(raw anthropicEvent) (agents.ProviderEvent, bool) {
	if raw.StopReason == nil || raw.StopReason.Type == "" || raw.StopReason.Type == "end_turn" {
		return agents.ProviderEvent{
			Type:  agents.ProviderEventTurnCompleted,
			Model: raw.Model,
			Usage: tokenUsage(raw.Usage),
		}, true
	}

	if raw.StopReason.Type == "requires_action" {
		return agents.ProviderEvent{
			Type:               agents.ProviderEventCustomToolResultsRequired,
			CustomToolEventIDs: append([]string(nil), raw.StopReason.EventIDs...),
		}, true
	}

	return agents.ProviderEvent{
		Type:  agents.ProviderEventTurnCompleted,
		Model: raw.Model,
		Usage: tokenUsage(raw.Usage),
	}, true
}

func tokenUsage(raw *anthropicUsage) *agents.TokenUsage {
	if raw == nil {
		return nil
	}

	usage := &agents.TokenUsage{
		InputTokens:      raw.InputTokens,
		OutputTokens:     raw.OutputTokens,
		TotalTokens:      raw.TotalTokens,
		CacheReadTokens:  firstNonZero(raw.CacheReadTokens, raw.CacheReadInputTokens),
		CacheWriteTokens: cacheCreationTokens(raw),
	}
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens + usage.CacheReadTokens + usage.CacheWriteTokens + raw.ServerToolUseInputTokens
	}
	if usage.TotalTokens == 0 {
		return nil
	}
	return usage
}

func cacheCreationTokens(raw *anthropicUsage) int64 {
	return firstNonZero(
		raw.CacheWriteTokens,
		raw.CacheCreationInputTokens,
		raw.CacheCreation.Ephemeral5mInputTokens+raw.CacheCreation.Ephemeral1hInputTokens,
	)
}

func firstNonZero(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func sessionFailedEvent(raw anthropicEvent) agents.ProviderEvent {
	msg := "agent session terminated"
	if raw.Error != nil && raw.Error.Message != "" {
		msg = raw.Error.Message
	}
	return agents.ProviderEvent{
		Type:         agents.ProviderEventSessionFailed,
		ErrorMessage: msg,
	}
}

func sessionNoticeEvent(raw anthropicEvent) agents.ProviderEvent {
	msg := "the agent hit a recoverable error and is retrying"
	if raw.Error != nil && raw.Error.Message != "" {
		msg = raw.Error.Message
	}
	return agents.ProviderEvent{
		Type:         agents.ProviderEventSessionNotice,
		ErrorMessage: redactSensitive(msg),
	}
}

func outcomeEvaluationStartEvent(raw anthropicEvent) agents.ProviderEvent {
	return agents.ProviderEvent{
		Type: agents.ProviderEventOutcomeEvaluationStart,
		OutcomeResult: &agents.OutcomeEvaluation{
			Iteration: raw.Iteration,
		},
	}
}

func outcomeEvaluationEndEvent(raw anthropicEvent) agents.ProviderEvent {
	return agents.ProviderEvent{
		Type: agents.ProviderEventOutcomeEvaluation,
		OutcomeResult: &agents.OutcomeEvaluation{
			Iteration:   raw.Iteration,
			Result:      raw.Result,
			Explanation: raw.Explanation,
		},
	}
}

func threadMessageEvent(raw anthropicEvent, eventType agents.ProviderEventType, agentName, threadID string) agents.ProviderEvent {
	return agents.ProviderEvent{
		ProviderEventID: raw.ID,
		Type:            eventType,
		Text:            redactSensitive(joinText(raw.Content)),
		AgentName:       agentName,
		ThreadID:        threadID,
	}
}

// toolUseID is the tool-call identifier shared by `agent.tool_use` and the
// matching `agent.tool_result`. We key our DB upsert on it so the two
// events collapse into one row (started → finished) instead of producing
// two distinct ones. Falls back to the event id when the field is missing
// for compatibility with stripped-down provider responses.
func toolUseID(raw anthropicEvent) string {
	if raw.ToolUseID != "" {
		return raw.ToolUseID
	}
	return raw.ID
}

func customToolUseID(raw anthropicEvent) string {
	return raw.ID
}

func providerToolName(raw anthropicEvent) string {
	if raw.Name != "" {
		return raw.Name
	}
	return raw.ToolName
}

// renderToolInput prefers the `command` field for shell-style tools and
// falls back to compact JSON for anything else.
func renderToolInput(input json.RawMessage) string {
	if len(input) == 0 {
		return ""
	}
	var fields map[string]any
	if err := json.Unmarshal(input, &fields); err == nil {
		if cmd, ok := fields["command"].(string); ok && cmd != "" {
			return cmd
		}
	}
	return strings.TrimSpace(string(input))
}

func joinText(blocks []anthropicContentBlock) string {
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "")
}

// Defense in depth: the agent is no longer handed any credential, but if a
// JWT-shaped secret ever surfaces in tool output or assistant text we still
// redact it rather than relay it to the user.
var jwtPattern = regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)

func redactSensitive(s string) string {
	if s == "" {
		return s
	}
	return jwtPattern.ReplaceAllString(s, "<redacted>")
}
