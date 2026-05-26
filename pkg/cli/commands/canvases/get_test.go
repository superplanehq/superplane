package canvases

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const testGetCanvasID = "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

func newCanvasGetServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/canvases/"+testGetCanvasID, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + testGetCanvasID + `","name":"my-canvas","organizationId":"org-uuid"},"spec":{"nodes":[],"edges":[]}}}`))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestGetCommandPrintsCanvasOnSuccess(t *testing.T) {
	server := newCanvasGetServer(t)
	ctx, stdout := newCreateCommandContextForTest(t, server, "text")
	ctx.Args = []string{testGetCanvasID}

	err := (&getCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "ID: "+testGetCanvasID)
	require.Contains(t, stdout.String(), "Name: my-canvas")
	require.Contains(t, stdout.String(), "Nodes: 0")
	require.Contains(t, stdout.String(), "Edges: 0")
	require.NotContains(t, stdout.String(), "Canvas URL:")
}

func TestGetCommandPrintsURLFromResponseOrgID(t *testing.T) {
	server := newCanvasGetServer(t)
	ctx, stdout := newCommandContextWithConfigForTest(t, server, "text", &fakeConfig{
		url: "https://app.superplane.com",
	})
	ctx.Args = []string{testGetCanvasID}

	err := (&getCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "ID: "+testGetCanvasID)
	require.Contains(t, stdout.String(), "Name: my-canvas")
	require.Contains(t, stdout.String(), "Canvas URL: https://app.superplane.com/org-uuid/canvases/"+testGetCanvasID)
	require.Contains(t, stdout.String(), "Nodes: 0")
	require.Contains(t, stdout.String(), "Edges: 0")
}

func TestGetCommandSkipsURLWhenResponseMissingOrgID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + testGetCanvasID + `","name":"my-canvas"},"spec":{"nodes":[],"edges":[]}}}`))
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newCommandContextWithConfigForTest(t, server, "text", &fakeConfig{
		url: "https://app.superplane.com",
	})
	ctx.Args = []string{testGetCanvasID}

	err := (&getCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "ID: "+testGetCanvasID)
	require.Contains(t, stdout.String(), "Name: my-canvas")
	require.NotContains(t, stdout.String(), "Canvas URL:")
}
