package firehydrant

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

func Test__FireHydrant__Sync(t *testing.T) {
	fh := &FireHydrant{}

	t.Run("no API key -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "",
			},
		}

		err := fh.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "API key is required")
	})

	t.Run("successful sync -> ready state and metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": [
							{
								"slug": "SEV0",
								"description": "Critical",
								"type": "severity"
							},
							{
								"slug": "SEV1",
								"description": "High",
								"type": "severity"
							}
						]
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		err := fh.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.firehydrant.io/v1/severities", httpContext.Requests[0].URL.String())

		metadata := integrationCtx.Metadata.(Metadata)
		assert.Len(t, metadata.Severities, 2)
		assert.Equal(t, "SEV0", metadata.Severities[0].Slug)
		assert.Equal(t, "SEV1", metadata.Severities[1].Slug)
	})

	t.Run("failed severity list -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error": "unauthorized"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "invalid-api-key",
			},
		}

		err := fh.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
	})

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}

		err := fh.Sync(core.SyncContext{
			Configuration: "invalid-config",
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "failed to decode config")
	})
}
