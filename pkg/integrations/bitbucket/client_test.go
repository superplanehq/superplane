package bitbucket

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Client__PathEscaping(t *testing.T) {
	t.Run("GetWorkspace path escapes workspace slug", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"uuid":"{workspace-uuid}","name":"My Workspace","slug":"my workspace"}`)),
				},
			},
		}

		client := &Client{
			AuthType: AuthTypeWorkspaceAccessToken,
			Token:    "token",
			HTTP:     httpCtx,
		}

		workspace, err := client.GetWorkspace("my workspace")
		require.NoError(t, err)
		assert.Equal(t, "My Workspace", workspace.Name)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://api.bitbucket.org/2.0/workspaces/my%20workspace", httpCtx.Requests[0].URL.String())
	})

	t.Run("ListRepositories path escapes workspace", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"values":[{"uuid":"{repo-uuid}","name":"repo","full_name":"my workspace/repo","slug":"repo"}]}`)),
				},
			},
		}

		client := &Client{
			AuthType: AuthTypeWorkspaceAccessToken,
			Token:    "token",
			HTTP:     httpCtx,
		}

		repos, err := client.ListRepositories("my workspace")
		require.NoError(t, err)
		require.Len(t, repos, 1)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://api.bitbucket.org/2.0/repositories/my%20workspace?pagelen=100", httpCtx.Requests[0].URL.String())
	})

	t.Run("CreateWebhook path escapes workspace and repo slug", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"uuid":"{hook-uuid}","url":"https://example.com","active":true}`)),
				},
			},
		}

		client := &Client{
			AuthType: AuthTypeWorkspaceAccessToken,
			Token:    "token",
			HTTP:     httpCtx,
		}

		hook, err := client.CreateWebhook("my workspace", "my repo", "https://example.com", "secret", []string{"repo:push"})
		require.NoError(t, err)
		assert.Equal(t, "{hook-uuid}", hook.UUID)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://api.bitbucket.org/2.0/repositories/my%20workspace/my%20repo/hooks", httpCtx.Requests[0].URL.String())
	})

	t.Run("DeleteWebhook path escapes workspace, repo slug, and webhook UID", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		client := &Client{
			AuthType: AuthTypeWorkspaceAccessToken,
			Token:    "token",
			HTTP:     httpCtx,
		}

		err := client.DeleteWebhook("my workspace", "my repo", "{hook uuid}")
		require.NoError(t, err)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://api.bitbucket.org/2.0/repositories/my%20workspace/my%20repo/hooks/%7Bhook%20uuid%7D", httpCtx.Requests[0].URL.String())
	})
}
