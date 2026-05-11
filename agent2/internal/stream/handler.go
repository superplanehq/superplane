package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/superplanehq/superplane/agent2/internal/anthropic"
	"github.com/superplanehq/superplane/agent2/internal/store"

	log "github.com/sirupsen/logrus"
)

type HandlerConfig struct {
	Client    *anthropic.Client
	Store     *store.Store
	JWTSecret string
}

type Handler struct {
	client    *anthropic.Client
	store     *store.Store
	jwtSecret string
}

func NewHandler(cfg HandlerConfig) *Handler {
	return &Handler{
		client:    cfg.Client,
		store:     cfg.Store,
		jwtSecret: cfg.JWTSecret,
	}
}

// HandleStream handles POST /agents/chats/{chatID}/stream
// The frontend sends a user message and receives SSE events back.
func (h *Handler) HandleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract chat ID from path: /agents/chats/{chatID}/stream
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 || parts[3] != "stream" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	chatID := parts[2]

	// TODO: validate JWT from Authorization header
	// For now, extract org_id from token claims

	// Parse request body
	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if body.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	// Look up chat session
	// TODO: get org_id from JWT claims
	orgID := r.Header.Get("X-Organization-ID")
	chat, err := h.store.GetChat(r.Context(), orgID, chatID)
	if err != nil {
		http.Error(w, "chat not found", http.StatusNotFound)
		return
	}

	// Update initial message if this is the first message
	if chat.InitialMessage == "" {
		truncated := body.Message
		if len(truncated) > 100 {
			truncated = truncated[:100]
		}
		h.store.UpdateInitialMessage(r.Context(), chatID, truncated)
	}

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

	// Send message to Anthropic
	if err := h.client.SendMessage(r.Context(), chat.AnthropicSessionID, body.Message); err != nil {
		writeSSE(w, flusher, map[string]any{"type": "run_failed", "error": err.Error()})
		writeSSE(w, flusher, map[string]any{"type": "done"})
		return
	}

	// Poll for completion and stream events
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	h.pollAndStream(ctx, w, flusher, chat.AnthropicSessionID)
}

func (h *Handler) pollAndStream(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, sessionID string) {
	lastEventCount := 0

	for {
		select {
		case <-ctx.Done():
			writeSSE(w, flusher, map[string]any{"type": "run_failed", "error": "timeout"})
			writeSSE(w, flusher, map[string]any{"type": "done"})
			return
		case <-time.After(2 * time.Second):
		}

		// Check session status
		session, err := h.client.GetSession(ctx, sessionID)
		if err != nil {
			log.WithError(err).Error("failed to poll session")
			continue
		}

		// Get new events
		events, err := h.client.ListEvents(ctx, sessionID, 200)
		if err != nil {
			log.WithError(err).Error("failed to list events")
			continue
		}

		// Stream new events to client
		if len(events.Data) > lastEventCount {
			newEvents := events.Data[lastEventCount:]
			for _, event := range newEvents {
				streamEvent(w, flusher, event)
			}
			lastEventCount = len(events.Data)
		}

		// Check if done
		if session.Status == "idle" && session.Usage.OutputTokens > 0 {
			// Send final answer from last assistant message
			for i := len(events.Data) - 1; i >= 0; i-- {
				if events.Data[i].Type == "agent.message" {
					text := ""
					for _, c := range events.Data[i].Content {
						if c.Type == "text" {
							text += c.Text
						}
					}
					if text != "" {
						writeSSE(w, flusher, map[string]any{"type": "final_answer", "output": text})
					}
					break
				}
			}
			writeSSE(w, flusher, map[string]any{"type": "run_completed"})
			writeSSE(w, flusher, map[string]any{"type": "done"})
			return
		}

		if session.Status == "failed" {
			writeSSE(w, flusher, map[string]any{"type": "run_failed", "error": "agent session failed"})
			writeSSE(w, flusher, map[string]any{"type": "done"})
			return
		}
	}
}

func streamEvent(w http.ResponseWriter, flusher http.Flusher, event anthropic.Event) {
	switch event.Type {
	case "agent.tool_use":
		label := event.Name
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
			"tool_label":   event.Name,
		})
	case "agent.message":
		text := ""
		for _, c := range event.Content {
			if c.Type == "text" {
				text += c.Text
			}
		}
		if text != "" {
			writeSSE(w, flusher, map[string]any{"type": "model_delta", "content": text})
		}
	}
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, data map[string]any) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", b)
	flusher.Flush()
}
