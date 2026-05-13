package agents

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_CreateSession(t *testing.T) {
	var capturedReq *http.Request
	var capturedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r.Clone(r.Context())
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "sesn_test123",
			"status": "idle",
		})
	}))
	defer server.Close()

	client := NewClient("test-key", "agent_123", "env_456")
	client.baseURL = server.URL

	session, err := client.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Verify session response
	if session.ID != "sesn_test123" {
		t.Errorf("expected session ID sesn_test123, got %s", session.ID)
	}
	if session.Status != "idle" {
		t.Errorf("expected status idle, got %s", session.Status)
	}

	// Verify request headers
	if capturedReq.Header.Get("x-api-key") != "test-key" {
		t.Errorf("expected x-api-key=test-key, got %s", capturedReq.Header.Get("x-api-key"))
	}
	if capturedReq.Header.Get("anthropic-version") != "2023-06-01" {
		t.Errorf("expected anthropic-version=2023-06-01, got %s", capturedReq.Header.Get("anthropic-version"))
	}
	if capturedReq.Header.Get("anthropic-beta") != "managed-agents-2026-04-01" {
		t.Errorf("expected anthropic-beta=managed-agents-2026-04-01, got %s", capturedReq.Header.Get("anthropic-beta"))
	}

	// Verify request body
	if capturedBody["agent"] != "agent_123" {
		t.Errorf("expected agent=agent_123, got %v", capturedBody["agent"])
	}
	if capturedBody["environment_id"] != "env_456" {
		t.Errorf("expected environment_id=env_456, got %v", capturedBody["environment_id"])
	}

	// Verify method and path
	if capturedReq.Method != "POST" {
		t.Errorf("expected POST, got %s", capturedReq.Method)
	}
	if capturedReq.URL.Path != "/sessions" {
		t.Errorf("expected /sessions, got %s", capturedReq.URL.Path)
	}
}

func TestClient_SendMessage(t *testing.T) {
	var capturedBody map[string]any
	var capturedPath string
	var capturedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	client := NewClient("test-key", "agent_123", "env_456")
	client.baseURL = server.URL

	err := client.SendMessage(context.Background(), "sesn_abc", "Hello world")
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	// Verify path includes session ID
	if capturedPath != "/sessions/sesn_abc/events" {
		t.Errorf("expected /sessions/sesn_abc/events, got %s", capturedPath)
	}

	// Verify beta=true query param
	if capturedQuery != "beta=true" {
		t.Errorf("expected beta=true query param, got %s", capturedQuery)
	}

	// Verify body format: {"events": [{"type": "user.message", "content": [...]}]}
	events, ok := capturedBody["events"].([]any)
	if !ok || len(events) != 1 {
		t.Fatalf("expected events array with 1 element, got %v", capturedBody["events"])
	}

	event := events[0].(map[string]any)
	if event["type"] != "user.message" {
		t.Errorf("expected event type user.message, got %v", event["type"])
	}

	content := event["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(content))
	}

	block := content[0].(map[string]any)
	if block["type"] != "text" {
		t.Errorf("expected content type text, got %v", block["type"])
	}
	if block["text"] != "Hello world" {
		t.Errorf("expected text 'Hello world', got %v", block["text"])
	}
}

func TestClient_GetSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sessions/sesn_abc" {
			t.Errorf("expected path /sessions/sesn_abc, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":     "sesn_abc",
			"status": "running",
			"usage": map[string]any{
				"input_tokens":  100,
				"output_tokens": 50,
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-key", "agent_123", "env_456")
	client.baseURL = server.URL

	session, err := client.GetSession(context.Background(), "sesn_abc")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if session.Status != "running" {
		t.Errorf("expected status running, got %s", session.Status)
	}
	if session.Usage.InputTokens != 100 {
		t.Errorf("expected input_tokens=100, got %d", session.Usage.InputTokens)
	}
	if session.Usage.OutputTokens != 50 {
		t.Errorf("expected output_tokens=50, got %d", session.Usage.OutputTokens)
	}
}

func TestClient_ListEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sessions/sesn_abc/events" {
			t.Errorf("expected path /sessions/sesn_abc/events, got %s", r.URL.Path)
		}
		// Must have beta=true
		if r.URL.Query().Get("beta") != "true" {
			t.Errorf("expected beta=true query param, got %s", r.URL.Query().Get("beta"))
		}
		if r.URL.Query().Get("limit") != "50" {
			t.Errorf("expected limit=50, got %s", r.URL.Query().Get("limit"))
		}

		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "evt_1", "type": "agent.message", "content": []map[string]any{{"type": "text", "text": "Hello"}}},
				{"id": "evt_2", "type": "agent.tool_use", "name": "list_canvases"},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-key", "agent_123", "env_456")
	client.baseURL = server.URL

	events, err := client.ListEvents(context.Background(), "sesn_abc", 50)
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}

	if len(events.Data) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events.Data))
	}
	if events.Data[0].Type != "agent.message" {
		t.Errorf("expected first event type agent.message, got %s", events.Data[0].Type)
	}
	if events.Data[1].Type != "agent.tool_use" {
		t.Errorf("expected second event type agent.tool_use, got %s", events.Data[1].Type)
	}
}

func TestClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"bad request"}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", "agent_123", "env_456")
	client.baseURL = server.URL

	_, err := client.CreateSession(context.Background())
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !contains(err.Error(), "400") {
		t.Errorf("expected error to contain 400, got: %s", err.Error())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
