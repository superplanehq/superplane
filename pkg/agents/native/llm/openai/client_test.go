package openai

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
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		var req chatCompletionRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.True(t, req.Stream)
		assert.Equal(t, "fast-model", req.Model)

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"content":"Hel"}}]}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"content":"lo"}}]}` + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL, Model: "fast-model"})
	require.NoError(t, err)

	events := streamWithClient(t, client, llm.StreamRequest{
		Messages: []llm.Message{llm.NewUserMessage("Say hello")},
	})

	require.Len(t, events, 2)
	assert.Equal(t, llm.StreamEventTextDelta, events[0].Type)
	assert.Equal(t, "Hel", events[0].Text)
	assert.Equal(t, "lo", events[1].Text)
}

func TestClientStreamsToolCallAfterArgumentChunks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call-1","type":"function","function":{"name":"superplane_app","arguments":"{\"action\""}}]}}]}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":":\"read\"}"}}]}}]}` + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL, Model: "fast-model"})
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
	assert.Equal(t, "call-1", events[0].ToolCall.ID)
	assert.Equal(t, "superplane_app", events[0].ToolCall.Name)
	assert.Equal(t, `{"action":"read"}`, events[0].ToolCall.Input)
}

func TestClientRetriesTransientErrorBeforeStreaming(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			http.Error(w, "temporary", http.StatusTooManyRequests)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"content":"ok"}}]}` + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client, err := New(Config{APIKey: "test-key", BaseURL: server.URL, Model: "fast-model", MaxRetries: 1})
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
