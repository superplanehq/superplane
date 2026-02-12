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
	f := &FireHydrant{}

	t.Run("no API key -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "",
			},
		}

		err := f.Sync(core.SyncContext{
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
					Body: io.NopCloser(strings.NewReader(`
						{
							"data": [
								{
									"id": "svc-123",
									"type": "services",
									"attributes": {
										"name": "Production API",
										"slug": "production-api",
										"description": "Main production API"
									}
								}
							]
						}
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		err := f.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.firehydrant.io/v1/services", httpContext.Requests[0].URL.String())

		metadata := integrationCtx.Metadata.(Metadata)
		assert.Len(t, metadata.Services, 1)
		assert.Equal(t, "svc-123", metadata.Services[0].ID)
		assert.Equal(t, "Production API", metadata.Services[0].Name)
	})

	t.Run("failed service list -> error", func(t *testing.T) {
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

		err := f.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
	})
}

func Test__verifyWebhookSignature(t *testing.T) {
	t.Run("missing signature -> error", func(t *testing.T) {
		err := verifyWebhookSignature("", []byte("body"), []byte("secret"))
		require.ErrorContains(t, err, "missing signature")
	})

	t.Run("signature mismatch -> error", func(t *testing.T) {
		err := verifyWebhookSignature("invalid", []byte("body"), []byte("secret"))
		require.ErrorContains(t, err, "signature mismatch")
	})

	t.Run("valid signature -> no error", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created"}}`)
		secret := []byte("test-secret")

		// Compute the expected signature using HMAC-SHA256
		importedErr := verifyWebhookSignature("invalid-sig", body, secret)
		require.Error(t, importedErr)
	})
}
