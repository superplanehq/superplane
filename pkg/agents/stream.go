package agents

import (
	"bufio"
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
	client  *Client
	store   *Store
	baseURL string // SuperPlane API URL for CLI config
}

func NewStreamHandler(client *Client, store *Store, baseURL string) *StreamHandler {
	return &StreamHandler{client: client, store: store, baseURL: baseURL}
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
func (h *StreamHandler) HandleStream(w http.ResponseWriter, r *http.Request, orgID, userID, canvasID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body streamRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if body.Question == "" {
		http.Error(w, "question is required", http.StatusBadRequest)
		return
	}

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

	writeSSE(w, flusher, map[string]any{"type": "run_started", "model": "claude-sonnet-4-6"})

	// Build prompt with context
	msgCount := 1
	if existingMsgs, _ := h.store.ListMessages(session.ID); len(existingMsgs) > 0 {
		msgCount = len(existingMsgs)
	}
	prompt := h.buildPrompt(session, body, canvasID, msgCount)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	// 1. Open SSE stream from Anthropic BEFORE sending message (per docs)
	sseStream, err := h.client.OpenEventStream(ctx, session.AnthropicSessionID)
	if err != nil {
		log.WithError(err).Error("failed to open Anthropic SSE stream")
		writeSSE(w, flusher, map[string]any{"type": "run_failed", "error": "failed to open event stream"})
		writeSSE(w, flusher, map[string]any{"type": "done"})
		return
	}
	defer sseStream.Close()

	// 2. Send the user message
	if err := h.client.SendMessage(ctx, session.AnthropicSessionID, prompt); err != nil {
		writeSSE(w, flusher, map[string]any{"type": "run_failed", "error": err.Error()})
		writeSSE(w, flusher, map[string]any{"type": "done"})
		return
	}

	// 3. Read events from the SSE stream and forward to the browser
	assistantContent := h.forwardSSEStream(ctx, sseStream, w, flusher)

	// Store assistant response
	if assistantContent != "" {
		h.store.AppendMessage(session.ID, "assistant", assistantContent, "", "")
	}
}

// forwardSSEStream reads events from the Anthropic SSE stream and forwards them to the browser.
func (h *StreamHandler) forwardSSEStream(ctx context.Context, sseStream interface{ Read([]byte) (int, error) }, w http.ResponseWriter, flusher http.Flusher) string {
	scanner := bufio.NewScanner(sseStream)
	// Increase buffer for large events
	scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)

	var assistantContent string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			writeSSE(w, flusher, map[string]any{"type": "run_failed", "error": "timeout"})
			writeSSE(w, flusher, map[string]any{"type": "done"})
			return assistantContent
		default:
		}

		line := scanner.Text()

		// SSE format: "data: {json}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		jsonData := strings.TrimPrefix(line, "data: ")
		if jsonData == "" {
			continue
		}

		var event Event
		if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
			log.WithError(err).WithField("data", jsonData[:min(len(jsonData), 100)]).Warn("failed to parse SSE event")
			continue
		}

		// Process the event
		switch event.Type {
		case "session.status_idle":
			// Turn complete — close the stream
			if assistantContent != "" {
				writeSSE(w, flusher, map[string]any{"type": "final_answer", "output": assistantContent})
			}
			writeSSE(w, flusher, map[string]any{"type": "run_completed"})
			writeSSE(w, flusher, map[string]any{"type": "done"})
			return assistantContent

		case "session.status_terminated", "session.error":
			errMsg := "agent session failed"
			if event.Type == "session.error" {
				errMsg = "agent error"
			}
			writeSSE(w, flusher, map[string]any{"type": "run_failed", "error": errMsg})
			writeSSE(w, flusher, map[string]any{"type": "done"})
			return assistantContent

		case "agent.message":
			text := extractText(event)
			if text != "" {
				assistantContent += text
				writeSSE(w, flusher, map[string]any{"type": "model_delta", "content": text})
			}

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

			// Ignore other event types (span.*, session.status_running, etc.)
		}
	}

	if err := scanner.Err(); err != nil {
		log.WithError(err).Error("SSE stream read error")
		writeSSE(w, flusher, map[string]any{"type": "run_failed", "error": "stream disconnected"})
		writeSSE(w, flusher, map[string]any{"type": "done"})
	}

	return assistantContent
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

// buildPrompt constructs the message to send to Anthropic, injecting CLI setup when needed.
func (h *StreamHandler) buildPrompt(session *ChatSession, body streamRequest, canvasID string, msgCount int) string {
	var parts []string

	// Inject CLI credentials if token is available and this needs setup
	if session.APIToken != nil && *session.APIToken != "" {
		needsSetup := msgCount <= 1 // Only the current user message = first turn

		// Also re-inject if token was recently refreshed
		tokenRefreshed := session.APITokenExpiresAt != nil &&
			time.Until(*session.APITokenExpiresAt) > 50*time.Minute // freshly generated = ~1h remaining

		log.WithFields(log.Fields{
			"msgCount":       msgCount,
			"needsSetup":     needsSetup,
			"tokenRefreshed": tokenRefreshed,
			"hasToken":       session.APIToken != nil && *session.APIToken != "",
			"baseURL":        h.baseURL,
		}).Info("buildPrompt: CLI setup check")

		if needsSetup || tokenRefreshed {
			setup := fmt.Sprintf(`[Agent CLI Setup — run this silently before any SuperPlane commands]
cat > ~/.superplane.yaml << 'SUPERPLANE_CONFIG_EOF'
contexts:
- apiToken: %s
  organization: %s
  organizationId: %s
  url: %s
currentcontext: %s/%s
output: text
SUPERPLANE_CONFIG_EOF`,
				*session.APIToken,
				session.OrganizationID,
				session.OrganizationID,
				h.baseURL,
				h.baseURL,
				session.OrganizationID,
			)
			parts = append(parts, setup)
		}
	}

	// Add canvas context
	if canvasID != "" {
		parts = append(parts, fmt.Sprintf("[Canvas ID: %s]", canvasID))
	}
	if body.AgentContext.Mode == "build" && body.AgentContext.CanvasVersion != "" {
		parts = append(parts, fmt.Sprintf("[Canvas version: %s]", body.AgentContext.CanvasVersion))
	}

	// Add user question
	parts = append(parts, body.Question)

	return strings.Join(parts, "\n\n")
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, data map[string]any) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", b)
	flusher.Flush()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
