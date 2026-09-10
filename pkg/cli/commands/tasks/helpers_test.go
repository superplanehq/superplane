package tasks

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cli "github.com/superplanehq/superplane/test/support/cli"
)

const testResolveWorkspaceID = "11111111-1111-1111-1111-111111111111"

func TestResolveTaskID_UUID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")
	id := "22222222-2222-2222-2222-222222222222"

	resolved, err := resolveTaskID(ctx, testResolveWorkspaceID, id)
	require.NoError(t, err)
	assert.Equal(t, id, resolved)
}

func TestResolveTaskID_Slug(t *testing.T) {
	const listPayload = `{"orders":[{"id":"22222222-2222-2222-2222-222222222222","key":"SP-42","title":"Task One"}]}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/factories/"+testResolveWorkspaceID+"/orders" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(listPayload))
			return
		}
		t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")
	resolved, err := resolveTaskID(ctx, testResolveWorkspaceID, "SP-42")
	require.NoError(t, err)
	assert.Equal(t, "22222222-2222-2222-2222-222222222222", resolved)
}

func TestResolveTaskID_SlugNotFound(t *testing.T) {
	const listPayload = `{"orders":[{"id":"22222222-2222-2222-2222-222222222222","key":"SP-42","title":"Task One"}]}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/factories/"+testResolveWorkspaceID+"/orders" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(listPayload))
			return
		}
		t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")
	_, err := resolveTaskID(ctx, testResolveWorkspaceID, "SP-99")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `task "SP-99" not found`)
}
