package agents

import "time"

// Session represents an Anthropic managed agent session.
type Session struct {
	ID     string       `json:"id"`
	Status string       `json:"status"` // "idle", "processing", "failed"
	Agent  SessionAgent `json:"agent"`
	Usage  Usage        `json:"usage"`
}

type SessionAgent struct {
	ID string `json:"id"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// EventList is the response from listing session events.
type EventList struct {
	Data []Event `json:"data"`
}

// Event represents a single event in a session.
type Event struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"` // "user_message", "agent.message", "agent.tool_use", "agent.tool_result"
	Name      string         `json:"name,omitempty"`
	Content   []ContentBlock `json:"content,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type ContentBlock struct {
	Type string `json:"type"` // "text", "tool_use", "tool_result"
	Text string `json:"text,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}
