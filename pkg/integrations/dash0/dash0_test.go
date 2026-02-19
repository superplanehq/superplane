package dash0

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
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

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/prometheus/api/v1/query")
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
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://api.us-west-2.aws.dash0.com/api/prometheus",
			},
		}

		err := d.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		// Should not have double /api/prometheus
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/api/prometheus/api/v1/query")
		assert.NotContains(t, httpContext.Requests[0].URL.String(), "/api/prometheus/api/prometheus")
	})

	t.Run("webhook URL is stored in metadata with /api/v1 prefix", func(t *testing.T) {
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
			},
		}

		webhookID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
		integrationCtx := &syncIntegrationContext{
			IntegrationContext: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://api.us-west-2.aws.dash0.com",
				},
			},
			webhookID: &webhookID,
		}

		err := d.Sync(core.SyncContext{
			Configuration:   integrationCtx.IntegrationContext.Configuration,
			HTTP:            httpContext,
			Integration:     integrationCtx,
			WebhooksBaseURL: "https://superplane.example.com",
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.IntegrationContext.State)

		metadata, ok := integrationCtx.IntegrationContext.Metadata.(Metadata)
		require.True(t, ok)
		assert.Equal(t, "https://superplane.example.com/api/v1/webhooks/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", metadata.WebhookURL)
	})
}

// syncIntegrationContext wraps the test IntegrationContext to return a fixed webhook ID.
type syncIntegrationContext struct {
	*contexts.IntegrationContext
	webhookID *uuid.UUID
}

func (c *syncIntegrationContext) EnsureIntegrationWebhook(_ any) (*uuid.UUID, error) {
	return c.webhookID, nil
}
