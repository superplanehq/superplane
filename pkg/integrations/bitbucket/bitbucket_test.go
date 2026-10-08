package bitbucket

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Bitbucket__Sync(t *testing.T) {
	b := &Bitbucket{}

	t.Run("workspace is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"token": "token",
			},
		}

		err := b.Sync(core.SyncContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: integrationCtx,
			Configuration: map[string]any{
				"authType": AuthTypeWorkspaceAccessToken,
			},
		})

		require.ErrorContains(t, err, "workspace is required")
	})

	t.Run("authType is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"token": "token",
			},
		}

		err := b.Sync(core.SyncContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: integrationCtx,
			Configuration: map[string]any{
				"workspace": "superplane",
			},
		})

		require.ErrorContains(t, err, "authType is required")
	})

	t.Run("unsupported authType returns error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"token": "token",
			},
		}

		err := b.Sync(core.SyncContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: integrationCtx,
			Configuration: map[string]any{
				"workspace": "superplane",
				"authType":  "unsupported",
			},
		})

		require.ErrorContains(t, err, "authType unsupported is not supported")
	})

	t.Run("workspace metadata is set and integration is ready", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"uuid":"{workspace-uuid}","name":"SuperPlane","slug":"superplane"}`)),
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"token": "token",
			},
		}

		err := b.Sync(core.SyncContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Configuration: map[string]any{
				"workspace": "superplane",
				"authType":  AuthTypeWorkspaceAccessToken,
			},
		})
		require.NoError(t, err)

		require.NotNil(t, integrationCtx.Metadata)
		metadata, ok := integrationCtx.Metadata.(Metadata)
		require.True(t, ok)
		assert.Equal(t, AuthTypeWorkspaceAccessToken, metadata.AuthType)
		require.NotNil(t, metadata.Workspace)
		assert.Equal(t, "{workspace-uuid}", metadata.Workspace.UUID)
		assert.Equal(t, "SuperPlane", metadata.Workspace.Name)
		assert.Equal(t, "superplane", metadata.Workspace.Slug)
		assert.Equal(t, "ready", integrationCtx.State)
	})
}

func Test__Bitbucket__SyncRepositoryAccessToken(t *testing.T) {
	b := &Bitbucket{}

	t.Run("repository is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "repo-token"},
		}
		err := b.Sync(core.SyncContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: integrationCtx,
			Configuration: map[string]any{
				"authType": AuthTypeRepositoryAccessToken,
			},
		})
		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("validates configured repository without workspace access", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"uuid":"{repo-uuid}","name":"my-repo","slug":"my-repo","full_name":"superplane/my-repo","mainbranch":{"name":"main"}}`)),
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "repo-token"},
		}
		err := b.Sync(core.SyncContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Configuration: map[string]any{
				"authType":   AuthTypeRepositoryAccessToken,
				"repository": "superplane/my-repo",
			},
		})
		require.NoError(t, err)
		metadata, ok := integrationCtx.Metadata.(Metadata)
		require.True(t, ok)
		assert.Equal(t, AuthTypeRepositoryAccessToken, metadata.AuthType)
		require.NotNil(t, metadata.Repository)
		assert.Equal(t, "superplane/my-repo", metadata.Repository.FullName)
		assert.Equal(t, "superplane", metadata.Workspace.Slug)
		assert.Equal(t, "ready", integrationCtx.State)
	})

	t.Run("expired token fails validation", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error": {"message": "Token is expired"}}`)),
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "expired-token"},
		}
		err := b.Sync(core.SyncContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Configuration: map[string]any{
				"authType":   AuthTypeRepositoryAccessToken,
				"repository": "superplane/my-repo",
			},
		})
		require.ErrorContains(t, err, "error getting repository")
		assert.Equal(t, "", integrationCtx.State)
	})

	t.Run("wrong repository is confined", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				AuthType:  AuthTypeRepositoryAccessToken,
				Workspace: &WorkspaceMetadata{Slug: "superplane"},
				Repository: &RepositoryMetadata{
					UUID:     "{repo-uuid}",
					FullName: "superplane/my-repo",
					Slug:     "my-repo",
					Name:     "my-repo",
				},
			},
		}
		err := requireRepositoryInWorkspace(integrationCtx, "superplane/other-repo")
		require.ErrorContains(t, err, "not accessible")
	})

	t.Run("resource lookup returns only the configured repository", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"uuid":"{repo-uuid}","name":"my-repo","slug":"my-repo","full_name":"superplane/my-repo","mainbranch":{"name":"main"}}`)),
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "repo-token"},
			Metadata: Metadata{
				AuthType:  AuthTypeRepositoryAccessToken,
				Workspace: &WorkspaceMetadata{Slug: "superplane"},
				Repository: &RepositoryMetadata{
					UUID:     "{repo-uuid}",
					FullName: "superplane/my-repo",
					Slug:     "my-repo",
					Name:     "my-repo",
				},
			},
		}
		resources, err := b.ListResources("repository", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
		})
		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "superplane/my-repo", resources[0].Name)
	})
}
