package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/agents/native/llm"
)

func TestClientStreamsTextDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Equal(t, defaultVersion, r.Header.Get("anthropic-version"))
		var req messagesRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.True(t, req.Stream)
		assert.Equal(t, "claude-test-model", req.Model)
		assert.Equal(t, "system prompt", req.System)
		require.Len(t, req.Messages, 1)
		assert.Equal(t, "user", req.Messages[0].Role)
		assert.Equal(t, "Say hello", req.Messages[0].Content[0].Text)

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hel"}}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"lo"}}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"type":"message_stop"}` + "\n\n"))
	}))
	defer server.Close()

	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL, Model: "claude-test-model"})
	require.NoError(t, err)

	events := streamWithClient(t, client, llm.StreamRequest{
		Messages: []llm.Message{
			llm.NewSystemMessage("system prompt"),
			llm.NewUserMessage("Say hello"),
		},
	})

	require.Len(t, events, 2)
	assert.Equal(t, llm.StreamEventTextDelta, events[0].Type)
	assert.Equal(t, "Hel", events[0].Text)
	assert.Equal(t, "lo", events[1].Text)
}

func TestClientStreamsToolCallAfterArgumentChunks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req messagesRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Len(t, req.Tools, 1)
		assert.Equal(t, "superplane_app", req.Tools[0].Name)

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"tool-1","name":"superplane_app"}}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"action\""}}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":":\"read\"}"}}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"type":"content_block_stop","index":0}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"type":"message_stop"}` + "\n\n"))
	}))
	defer server.Close()

	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL, Model: "claude-test-model"})
	require.NoError(t, err)

	events := streamWithClient(t, client, llm.StreamRequest{
		Messages: []llm.Message{llm.NewUserMessage("Read canvas")},
		Tools: []llm.ToolDefinition{{
			Name:        "superplane_app",
			Description: "Read app",
			InputSchema: map[string]any{
				"type": "object",
			},
		}},
	})

	require.Len(t, events, 1)
	require.NotNil(t, events[0].ToolCall)
	assert.Equal(t, llm.StreamEventToolCall, events[0].Type)
	assert.Equal(t, "tool-1", events[0].ToolCall.ID)
	assert.Equal(t, "superplane_app", events[0].ToolCall.Name)
	assert.Equal(t, `{"action":"read"}`, events[0].ToolCall.Input)
}

func TestClientSendsToolResultsAsUserToolResultBlocks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req messagesRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Len(t, req.Messages, 3)
		assert.Equal(t, "assistant", req.Messages[1].Role)
		assert.Equal(t, "tool_use", req.Messages[1].Content[0].Type)
		assert.Equal(t, "user", req.Messages[2].Role)
		assert.Equal(t, "tool_result", req.Messages[2].Content[0].Type)
		assert.Equal(t, "tool-1", req.Messages[2].Content[0].ToolUseID)

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"done"}}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"type":"message_stop"}` + "\n\n"))
	}))
	defer server.Close()

	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL, Model: "claude-test-model"})
	require.NoError(t, err)

	events := streamWithClient(t, client, llm.StreamRequest{
		Messages: []llm.Message{
			llm.NewUserMessage("Read canvas"),
			llm.NewAssistantMessage([]llm.Block{{Type: llm.BlockTypeToolUse, ToolCall: &llm.ToolCall{ID: "tool-1", Name: "superplane_app", Input: `{"action":"read"}`}}}),
			llm.NewToolResultMessage([]llm.ToolResult{{ToolCallID: "tool-1", Name: "superplane_app", Content: `{"ok":true}`}}),
		},
	})

	require.Len(t, events, 1)
	assert.Equal(t, "done", events[0].Text)
}

func TestClientDowngradesOrphanToolResultToText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req messagesRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Len(t, req.Messages, 1)
		assert.Equal(t, "user", req.Messages[0].Role)
		require.Len(t, req.Messages[0].Content, 1)
		assert.Equal(t, "text", req.Messages[0].Content[0].Type)
		assert.Empty(t, req.Messages[0].Content[0].ToolUseID)
		assert.Contains(t, req.Messages[0].Content[0].Text, "Historical tool result")
		assert.Contains(t, req.Messages[0].Content[0].Text, "superplane_app")

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"ok"}}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"type":"message_stop"}` + "\n\n"))
	}))
	defer server.Close()

	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL, Model: "claude-test-model"})
	require.NoError(t, err)

	events := streamWithClient(t, client, llm.StreamRequest{
		Messages: []llm.Message{
			llm.NewToolResultMessage([]llm.ToolResult{{ToolCallID: "tool-1", Name: "superplane_app", Content: `{"ok":true}`}}),
		},
	})

	require.Len(t, events, 1)
	assert.Equal(t, "ok", events[0].Text)
}

func TestClientRetriesTransientErrorBeforeStreaming(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			http.Error(w, "temporary", http.StatusTooManyRequests)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"ok"}}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"type":"message_stop"}` + "\n\n"))
	}))
	defer server.Close()

	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL, Model: "claude-test-model", MaxRetries: 1})
	require.NoError(t, err)

	events := streamWithClient(t, client, llm.StreamRequest{
		Messages: []llm.Message{llm.NewUserMessage("Say hello")},
	})

	require.Len(t, events, 1)
	assert.Equal(t, "ok", events[0].Text)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts))
}

func streamWithClient(t *testing.T, client *Client, req llm.StreamRequest) []llm.StreamEvent {
	t.Helper()
	events := []llm.StreamEvent{}
	err := client.Stream(context.Background(), req, func(event llm.StreamEvent) error {
		events = append(events, event)
		return nil
	})
	require.NoError(t, err)
	return events
}
