package dash0

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

func Test__Dash0__Sync(t *testing.T) {
	d := &Dash0{}

	t.Run("no apiToken -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "apiToken is required")
	})

	t.Run("no baseURL -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "baseURL is required")
	})

	t.Run("successful connection test -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": []
							}
						}
					`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"kind": "Dash0NotificationChannel",
						"metadata": {
							"name": "SuperPlane (8f5fbc57-2738-409a-a6f8-af65c2de733c)",
							"labels": { "dash0.com/id": "channel-abc" }
						},
						"spec": { "type": "webhook", "config": { "url": "https://hooks.example.com/webhook" } }
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   integrationCtx.Configuration,
			HTTP:            httpContext,
			Integration:     integrationCtx,
			WebhooksBaseURL: "https://hooks.example.com",
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		metadata, ok := integrationCtx.Metadata.(Metadata)
		require.True(t, ok)
		assert.Equal(t, "channel-abc", metadata.NotificationChannelID)
		require.Len(t, httpContext.Requests, 3)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/prometheus/api/v1/query")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/api/notification-channels")
		assert.Equal(t, http.MethodGet, httpContext.Requests[1].Method)
		assert.Equal(t, http.MethodPost, httpContext.Requests[2].Method)
		assert.Equal(t, "Bearer token123", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("connection test failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error validating connection")
		assert.NotEqual(t, "ready", integrationCtx.State)
	})

	t.Run("baseURL with /api/prometheus suffix -> strips suffix", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"status": "success",
							"data": {
								"resultType": "vector",
								"result": []
							}
						}
					`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"kind": "Dash0NotificationChannel",
						"metadata": {
							"name": "SuperPlane",
							"labels": { "dash0.com/id": "channel-xyz" }
						},
						"spec": { "type": "webhook", "config": { "url": "https://example.com" } }
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com/api/prometheus",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration:   integrationCtx.Configuration,
			HTTP:            httpContext,
			Integration:     integrationCtx,
			WebhooksBaseURL: "https://hooks.example.com",
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 3)
		// Should not have double /api/prometheus
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/prometheus/api/v1/query")
		assert.NotContains(t, httpContext.Requests[0].URL.String(), "/api/prometheus/api/prometheus")
	})
}

func Test__Dash0__Cleanup(t *testing.T) {
	d := &Dash0{}

	t.Run("deletes notification channel from metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
			Metadata: Metadata{
				NotificationChannelID: "channel-to-delete",
			},
		}

		err := d.Cleanup(core.IntegrationCleanupContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/notification-channels/channel-to-delete")
	})

	t.Run("no channel id -> no-op", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com",
			},
		}

		err := d.Cleanup(core.IntegrationCleanupContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, httpContext.Requests)
	})
}
