package anthropic

import "time"

// Session represents an Anthropic managed agent session.
type Session struct {
	ID        string        `json:"id"`
	Status    string        `json:"status"`
	Agent     SessionAgent  `json:"agent"`
	Usage     SessionUsage  `json:"usage"`
	Stats     SessionStats  `json:"stats"`
	CreatedAt time.Time     `json:"created_at"`
}

type SessionAgent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SessionUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

type SessionStats struct {
	DurationSeconds float64 `json:"duration_seconds"`
}

// Event represents a session event.
type Event struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Name      string      `json:"name,omitempty"`
	Content   []Content   `json:"content,omitempty"`
	Input     any         `json:"input,omitempty"`
	CreatedAt string      `json:"created_at,omitempty"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// EventsList is the response from listing session events.
type EventsList struct {
	Data    []Event `json:"data"`
	HasMore bool    `json:"has_more"`
}

// CreateSessionRequest is the request body for creating a session.
type CreateSessionRequest struct {
	Agent         string     `json:"agent"`
	EnvironmentID string     `json:"environment_id,omitempty"`
	Resources     []Resource `json:"resources,omitempty"`
}

type Resource struct {
	Type      string `json:"type"`
	FileID    string `json:"file_id"`
	MountPath string `json:"mount_path"`
}

// SendEventRequest is the request body for sending an event to a session.
type SendEventRequest struct {
	Events []UserEvent `json:"events"`
}

type UserEvent struct {
	Type    string    `json:"type"`
	Content []Content `json:"content"`
}
