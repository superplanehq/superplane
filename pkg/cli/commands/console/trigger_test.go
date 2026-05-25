package console

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTriggerInvokesNodeHookWithParameters(t *testing.T) {
	var captured map[string]any
	server := newAPITestServer(t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/canvas-123",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvas": {"spec": {"nodes": [{"id":"node-1","name":"deploy"}]}}}`))
			},
		},
		requestExpectation{
			method: http.MethodPost,
			path:   "/api/v1/canvases/canvas-123/triggers/node-1/hooks/run",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.NoError(t, json.Unmarshal(body, &captured))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			},
		},
	)

	ctx, _ := newCommandContext(t, server.server, "text")
	cmd := &triggerCommand{
		canvasID:   stringPtr("canvas-123"),
		node:       stringPtr("deploy"),
		hook:       stringPtr("run"),
		parameters: stringPtr(`{"environment": "prod"}`),
	}

	require.NoError(t, cmd.Execute(ctx))

	parameters, _ := captured["parameters"].(map[string]any)
	require.NotNil(t, parameters)
	require.Equal(t, "prod", parameters["environment"])
}

func TestTriggerRejectsInvalidJSON(t *testing.T) {
	server := newAPITestServer(t)
	ctx, _ := newCommandContext(t, server.server, "text")

	cmd := &triggerCommand{
		canvasID:   stringPtr("canvas-123"),
		node:       stringPtr("node-1"),
		hook:       stringPtr("run"),
		parameters: stringPtr("not-json"),
	}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid JSON parameters")
}

func TestTriggerRequiresNode(t *testing.T) {
	server := newAPITestServer(t)
	ctx, _ := newCommandContext(t, server.server, "text")

	cmd := &triggerCommand{
		canvasID:   stringPtr("canvas-123"),
		node:       stringPtr(""),
		hook:       stringPtr("run"),
		parameters: stringPtr(""),
	}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--node is required")
}
