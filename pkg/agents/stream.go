package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// StreamHandler handles SSE streaming for agent chats.
type StreamHandler struct {
	client *Client
	store  *Store
}

func NewStreamHandler(client *Client, store *Store) *StreamHandler {
	return &StreamHandler{client: client, store: store}
}

// streamRequest is the POST body from the frontend.
type streamRequest struct {
	Question     string       `json:"question"`
	AgentContext agentContext `json:"agent_context"`
}

type agentContext struct {
	Enabled       bool   `json:"enabled"`
	Mode          string `json:"mode"`
	CanvasVersion string `json:"canvas_version"`
}

// HandleStream handles POST /api/v1/agents/chats/{canvas_id}/stream
// Authentication is handled by the caller (middleware extracts user/org from cookie).
func (h *StreamHandler) HandleStream(w http.ResponseWriter, r *http.Request, orgID, userID, canvasID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var body streamRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if body.Question == "" {
		http.Error(w, "question is required", http.StatusBadRequest)
		return
	}

	// Get or create session
	session, err := h.store.FindSession(orgID, userID, canvasID)
	if err != nil {
		http.Error(w, "session not found — call GetOrCreateChat first", http.StatusNotFound)
		return
	}

	// Store user message
	h.store.AppendMessage(session.ID, "user", body.Question, "", "")

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send run_started
	writeSSE(w, flusher, map[string]any{"type": "run_started", "model": "claude-sonnet-4-6"})

	// Build prompt with canvas context
	prompt := body.Question
	if body.AgentContext.Mode == "build" && body.AgentContext.CanvasVersion != "" {
		prompt = fmt.Sprintf("[Canvas version: %s]\n\n%s", body.AgentContext.CanvasVersion, body.Question)
	}

	// Count existing events before sending (to skip old turns when streaming)
	existingEvents, _ := h.client.ListEvents(r.Context(), session.AnthropicSessionID, 200)
	var skipEventIDs map[string]bool
	if existingEvents != nil {
		skipEventIDs = make(map[string]bool, len(existingEvents.Data))
		for _, ev := range existingEvents.Data {
			skipEventIDs[ev.ID] = true
		}
	} else {
		skipEventIDs = make(map[string]bool)
	}

	// Send message to Anthropic
	if err := h.client.SendMessage(r.Context(), session.AnthropicSessionID, prompt); err != nil {
		writeSSE(w, flusher, map[string]any{"type": "run_failed", "error": err.Error()})
		writeSSE(w, flusher, map[string]any{"type": "done"})
		return
	}

	// Poll for completion and stream events
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	assistantContent := h.pollAndStream(ctx, w, flusher, session.AnthropicSessionID, skipEventIDs)

	// Store assistant response
	if assistantContent != "" {
		h.store.AppendMessage(session.ID, "assistant", assistantContent, "", "")
	}
}

func (h *StreamHandler) pollAndStream(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, sessionID string, skipEventIDs map[string]bool) string {
	seenEventIDs := make(map[string]bool)
	// Pre-populate with events from previous turns
	for id := range skipEventIDs {
		seenEventIDs[id] = true
	}
	var assistantContent string

	for {
		select {
		case <-ctx.Done():
			writeSSE(w, flusher, map[string]any{"type": "run_failed", "error": "timeout"})
			writeSSE(w, flusher, map[string]any{"type": "done"})
			return assistantContent
		case <-time.After(2 * time.Second):
		}

		// Check session status
		session, err := h.client.GetSession(ctx, sessionID)
		if err != nil {
			log.WithError(err).Error("failed to poll session")
			continue
		}

		// Get events
		events, err := h.client.ListEvents(ctx, sessionID, 200)
		if err != nil {
			log.WithError(err).Error("failed to list events")
			continue
		}

		// Stream new events
		for _, event := range events.Data {
			if seenEventIDs[event.ID] {
				continue
			}
			seenEventIDs[event.ID] = true

			text := h.streamEvent(w, flusher, event)
			if text != "" {
				assistantContent += text
			}
		}

		// Check if done
		if session.Status == "idle" && session.Usage.OutputTokens > 0 {
			if assistantContent != "" {
				writeSSE(w, flusher, map[string]any{"type": "final_answer", "output": assistantContent})
			}
			writeSSE(w, flusher, map[string]any{"type": "run_completed"})
			writeSSE(w, flusher, map[string]any{"type": "done"})
			return assistantContent
		}

		if session.Status == "failed" {
			writeSSE(w, flusher, map[string]any{"type": "run_failed", "error": "agent session failed"})
			writeSSE(w, flusher, map[string]any{"type": "done"})
			return assistantContent
		}
	}
}

func (h *StreamHandler) streamEvent(w http.ResponseWriter, flusher http.Flusher, event Event) string {
	switch event.Type {
	case "agent.tool_use":
		label := event.Name
		if label == "" {
			label = "working..."
		}
		writeSSE(w, flusher, map[string]any{
			"type":         "tool_started",
			"tool_name":    event.Name,
			"tool_call_id": event.ID,
			"tool_label":   label,
		})
	case "agent.tool_result":
		writeSSE(w, flusher, map[string]any{
			"type":         "tool_finished",
			"tool_name":    event.Name,
			"tool_call_id": event.ID,
		})
	case "agent.message":
		text := extractText(event)
		if text != "" {
			writeSSE(w, flusher, map[string]any{"type": "model_delta", "content": text})
			return text
		}
	}
	return ""
}

func extractText(event Event) string {
	var parts []string
	for _, c := range event.Content {
		if c.Type == "text" && c.Text != "" {
			parts = append(parts, c.Text)
		}
	}
	return strings.Join(parts, "")
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, data map[string]any) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", b)
	flusher.Flush()
}
