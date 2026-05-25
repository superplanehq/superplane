package console

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClearReplacesPanelsAndLayoutWithEmptyArrays(t *testing.T) {
	var captured map[string]any
	server := newAPITestServer(t, requestExpectation{
		method: http.MethodPut,
		path:   "/api/v1/canvases/canvas-123/dashboard",
		handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(body, &captured))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"dashboard": {"panels": [], "layout": []}}`))
		},
	})

	ctx, _ := newCommandContext(t, server.server, "text")
	cmd := &clearCommand{canvasID: stringPtr("canvas-123"), yes: boolPtr(true)}

	require.NoError(t, cmd.Execute(ctx))
	panels, _ := captured["panels"].([]any)
	layout, _ := captured["layout"].([]any)
	require.Len(t, panels, 0)
	require.Len(t, layout, 0)
}
