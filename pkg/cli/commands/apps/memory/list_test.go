package memory

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/cli"
)

const memoriesListResponse = `{
	"items": [
		{
			"id": "memory-001",
			"namespace": "deployments",
			"values": {
				"environment": "production",
				"version": "v1.2.3"
			},
			"source": "SOURCE_NODE",
			"createdAt": "2026-06-08T10:00:00Z",
			"updatedAt": "2026-06-08T10:15:00Z"
		},
		{
			"id": "memory-002",
			"namespace": "incidents",
			"values": {
				"severity": "critical",
				"ticket": 42
			},
			"source": "SOURCE_MANUAL",
			"createdAt": "2026-06-08T11:00:00Z",
			"updatedAt": "2026-06-08T11:15:00Z"
		}
	]
}`

const memoryCanvasID = "11111111-1111-1111-1111-111111111111"

func newMemoryListServer(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"canvases":[{"id":"` + memoryCanvasID + `","name":"my-app"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/"+memoryCanvasID+"/memory":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(memoriesListResponse))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestListCommandReturnsJSON(t *testing.T) {
	server := newMemoryListServer(t)
	ctx, stdout := cli.NewCommandContext(t, server, "json")
	ctx.Args = []string{"my-app"}

	err := (&listCommand{}).Execute(ctx)
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Len(t, result, 2)
	require.Equal(t, "memory-001", result[0]["id"])
	require.Equal(t, "deployments", result[0]["namespace"])
	require.Equal(t, "SOURCE_NODE", result[0]["source"])
	require.Equal(t, map[string]any{
		"environment": "production",
		"version":     "v1.2.3",
	}, result[0]["values"])
}

func TestListCommandReturnsTextOutput(t *testing.T) {
	server := newMemoryListServer(t)
	ctx, stdout := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{"my-app"}

	err := (&listCommand{}).Execute(ctx)
	require.NoError(t, err)

	raw := stdout.String()
	require.Contains(t, raw, "Namespace: deployments")
	require.Contains(t, raw, "Namespace: incidents")
	require.Contains(t, raw, "environment")
	require.Contains(t, raw, "version")
	require.Contains(t, raw, "production")
	require.Contains(t, raw, "v1.2.3")
	require.Contains(t, raw, "severity")
	require.Contains(t, raw, "ticket")
	require.Contains(t, raw, "critical")
	require.Contains(t, raw, "42")
	require.NotContains(t, raw, "memory-001")
	require.NotContains(t, raw, "SOURCE_NODE")
	require.NotContains(t, raw, "CREATED_AT")
}

func TestListCommandFiltersByNamespace(t *testing.T) {
	server := newMemoryListServer(t)
	ctx, stdout := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{"my-app"}
	namespace := "deployments"

	err := (&listCommand{namespace: &namespace}).Execute(ctx)
	require.NoError(t, err)

	raw := stdout.String()
	require.Contains(t, raw, "environment")
	require.Contains(t, raw, "version")
	require.Contains(t, raw, "production")
	require.NotContains(t, raw, "Namespace:")
	require.NotContains(t, raw, "severity")
	require.NotContains(t, raw, "critical")
}

func TestListCommandUsesActiveAppWhenArgumentIsOmitted(t *testing.T) {
	server := newMemoryListServer(t)
	ctx, stdout := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{ActiveApp: memoryCanvasID})

	err := (&listCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "production")
}

func TestListCommandPrintsEmptyTextMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/canvases/"+memoryCanvasID+"/memory", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{ActiveApp: memoryCanvasID})

	err := (&listCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, "No memory records found.\n", stdout.String())
}
