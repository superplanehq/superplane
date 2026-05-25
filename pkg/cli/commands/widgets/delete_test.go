package widgets

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeleteRemovesWidgetAndEdges(t *testing.T) {
	var captured map[string]any
	server := newAPITestServer(t,
		// changeManagementEnabled lookup
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/canvas-123",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvas": {"spec": {"nodes": [{"id":"w1","name":"note","type":"TYPE_WIDGET"}], "edges": [{"sourceId":"w1","targetId":"x"},{"sourceId":"x","targetId":"y"}], "changeManagement": {"enabled": false}}}}`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/canvas-123/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions": [{"metadata": {"id": "ver-1", "state": "STATE_DRAFT"}}]}`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/canvas-123/versions/ver-1",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version": {"metadata":{"id":"ver-1"}, "spec": {"nodes": [{"id":"w1","name":"note","type":"TYPE_WIDGET"}], "edges": [{"sourceId":"w1","targetId":"x"},{"sourceId":"x","targetId":"y"}]}}}`))
			},
		},
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/canvas-123/versions",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.NoError(t, json.Unmarshal(body, &captured))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version": {"metadata":{"id":"ver-1"}, "spec":{"nodes": [], "edges": []}}}`))
			},
		},
		requestExpectation{
			method: http.MethodPatch,
			path:   "/api/v1/canvases/canvas-123/versions/ver-1/publish",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			},
		},
	)

	ctx, _ := newCommandContext(t, server.server, "text")
	ctx.Args = []string{"w1"}
	cmd := &deleteCommand{
		canvasID: stringPtr("canvas-123"),
		yes:      boolPtr(true),
		draft:    boolPtr(false),
	}
	require.NoError(t, cmd.Execute(ctx))

	canvas, _ := captured["canvas"].(map[string]any)
	require.NotNil(t, canvas)
	spec, _ := canvas["spec"].(map[string]any)

	nodes, _ := spec["nodes"].([]any)
	require.Len(t, nodes, 0)

	edges, _ := spec["edges"].([]any)
	require.Len(t, edges, 1)
	first, _ := edges[0].(map[string]any)
	require.Equal(t, "x", first["sourceId"])
	require.Equal(t, "y", first["targetId"])
}
