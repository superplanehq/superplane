package coolify

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

func validIntegrationConfig() map[string]any {
	return map[string]any{
		"baseUrl":  "https://coolify.example.com",
		"apiToken": "coolify_test_token",
	}
}

func Test__Coolify__Sync(t *testing.T) {
	integration := &Coolify{}

	t.Run("missing baseUrl -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "t"},
		}
		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          &contexts.HTTPContext{},
			Integration:   integrationCtx,
		})
		require.ErrorContains(t, err, "baseUrl is required")
	})

	t.Run("missing apiToken -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"baseUrl": "https://coolify.example.com"},
		}
		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          &contexts.HTTPContext{},
			Integration:   integrationCtx,
		})
		require.ErrorContains(t, err, "apiToken is required")
	})

	t.Run("invalid baseUrl -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "coolify.example.com",
				"apiToken": "t",
			},
		}
		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          &contexts.HTTPContext{},
			Integration:   integrationCtx,
		})
		require.ErrorContains(t, err, "invalid baseUrl")
	})

	t.Run("valid credentials -> ready", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`"4.0.0"`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: validIntegrationConfig(),
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpCtx,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://coolify.example.com/api/v1/version", httpCtx.Requests[0].URL.String())
		assert.Equal(t, "Bearer coolify_test_token", httpCtx.Requests[0].Header.Get("Authorization"))
	})

	t.Run("API rejects credentials -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"message":"Unauthenticated."}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: validIntegrationConfig(),
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpCtx,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "failed to verify Coolify credentials")
		require.ErrorContains(t, err, "Unauthenticated.")
	})

	t.Run("trailing slash on baseUrl is normalized", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`"4.0.0"`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://coolify.example.com/",
				"apiToken": "coolify_test_token",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpCtx,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://coolify.example.com/api/v1/version", httpCtx.Requests[0].URL.String())
	})

	t.Run("baseUrl with /api/v1 suffix is normalized", func(t *testing.T) {
		for _, baseURL := range []string{
			"https://coolify.example.com/api/v1",
			"https://coolify.example.com/api/v1/",
		} {
			httpCtx := &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`"4.0.0"`)),
					},
				},
			}

			integrationCtx := &contexts.IntegrationContext{
				Configuration: map[string]any{
					"baseUrl":  baseURL,
					"apiToken": "coolify_test_token",
				},
			}

			err := integration.Sync(core.SyncContext{
				Configuration: integrationCtx.Configuration,
				HTTP:          httpCtx,
				Integration:   integrationCtx,
			})

			require.NoError(t, err)
			require.Len(t, httpCtx.Requests, 1)
			assert.Equal(t, "https://coolify.example.com/api/v1/version", httpCtx.Requests[0].URL.String(), "baseUrl %q should not duplicate the API path prefix", baseURL)
		}
	})
}

func Test__Coolify__ListResources(t *testing.T) {
	integration := &Coolify{}
	integrationCtx := &contexts.IntegrationContext{Configuration: validIntegrationConfig()}

	t.Run("application", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"uuid":"abc123","name":"frontend"},{"uuid":"def456","name":"api"}]`,
					)),
				},
			},
		}

		resources, err := integration.ListResources("application", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "application", resources[0].Type)
		assert.Equal(t, "frontend", resources[0].Name)
		assert.Equal(t, "abc123", resources[0].ID)
		assert.Equal(t, "api", resources[1].Name)
		assert.Equal(t, "def456", resources[1].ID)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://coolify.example.com/api/v1/applications", httpCtx.Requests[0].URL.String())
	})

	t.Run("service", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"uuid":"svc1","name":"postgres"},{"uuid":"svc2","name":"plausible"}]`,
					)),
				},
			},
		}

		resources, err := integration.ListResources("service", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "service", resources[0].Type)
		assert.Equal(t, "postgres", resources[0].Name)
		assert.Equal(t, "svc1", resources[0].ID)
	})

	t.Run("application without name uses uuid", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"uuid":"abc123"}]`)),
				},
			},
		}

		resources, err := integration.ListResources("application", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "abc123", resources[0].Name)
		assert.Equal(t, "abc123", resources[0].ID)
	})

	t.Run("application without uuid is skipped", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"uuid":"abc123","name":"frontend"},{"name":"orphan"}]`,
					)),
				},
			},
		}

		resources, err := integration.ListResources("application", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "abc123", resources[0].ID)
	})

	t.Run("unknown resource type -> empty", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}

		resources, err := integration.ListResources("unknown", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
		assert.Empty(t, httpCtx.Requests, "no API call should be made for unknown resource type")
	})
}
