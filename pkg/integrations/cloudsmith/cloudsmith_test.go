package cloudsmith

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

func Test__Cloudsmith__Sync(t *testing.T) {
	integration := &Cloudsmith{}

	t.Run("missing apiKey returns error", func(t *testing.T) {
		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{},
			Integration:   &contexts.IntegrationContext{},
		})

		require.ErrorContains(t, err, "apiKey is required")
	})

	t.Run("valid apiKey sets metadata and marks ready", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-key",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{
				"apiKey": "test-key",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"authenticated": true,
							"slug": "service-account",
							"name": "SuperPlane Service Account",
							"email": "service@acme.test"
						}`)),
					},
				},
			},
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)

		metadata, ok := integrationCtx.Metadata.(Metadata)
		require.True(t, ok)
		assert.Equal(t, "service-account", metadata.UserSlug)
		assert.Equal(t, "SuperPlane Service Account", metadata.UserName)
		assert.Equal(t, "service@acme.test", metadata.UserEmail)
	})

	t.Run("unauthenticated response returns error", func(t *testing.T) {
		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{
				"apiKey": "test-key",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"authenticated": false}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiKey": "test-key",
				},
			},
		})

		require.ErrorContains(t, err, "not valid")
	})

	t.Run("invalid apiKey (401) returns error", func(t *testing.T) {
		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{
				"apiKey": "bad-key",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusUnauthorized,
						Body:       io.NopCloser(strings.NewReader(`{"detail":"Invalid api key."}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiKey": "bad-key",
				},
			},
		})

		require.ErrorContains(t, err, "error validating API key")
	})
}
