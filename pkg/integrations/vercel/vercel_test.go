package vercel

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Vercel_Sync(t *testing.T) {
	t.Run("missing access token -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{}}
		err := (&Vercel{}).Sync(core.SyncContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   integrationCtx,
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "accessToken is required")
		assert.NotEqual(t, "ready", integrationCtx.State)
	})

	t.Run("invalid token -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"accessToken": "bad-token"},
		}

		err := (&Vercel{}).Sync(core.SyncContext{
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusUnauthorized,
						Body:       io.NopCloser(strings.NewReader(`{"error":{"code":"NOT_AUTHORIZED","message":"Unauthorized"}}`)),
					},
				},
			},
			Integration:   integrationCtx,
			Configuration: map[string]any{"accessToken": "bad-token"},
		})

		require.ErrorContains(t, err, "failed to verify Vercel credentials")
		assert.NotEqual(t, "ready", integrationCtx.State)
	})

	t.Run("valid token -> ready and bearer header set", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"user":{"id":"usr-123","username":"acme","name":"Acme"}}`,
					)),
				},
			},
		}

		err := (&Vercel{}).Sync(core.SyncContext{
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				State:         "pending",
				Configuration: map[string]any{"accessToken": "vercel_token_123"},
			},
			Configuration: map[string]any{"accessToken": "vercel_token_123"},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)

		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "/v2/user", request.URL.Path)
		assert.Empty(t, request.URL.Query().Get("teamId"))
	})
}

func Test__Vercel_ListResources__Project(t *testing.T) {
	t.Run("returns projects from first page", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"projects":[{"id":"prj_1","name":"my-app"},{"id":"prj_2","name":"other"}],"pagination":{"count":2}}`,
					)),
				},
			},
		}

		resources, err := listProjects(core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"accessToken": "vercel_token_123"}},
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "project", resources[0].Type)
		assert.Equal(t, "prj_1", resources[0].ID)
		assert.Equal(t, "my-app", resources[0].Name)
	})
}
