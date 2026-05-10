package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

const testEmitEventResponse = `{
	"eventId": "event-001"
}`

const testListEventsResponse = `{
	"events": [
		{
			"id": "event-001",
			"canvasId": "canvas-001",
			"nodeId": "node-001",
			"channel": "output",
			"customName": "Test Event",
			"createdAt": "2025-01-15T10:00:00Z"
		},
		{
			"id": "event-002",
			"canvasId": "canvas-001",
			"nodeId": "node-002",
			"createdAt": "2025-01-15T11:00:00Z"
		}
	]
}`

func TestHandleEmitEvent(t *testing.T) {
	apiClient := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/canvases/canvas-001/nodes/node-001/events", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Contains(t, body, "data")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(testEmitEventResponse))
	})

	ctx := context.Background()
	data := map[string]interface{}{"key": "value", "count": float64(42)}
	result, err := handleEmitEvent(ctx, apiClient, "canvas-001", "node-001", "default", data)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent := result.Content[0].(*mcp.TextContent)
	require.NotEmpty(t, textContent.Text)

	var event map[string]any
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &event))
	require.Equal(t, "event-001", event["event_id"])
	require.Equal(t, "canvas-001", event["canvas_id"])
	require.Equal(t, "node-001", event["node_id"])
}

func TestHandleListEvents(t *testing.T) {
	apiClient := newTestAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/canvases/canvas-001/events", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(testListEventsResponse))
	})

	ctx := context.Background()
	result, err := handleListEvents(ctx, apiClient, "canvas-001")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Content, 1)

	textContent := result.Content[0].(*mcp.TextContent)
	require.NotEmpty(t, textContent.Text)

	var events []map[string]any
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &events))
	require.Len(t, events, 2)

	require.Equal(t, "event-001", events[0]["id"])
	require.Equal(t, "node-001", events[0]["node_id"])
	require.Equal(t, "output", events[0]["channel"])
	require.Equal(t, "Test Event", events[0]["custom_name"])

	require.Equal(t, "event-002", events[1]["id"])
	require.Equal(t, "node-002", events[1]["node_id"])
}
