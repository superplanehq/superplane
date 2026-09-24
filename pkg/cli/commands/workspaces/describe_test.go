package workspaces

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	cli "github.com/superplanehq/superplane/test/support/cli"
)

const describeFactoryPayload = `{
  "factory": {
    "id": "11111111-1111-1111-1111-111111111111",
    "name": "SuperPlane",
    "key": "SUPER",
    "description": "Primary workspace",
    "lines": [
      {"id": "line-1", "name": "build", "steps": [{"type": "app"}, {"type": "app"}]},
      {"id": "line-2", "name": "review", "steps": [{"type": "app"}]}
    ]
  }
}`

func newDescribeFactoryServer(t *testing.T, wantFactoryPath string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, wantFactoryPath, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(describeFactoryPayload))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestDescribeCommandUsesPositionalKey(t *testing.T) {
	ctx, stdout := cli.NewCommandContext(t, newDescribeFactoryServer(t, "/api/v1/factories/super"), "text")
	ctx.Args = []string{"super"}

	require.NoError(t, (&describeCommand{}).Execute(ctx))

	out := stdout.String()
	require.Contains(t, out, "SUPER")
	require.Contains(t, out, "SuperPlane")
	require.Contains(t, out, "11111111-1111-1111-1111-111111111111")
	require.NotContains(t, out, "Description:")
	require.NotContains(t, out, "Primary workspace")
	require.Contains(t, out, "build, review")
	require.NotContains(t, out, "step")
	require.NotContains(t, out, "Tasks")
}

func TestDescribeCommandUsesWorkspaceFlag(t *testing.T) {
	workspace := "ship"
	ctx, stdout := cli.NewCommandContextWithConfig(
		t,
		newDescribeFactoryServer(t, "/api/v1/factories/ship"),
		"text",
		&cli.FakeConfig{ActiveWorkspace: "11111111-1111-1111-1111-111111111111"},
	)

	require.NoError(t, (&describeCommand{workspace: &workspace}).Execute(ctx))
	require.Contains(t, stdout.String(), "SUPER")
}

func TestDescribeCommandUsesActiveWorkspace(t *testing.T) {
	workspaceID := "11111111-1111-1111-1111-111111111111"
	ctx, stdout := cli.NewCommandContextWithConfig(
		t,
		newDescribeFactoryServer(t, "/api/v1/factories/"+workspaceID),
		"text",
		&cli.FakeConfig{ActiveWorkspace: workspaceID},
	)

	require.NoError(t, (&describeCommand{}).Execute(ctx))
	require.Contains(t, stdout.String(), "SUPER")
}

func TestDescribeCommandRendersJSON(t *testing.T) {
	ctx, stdout := cli.NewCommandContext(t, newDescribeFactoryServer(t, "/api/v1/factories/super"), "json")
	ctx.Args = []string{"super"}

	require.NoError(t, (&describeCommand{}).Execute(ctx))
	require.Contains(t, stdout.String(), `"key": "SUPER"`)
	require.Contains(t, stdout.String(), `"name": "build"`)
}

func TestDescribeCommandRequiresWorkspace(t *testing.T) {
	ctx, _ := cli.NewCommandContextWithConfig(t, nil, "text", &cli.FakeConfig{})
	err := (&describeCommand{}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "superplane workspace active")
}

func TestDescribeCommandRejectsFlagAndArgument(t *testing.T) {
	workspace := "ship"
	ctx, _ := cli.NewCommandContextWithConfig(t, nil, "text", &cli.FakeConfig{})
	ctx.Args = []string{"super"}
	err := (&describeCommand{workspace: &workspace}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not both")
}
