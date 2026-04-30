package events

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const nodeEventsResponse = `{
	"events": [
		{
			"id": "evt-001",
			"canvasId": "canvas-001",
			"nodeId": "node-001",
			"channel": "default",
			"data": {"key": "large-payload"},
			"createdAt": "2025-01-15T10:00:00Z"
		}
	],
	"totalCount": 1,
	"hasNextPage": false
}`

const canvasEventsResponse = `{
	"events": [
		{
			"id": "evt-002",
			"canvasId": "canvas-001",
			"nodeId": "node-001",
			"channel": "default",
			"data": {"key": "large-payload"},
			"createdAt": "2025-01-15T10:00:00Z",
			"executions": [
				{"id": "exec-001", "nodeId": "node-001"},
				{"id": "exec-002", "nodeId": "node-001"}
			]
		}
	],
	"totalCount": 1,
	"hasNextPage": false
}`

func TestListNodeEventsReturnsSummaryJSON(t *testing.T) {
	server := newNodeEventsServer(t)
	ctx, stdout := newEventsCommandContext(t, server, "json")
	canvasID := "canvas-001"
	nodeID := "node-001"
	full := false
	cmd := &ListEventsCommand{CanvasID: &canvasID, NodeID: &nodeID, Full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	var result []map[string]string
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Len(t, result, 1)
	require.Equal(t, "evt-001", result[0]["id"])
	require.Equal(t, "default", result[0]["channel"])
	require.Equal(t, "2025-01-15T10:00:00Z", result[0]["createdAt"])

	raw := stdout.String()
	require.NotContains(t, raw, "large-payload")
}

func TestListNodeEventsReturnsFullJSON(t *testing.T) {
	server := newNodeEventsServer(t)
	ctx, stdout := newEventsCommandContext(t, server, "json")
	canvasID := "canvas-001"
	nodeID := "node-001"
	full := true
	cmd := &ListEventsCommand{CanvasID: &canvasID, NodeID: &nodeID, Full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	raw := stdout.String()
	require.Contains(t, raw, "large-payload")
	require.Contains(t, raw, "evt-001")
}

func TestListCanvasEventsReturnsSummaryJSON(t *testing.T) {
	server := newCanvasEventsServer(t)
	ctx, stdout := newEventsCommandContext(t, server, "json")
	canvasID := "canvas-001"
	nodeID := ""
	full := false
	cmd := &ListEventsCommand{CanvasID: &canvasID, NodeID: &nodeID, Full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Len(t, result, 1)
	require.Equal(t, "evt-002", result[0]["id"])
	require.Equal(t, "node-001", result[0]["nodeId"])
	require.Equal(t, "default", result[0]["channel"])
	require.Equal(t, float64(2), result[0]["executions"])
	require.Equal(t, "2025-01-15T10:00:00Z", result[0]["createdAt"])

	raw := stdout.String()
	require.NotContains(t, raw, "large-payload")
	require.NotContains(t, raw, "exec-001")
}

func TestListCanvasEventsReturnsFullJSON(t *testing.T) {
	server := newCanvasEventsServer(t)
	ctx, stdout := newEventsCommandContext(t, server, "json")
	canvasID := "canvas-001"
	nodeID := ""
	full := true
	cmd := &ListEventsCommand{CanvasID: &canvasID, NodeID: &nodeID, Full: &full}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	raw := stdout.String()
	require.Contains(t, raw, "large-payload")
	require.Contains(t, raw, "exec-001")
	require.Contains(t, raw, "exec-002")
}

func newNodeEventsServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/canvases/canvas-001/nodes/node-001/events", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(nodeEventsResponse))
	}))
	t.Cleanup(server.Close)
	return server
}

func newCanvasEventsServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/canvases/canvas-001/events", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(canvasEventsResponse))
	}))
	t.Cleanup(server.Close)
	return server
}

func newEventsCommandContext(
	t *testing.T,
	server *httptest.Server,
	outputFormat string,
) (core.CommandContext, *bytes.Buffer) {
	t.Helper()

	stdout := bytes.NewBuffer(nil)
	renderer, err := core.NewRenderer(outputFormat, stdout)
	require.NoError(t, err)

	cobraCmd := &cobra.Command{}
	cobraCmd.SetOut(stdout)

	config := openapi_client.NewConfiguration()
	config.Servers = openapi_client.ServerConfigurations{
		{URL: server.URL},
	}

	return core.CommandContext{
		Context:  context.Background(),
		Cmd:      cobraCmd,
		API:      openapi_client.NewAPIClient(config),
		Renderer: renderer,
	}, stdout
}
