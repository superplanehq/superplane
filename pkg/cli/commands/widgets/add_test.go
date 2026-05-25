package widgets

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// canvasResponseWith returns a canvas response that has change management
// disabled — used to seed the GET /canvases/<id> calls that mutation
// commands make to check change management state and to read the spec.
func canvasResponseWith(spec string) string {
	return `{"canvas": {"spec": {` + spec + `, "changeManagement": {"enabled": false}}}}`
}

func TestAddPublishesAnnotationWidgetNode(t *testing.T) {
	var captured map[string]any
	server := newAPITestServer(t,
		// changeManagementEnabled lookup
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/canvas-123",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(canvasResponseWith(`"nodes": [], "edges": []`)))
			},
		},
		// list versions to find existing draft
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/canvas-123/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions": [{"metadata": {"id": "ver-1", "state": "STATE_DRAFT"}}]}`))
			},
		},
		// describe draft version (returns spec used as base)
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/canvas-123/versions/ver-1",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version": {"metadata":{"id":"ver-1"}, "spec": {"nodes": [], "edges": []}}}`))
			},
		},
		// update version
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
		// publish the draft (default: not in --draft mode)
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
	cmd := &addCommand{
		canvasID:      stringPtr("canvas-123"),
		component:     stringPtr("annotation"),
		name:          stringPtr("note-1"),
		configuration: stringPtr(""),
		positionX:     int32Ptr(0),
		positionY:     int32Ptr(0),
		width:         int32Ptr(0),
		height:        int32Ptr(0),
		color:         stringPtr(""),
		text:          stringPtr("Hello!"),
		draft:         boolPtr(false),
	}

	require.NoError(t, cmd.Execute(ctx))

	canvas, _ := captured["canvas"].(map[string]any)
	require.NotNil(t, canvas)
	spec, _ := canvas["spec"].(map[string]any)
	require.NotNil(t, spec)
	nodes, _ := spec["nodes"].([]any)
	require.Len(t, nodes, 1)

	first, _ := nodes[0].(map[string]any)
	require.Equal(t, "TYPE_WIDGET", first["type"])
	require.Equal(t, "annotation", first["component"])
	require.Equal(t, "note-1", first["name"])

	cfg, _ := first["configuration"].(map[string]any)
	require.Equal(t, "Hello!", cfg["text"])
}

func TestAddRejectsWhenChangeManagementEnabledWithoutDraft(t *testing.T) {
	server := newAPITestServer(t, requestExpectation{
		method: http.MethodGet,
		path:   "/api/v1/canvases/canvas-123",
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"canvas": {"spec": {"nodes": [], "changeManagement": {"enabled": true}}}}`))
		},
	})

	ctx, _ := newCommandContext(t, server.server, "text")
	cmd := &addCommand{
		canvasID:      stringPtr("canvas-123"),
		component:     stringPtr("annotation"),
		name:          stringPtr(""),
		configuration: stringPtr(""),
		positionX:     int32Ptr(0),
		positionY:     int32Ptr(0),
		width:         int32Ptr(0),
		height:        int32Ptr(0),
		color:         stringPtr(""),
		text:          stringPtr(""),
		draft:         boolPtr(false),
	}
	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "change management is enabled")
}
