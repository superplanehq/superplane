package agents

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStreamHandler_MissingQuestion(t *testing.T) {
	handler := NewStreamHandler(&Client{}, &Store{})

	body := `{"question":"","agent_context":{"enabled":true,"mode":"inspect"}}`
	req := httptest.NewRequest("POST", "/stream", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleStream(w, req, "org-1", "user-1", "canvas-1")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "question is required") {
		t.Errorf("expected 'question is required' error, got: %s", w.Body.String())
	}
}

func TestStreamHandler_InvalidJSON(t *testing.T) {
	handler := NewStreamHandler(&Client{}, &Store{})

	req := httptest.NewRequest("POST", "/stream", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleStream(w, req, "org-1", "user-1", "canvas-1")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestStreamHandler_MethodNotAllowed(t *testing.T) {
	handler := NewStreamHandler(&Client{}, &Store{})

	req := httptest.NewRequest("GET", "/stream", nil)
	w := httptest.NewRecorder()

	handler.HandleStream(w, req, "org-1", "user-1", "canvas-1")

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestExtractText(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		expected string
	}{
		{
			name: "single text block",
			event: Event{
				Content: []ContentBlock{{Type: "text", Text: "Hello"}},
			},
			expected: "Hello",
		},
		{
			name: "multiple text blocks",
			event: Event{
				Content: []ContentBlock{
					{Type: "text", Text: "Hello "},
					{Type: "text", Text: "World"},
				},
			},
			expected: "Hello World",
		},
		{
			name: "non-text blocks ignored",
			event: Event{
				Content: []ContentBlock{
					{Type: "thinking", Text: "internal thought"},
					{Type: "text", Text: "visible"},
				},
			},
			expected: "visible",
		},
		{
			name:     "empty content",
			event:    Event{Content: nil},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractText(tt.event)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestWriteSSE(t *testing.T) {
	w := httptest.NewRecorder()

	writeSSE(w, w, map[string]any{"type": "model_delta", "content": "Hello"})

	body := w.Body.String()
	if !strings.HasPrefix(body, "data: ") {
		t.Errorf("expected SSE format starting with 'data: ', got: %s", body)
	}
	if !strings.HasSuffix(body, "\n\n") {
		t.Errorf("expected SSE format ending with double newline, got: %s", body)
	}

	// Parse the JSON data
	jsonStr := strings.TrimPrefix(body, "data: ")
	jsonStr = strings.TrimSpace(jsonStr)
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		t.Fatalf("expected valid JSON in SSE data: %v", err)
	}
	if data["type"] != "model_delta" {
		t.Errorf("expected type=model_delta, got %v", data["type"])
	}
	if data["content"] != "Hello" {
		t.Errorf("expected content=Hello, got %v", data["content"])
	}
}

func TestStreamRequest_ParsesCorrectly(t *testing.T) {
	input := `{"question":"List canvases","agent_context":{"enabled":true,"mode":"build","canvas_version":"v123"}}`
	var req streamRequest
	if err := json.NewDecoder(bytes.NewReader([]byte(input))).Decode(&req); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if req.Question != "List canvases" {
		t.Errorf("expected question='List canvases', got %q", req.Question)
	}
	if !req.AgentContext.Enabled {
		t.Error("expected enabled=true")
	}
	if req.AgentContext.Mode != "build" {
		t.Errorf("expected mode=build, got %q", req.AgentContext.Mode)
	}
	if req.AgentContext.CanvasVersion != "v123" {
		t.Errorf("expected canvas_version=v123, got %q", req.AgentContext.CanvasVersion)
	}
}
