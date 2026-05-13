package anthropic

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/superplanehq/superplane/pkg/agents"
)

type anthropicEvent struct {
	ID      string                  `json:"id"`
	Type    string                  `json:"type"`
	Name    string                  `json:"name"`
	Input   json.RawMessage         `json:"input,omitempty"`
	Content []anthropicContentBlock `json:"content"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

func mapEvent(raw anthropicEvent) (agents.ProviderEvent, bool) {
	switch raw.Type {
	case "agent.message":
		return agents.ProviderEvent{
			ProviderEventID: raw.ID,
			Type:            agents.ProviderEventAssistantMessage,
			Text:            redactSensitive(joinText(raw.Content)),
		}, true

	case "agent.tool_use":
		return agents.ProviderEvent{
			ProviderEventID: raw.ID,
			Type:            agents.ProviderEventToolUseStarted,
			ToolName:        raw.Name,
			ToolCallID:      raw.ID,
			ToolInput:       redactSensitive(renderToolInput(raw.Input)),
		}, true

	case "agent.tool_result":
		return agents.ProviderEvent{
			ProviderEventID: raw.ID,
			Type:            agents.ProviderEventToolUseFinished,
			ToolName:        raw.Name,
			ToolCallID:      raw.ID,
		}, true

	case "session.status_idle":
		return agents.ProviderEvent{Type: agents.ProviderEventTurnCompleted}, true

	case "session.status_terminated", "session.error":
		msg := "agent session terminated"
		if raw.Error != nil && raw.Error.Message != "" {
			msg = raw.Error.Message
		}
		return agents.ProviderEvent{
			Type:         agents.ProviderEventSessionFailed,
			ErrorMessage: msg,
		}, true
	}

	return agents.ProviderEvent{}, false
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

// JWTs are the only secret shape we inject into the preamble today; the
// agent shouldn't be echoing them back through bash or assistant text.
var jwtPattern = regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)

func redactSensitive(s string) string {
	if s == "" {
		return s
	}
	return jwtPattern.ReplaceAllString(s, "<redacted>")
}
