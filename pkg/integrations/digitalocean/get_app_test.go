package digitalocean

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetApp__Setup(t *testing.T) {
	component := &GetApp{}

	t.Run("missing app returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "app ID is required")
	})

	t.Run("empty app returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"app": "",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "app ID is required")
	})

	t.Run("expression app is accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"app": "{{ $.trigger.data.appID }}",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})

	t.Run("valid app -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"app": "6d8abe1c-7cc5-4db3-b1aa-d9cdd1c127e7",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"app": {
								"id": "6d8abe1c-7cc5-4db3-b1aa-d9cdd1c127e7",
								"spec": {
									"name": "test-app"
								}
							}
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "test-token",
				},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})
}

func Test__GetApp__Execute(t *testing.T) {
	component := &GetApp{}

	t.Run("successful fetch -> emits app data", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"app": {
							"id": "6d8abe1c-7cc5-4db3-b1aa-d9cdd1c127e7",
							"spec": {
								"name": "my-app",
								"region": "nyc"
							},
							"default_ingress": "https://my-app-b6v8c.ondigitalocean.app",
							"live_url": "https://my-app-b6v8c.ondigitalocean.app",
							"created_at": "2026-03-20T10:00:00Z",
							"updated_at": "2026-03-20T10:00:00Z",
							"region": {
								"slug": "nyc",
								"label": "New York",
								"continent": "North America",
								"flag": "us",
								"data_centers": ["nyc1", "nyc3"]
							},
							"active_deployment": {
								"id": "deployment-123",
								"phase": "ACTIVE",
								"created_at": "2026-03-20T10:00:00Z"
							}
						}
					}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"app": "6d8abe1c-7cc5-4db3-b1aa-d9cdd1c127e7",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "test-token",
				},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)

		// Verify emission
		require.True(t, executionState.Passed)
		require.Equal(t, "default", executionState.Channel)
		require.Equal(t, "digitalocean.app.fetched", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		// Verify request
		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		require.Equal(t, http.MethodGet, req.Method)
		require.Equal(t, "https://api.digitalocean.com/v2/apps/6d8abe1c-7cc5-4db3-b1aa-d9cdd1c127e7", req.URL.String())
		require.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
	})

	t.Run("non-existent app -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body: io.NopCloser(strings.NewReader(`{
						"id": "not_found",
						"message": "The resource you requested could not be found."
					}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"app": "non-existent-app-id",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "test-token",
				},
			},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get app")
		require.False(t, executionState.Passed)
	})

	t.Run("invalid API token -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body: io.NopCloser(strings.NewReader(`{
						"id": "unauthorized",
						"message": "Unable to authenticate you."
					}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"app": "6d8abe1c-7cc5-4db3-b1aa-d9cdd1c127e7",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "invalid-token",
				},
			},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get app")
		require.False(t, executionState.Passed)
	})
}

func Test__GetApp__Name(t *testing.T) {
	component := &GetApp{}
	require.Equal(t, "digitalocean.getApp", component.Name())
}
