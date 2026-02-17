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
