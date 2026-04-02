package canvases

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func newCanvasDeleteServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"canvases":[{"metadata":{"id":"abc-123","name":"my-canvas"}}]}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/canvases/abc-123":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestDeleteCommandPrintsConfirmation(t *testing.T) {
	server := newCanvasDeleteServer(t)
	ctx, stdout := newCreateCommandContextForTest(t, server, "text")
	ctx.Args = []string{"my-canvas"}

	err := (&deleteCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Canvas deleted: my-canvas")
}

func TestDeleteCommandReturnsJSONOutput(t *testing.T) {
	server := newCanvasDeleteServer(t)
	ctx, stdout := newCreateCommandContextForTest(t, server, "json")
	ctx.Args = []string{"my-canvas"}

	err := (&deleteCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `"id": "abc-123"`)
	require.Contains(t, stdout.String(), `"deleted": "true"`)
}

func TestDeleteCommandFailsOnServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"canvases":[{"metadata":{"id":"abc-123","name":"my-canvas"}}]}`))
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":13,"message":"internal error"}`))
		}
	}))
	t.Cleanup(server.Close)

	ctx, _ := newCreateCommandContextForTest(t, server, "text")
	ctx.Args = []string{"my-canvas"}

	err := (&deleteCommand{}).Execute(ctx)
	require.Error(t, err)
}
