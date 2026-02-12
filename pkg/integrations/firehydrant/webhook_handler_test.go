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

// TestWebhookContext implements core.WebhookContext for testing
type TestWebhookContext struct {
	ID            string
	URL           string
	Secret        string
	Configuration any
	Metadata      any
}

func (w *TestWebhookContext) GetID() string {
	return w.ID
}

func (w *TestWebhookContext) GetURL() string {
	return w.URL
}

func (w *TestWebhookContext) GetSecret() ([]byte, error) {
	return []byte(w.Secret), nil
}

func (w *TestWebhookContext) GetMetadata() any {
	return w.Metadata
}

func (w *TestWebhookContext) GetConfiguration() any {
	return w.Configuration
}

func (w *TestWebhookContext) SetSecret(secret []byte) error {
	w.Secret = string(secret)
	return nil
}

func Test__FireHydrantWebhookHandler__CompareConfig(t *testing.T) {
	handler := &FireHydrantWebhookHandler{}

	t.Run("same events -> match", func(t *testing.T) {
		match, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{"incident.created"}},
			WebhookConfiguration{Events: []string{"incident.created"}},
		)
		require.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("superset of events -> match", func(t *testing.T) {
		match, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{"incident.created", "incident.resolved"}},
			WebhookConfiguration{Events: []string{"incident.created"}},
		)
		require.NoError(t, err)
		assert.True(t, match)
	})

	t.Run("missing events -> no match", func(t *testing.T) {
		match, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{"incident.created"}},
			WebhookConfiguration{Events: []string{"incident.resolved"}},
		)
		require.NoError(t, err)
		assert.False(t, match)
	})
}

func Test__FireHydrantWebhookHandler__Setup(t *testing.T) {
	handler := &FireHydrantWebhookHandler{}

	t.Run("successful setup", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`
						{
							"data": {
								"id": "webhook-123",
								"type": "webhook_endpoint",
								"attributes": {
									"url": "https://example.com/webhook",
									"secret": "webhook-secret",
									"active": true
								}
							}
						}
					`)),
				},
			},
		}

		webhookCtx := &TestWebhookContext{
			URL:    "https://example.com/webhook",
			Secret: "",
			Configuration: WebhookConfiguration{
				Events: []string{"incident.created"},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
			Secrets: make(map[string]core.IntegrationSecret),
		}

		metadata, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Webhook:     webhookCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)

		webhookMetadata := metadata.(WebhookMetadata)
		assert.Equal(t, "webhook-123", webhookMetadata.EndpointID)
		assert.Equal(t, "webhook-secret", webhookCtx.Secret)
	})
}

func Test__FireHydrantWebhookHandler__Cleanup(t *testing.T) {
	handler := &FireHydrantWebhookHandler{}

	t.Run("successful cleanup", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		webhookCtx := &TestWebhookContext{
			Metadata: WebhookMetadata{
				EndpointID: "webhook-123",
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
			Secrets: make(map[string]core.IntegrationSecret),
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Webhook:     webhookCtx,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.firehydrant.io/v1/webhooks/endpoints/webhook-123", httpContext.Requests[0].URL.String())
	})
}
