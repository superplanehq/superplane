package workspaces

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cli "github.com/superplanehq/superplane/test/support/cli"
)

func TestFindWorkspaceID(t *testing.T) {
	workspaceID := "11111111-1111-1111-1111-111111111111"
	const describePayload = `{"factory":{"id":"11111111-1111-1111-1111-111111111111","name":"SuperPlane","key":"SUPER"}}`

	newDescribeServer := func(t *testing.T, wantPath string) *httptest.Server {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && r.URL.Path == wantPath {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(describePayload))
				return
			}
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}))
		t.Cleanup(server.Close)
		return server
	}

	t.Run("returns UUID unchanged", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}))
		t.Cleanup(server.Close)

		ctx, _ := cli.NewCommandContext(t, server, "text")
		got, err := FindWorkspaceID(ctx, workspaceID)
		require.NoError(t, err)
		assert.Equal(t, workspaceID, got)
	})

	t.Run("resolves workspace key through the API", func(t *testing.T) {
		ctx, _ := cli.NewCommandContext(t, newDescribeServer(t, "/api/v1/factories/super"), "text")
		got, err := FindWorkspaceID(ctx, "super")
		require.NoError(t, err)
		assert.Equal(t, workspaceID, got)
	})
}

func TestResolveWorkspaceID(t *testing.T) {
	workspaceID := "11111111-1111-1111-1111-111111111111"

	t.Run("uses explicit workspace flag", func(t *testing.T) {
		ctx, _ := cli.NewCommandContextWithConfig(t, nil, "text", &cli.FakeConfig{
			ActiveWorkspace: "ignored",
		})
		got, err := ResolveWorkspaceID(ctx, "super")
		require.NoError(t, err)
		assert.Equal(t, "super", got)
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
