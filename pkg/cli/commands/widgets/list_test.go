package widgets

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListShowsOnlyWidgetNodes(t *testing.T) {
	server := newAPITestServer(t, requestExpectation{
		method: http.MethodGet,
		path:   "/api/v1/canvases/canvas-123",
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"canvas": {
					"spec": {
						"nodes": [
							{"id":"a","name":"trigger","type":"TYPE_TRIGGER"},
							{"id":"b","name":"my-note","type":"TYPE_WIDGET","component":"annotation","position":{"x":10,"y":20}},
							{"id":"c","name":"action","type":"TYPE_ACTION"}
						]
					}
				}
			}`))
		},
	})

	ctx, stdout := newCommandContext(t, server.server, "text")
	cmd := &listCommand{canvasID: stringPtr("canvas-123")}

	require.NoError(t, cmd.Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "my-note")
	require.Contains(t, out, "annotation")
	require.Contains(t, out, "10,20")
	require.NotContains(t, out, "trigger")
	require.NotContains(t, out, "action")
}

func TestListReportsEmptyState(t *testing.T) {
	server := newAPITestServer(t, requestExpectation{
		method: http.MethodGet,
		path:   "/api/v1/canvases/canvas-123",
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"canvas": {"spec": {"nodes": []}}}`))
		},
	})

	ctx, stdout := newCommandContext(t, server.server, "text")
	cmd := &listCommand{canvasID: stringPtr("canvas-123")}

	require.NoError(t, cmd.Execute(ctx))
	require.Contains(t, stdout.String(), "No widget nodes")
}
