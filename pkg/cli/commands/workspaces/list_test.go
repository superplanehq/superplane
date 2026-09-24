package workspaces

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	cli "github.com/superplanehq/superplane/test/support/cli"
)

const listFactoriesPayload = `{
  "factories": [
    {
      "id": "11111111-1111-1111-1111-111111111111",
      "name": "SuperPlane",
      "key": "SUPER",
      "description": "Primary workspace"
    },
    {
      "id": "22222222-2222-2222-2222-222222222222",
      "name": "Shipping",
      "key": "SHIP"
    }
  ]
}`

func newListFactoriesServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/factories", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(listFactoriesPayload))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestListCommandRendersText(t *testing.T) {
	ctx, stdout := cli.NewCommandContextWithConfig(t, newListFactoriesServer(t), "text", &cli.FakeConfig{
		ActiveWorkspace: "11111111-1111-1111-1111-111111111111",
	})

	require.NoError(t, (&listCommand{}).Execute(ctx))

	out := stdout.String()
	require.Contains(t, out, "KEY")
	require.Contains(t, out, "NAME")
	require.Contains(t, out, "ID")
	require.Contains(t, out, "SUPER")
	require.Contains(t, out, "SuperPlane")
	require.Contains(t, out, "SHIP")
	require.Contains(t, out, "*")
}

func TestListCommandRendersJSON(t *testing.T) {
	ctx, stdout := cli.NewCommandContext(t, newListFactoriesServer(t), "json")

	require.NoError(t, (&listCommand{}).Execute(ctx))

	out := stdout.String()
	require.Contains(t, out, `"key": "SUPER"`)
	require.Contains(t, out, `"name": "Shipping"`)
}

func TestListCommandShowsEmptyState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/factories", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"factories":[]}`))
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	require.NoError(t, (&listCommand{}).Execute(ctx))
	require.Contains(t, stdout.String(), "No workspaces found.")
}
