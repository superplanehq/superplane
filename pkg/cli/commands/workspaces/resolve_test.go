package workspaces

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cli "github.com/superplanehq/superplane/test/support/cli"
)

func TestResolveWorkspaceID(t *testing.T) {
	workspaceID := "11111111-1111-1111-1111-111111111111"

	t.Run("uses explicit workspace flag", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(
				`{"factories":[{"id":%q,"name":"shipping"}]}`,
				workspaceID,
			)))
		}))
		t.Cleanup(server.Close)

		ctx, _ := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{
			ActiveWorkspace: "ignored",
		})
		got, err := ResolveWorkspaceID(ctx, "shipping")
		require.NoError(t, err)
		assert.Equal(t, workspaceID, got)
	})

	t.Run("falls back to active workspace", func(t *testing.T) {
		ctx, _ := cli.NewCommandContextWithConfig(t, nil, "text", &cli.FakeConfig{
			ActiveWorkspace: workspaceID,
		})
		got, err := ResolveWorkspaceID(ctx, "")
		require.NoError(t, err)
		assert.Equal(t, workspaceID, got)
	})

	t.Run("errors when workspace missing", func(t *testing.T) {
		ctx, _ := cli.NewCommandContextWithConfig(t, nil, "text", &cli.FakeConfig{})
		_, err := ResolveWorkspaceID(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "superplane workspace active")
	})
}
