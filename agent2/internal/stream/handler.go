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
	"github.com/superplanehq/superplane/pkg/jwt"

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
	signer    *jwt.Signer
}

func NewHandler(cfg HandlerConfig) *Handler {
	return &Handler{
		client: cfg.Client,
		store:  cfg.Store,
		signer: jwt.NewSigner(cfg.JWTSecret),
	}
}

// streamRequest matches the frontend's POST body.
type streamRequest struct {
	Question     string       `json:"question"`
	AgentContext agentContext `json:"agent_context"`
}

type agentContext struct {
	Enabled       bool   `json:"enabled"`
	Mode          string `json:"mode"`
	CanvasVersion string `json:"canvas_version"`
}

// HandleStream handles POST /agents/chats/{chatID}/stream
func (h *Handler) HandleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract chat ID from path: /agents/chats/{chatID}/stream
	chatID := extractChatID(r.URL.Path)
	if chatID == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Validate JWT
	claims, err := h.validateAuth(r)
	if err != nil {
		http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
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

	// Look up chat session
	chat, err := h.store.GetChat(r.Context(), claims.OrgID, chatID)
	if err != nil {
		http.Error(w, "chat not found", http.StatusNotFound)
		return
	}

	// Verify user owns this chat
	if chat.UserID != claims.Subject {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Update initial message if first message
	if chat.InitialMessage == "" {
		truncated := body.Question
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

	// Build prompt with canvas context
	prompt := body.Question
	if body.AgentContext.Mode == "build" && body.AgentContext.CanvasVersion != "" {
		prompt = fmt.Sprintf("[Canvas version: %s]\n\n%s", body.AgentContext.CanvasVersion, body.Question)
	}

	// Send message to Anthropic
	if err := h.client.SendMessage(r.Context(), chat.AnthropicSessionID, prompt); err != nil {
		writeSSE(w, flusher, map[string]any{"type": "run_failed", "error": err.Error()})
		writeSSE(w, flusher, map[string]any{"type": "done"})
		return
	}

	// Poll for completion and stream events
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	h.pollAndStream(ctx, w, flusher, chat.AnthropicSessionID)
}

func (h *Handler) validateAuth(r *http.Request) (*jwt.ScopedTokenClaims, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return nil, fmt.Errorf("missing authorization header")
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, fmt.Errorf("invalid authorization format")
	}

	claims, err := h.signer.ValidateScopedToken(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims.Purpose != "agent-builder" {
		return nil, fmt.Errorf("invalid token purpose")
	}

	return claims, nil
}

func (h *Handler) pollAndStream(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, sessionID string) {
	lastEventCount := 0
	seenEventIDs := make(map[string]bool)

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

		// Get events
		events, err := h.client.ListEvents(ctx, sessionID, 200)
		if err != nil {
			log.WithError(err).Error("failed to list events")
			continue
		}

		// Stream new events
		for i := lastEventCount; i < len(events.Data); i++ {
			event := events.Data[i]
			if seenEventIDs[event.ID] {
				continue
			}
			seenEventIDs[event.ID] = true
			streamEvent(w, flusher, event)
		}
		lastEventCount = len(events.Data)

		// Check if done
		if session.Status == "idle" && session.Usage.OutputTokens > 0 {
			// Extract final assistant message
			for i := len(events.Data) - 1; i >= 0; i-- {
				if events.Data[i].Type == "agent.message" {
					text := extractText(events.Data[i])
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
			"tool_label":   event.Name,
		})
	case "agent.message":
		text := extractText(event)
		if text != "" {
			writeSSE(w, flusher, map[string]any{"type": "model_delta", "content": text})
		}
	}
}

func extractText(event anthropic.Event) string {
	var text string
	for _, c := range event.Content {
		if c.Type == "text" {
			text += c.Text
		}
	}
	return text
}

func extractChatID(path string) string {
	// /agents/chats/{chatID}/stream
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 4 && parts[0] == "agents" && parts[1] == "chats" && parts[3] == "stream" {
		return parts[2]
	}
	return ""
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, data map[string]any) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", b)
	flusher.Flush()
}
