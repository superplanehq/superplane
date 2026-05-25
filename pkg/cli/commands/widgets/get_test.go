package widgets

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetReturnsWidgetByID(t *testing.T) {
	server := newAPITestServer(t, requestExpectation{
		method: http.MethodGet,
		path:   "/api/v1/canvases/canvas-123",
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"canvas": {
					"spec": {
						"nodes": [
							{"id":"w1","name":"my-note","type":"TYPE_WIDGET","component":"annotation","configuration":{"text":"hi"},"position":{"x":5,"y":7}}
						]
					}
				}
			}`))
		},
	})

	ctx, stdout := newCommandContext(t, server.server, "text")
	ctx.Args = []string{"w1"}
	cmd := &getCommand{canvasID: stringPtr("canvas-123")}

	require.NoError(t, cmd.Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "my-note")
	require.Contains(t, out, "annotation")
	require.Contains(t, out, "hi")
}

func TestGetReturnsWidgetByName(t *testing.T) {
	server := newAPITestServer(t, requestExpectation{
		method: http.MethodGet,
		path:   "/api/v1/canvases/canvas-123",
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"canvas": {
					"spec": {
						"nodes": [
							{"id":"w1","name":"my-note","type":"TYPE_WIDGET","component":"annotation"}
						]
					}
				}
			}`))
		},
	})

	ctx, stdout := newCommandContext(t, server.server, "text")
	ctx.Args = []string{"my-note"}
	cmd := &getCommand{canvasID: stringPtr("canvas-123")}

	require.NoError(t, cmd.Execute(ctx))
	require.Contains(t, stdout.String(), "my-note")
}

func TestGetErrorsWhenWidgetMissing(t *testing.T) {
	server := newAPITestServer(t, requestExpectation{
		method: http.MethodGet,
		path:   "/api/v1/canvases/canvas-123",
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"canvas": {"spec": {"nodes": []}}}`))
		},
	})

	ctx, _ := newCommandContext(t, server.server, "text")
	ctx.Args = []string{"missing"}
	cmd := &getCommand{canvasID: stringPtr("canvas-123")}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing")
}
