package launchdarkly

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

// testHandlerWebhookContext implements core.WebhookContext for webhook handler tests.
type testHandlerWebhookContext struct {
	id            string
	url           string
	secret        []byte
	metadata      any
	configuration any
}

func (w *testHandlerWebhookContext) GetID() string              { return w.id }
func (w *testHandlerWebhookContext) GetURL() string             { return w.url }
func (w *testHandlerWebhookContext) GetSecret() ([]byte, error) { return w.secret, nil }
func (w *testHandlerWebhookContext) GetMetadata() any           { return w.metadata }
func (w *testHandlerWebhookContext) GetConfiguration() any      { return w.configuration }
func (w *testHandlerWebhookContext) SetSecret(secret []byte) error {
	w.secret = secret
	return nil
}

func Test__LaunchDarklyWebhookHandler__CompareConfig(t *testing.T) {
	handler := &LaunchDarklyWebhookHandler{}

	t.Run("identical events", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{KindFlag}},
			WebhookConfiguration{Events: []string{KindFlag}},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("A superset of B", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{KindFlag, "project"}},
			WebhookConfiguration{Events: []string{KindFlag}},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("A subset of B -> true", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{KindFlag}},
			WebhookConfiguration{Events: []string{KindFlag, "project"}},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("different events -> false", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{Events: []string{"project"}},
			WebhookConfiguration{Events: []string{KindFlag}},
		)
		require.NoError(t, err)
		assert.False(t, equal)
	})
}

func Test__LaunchDarklyWebhookHandler__Merge(t *testing.T) {
	handler := &LaunchDarklyWebhookHandler{}

	t.Run("Merge adds events when requested is superset of current", func(t *testing.T) {
		merged, changed, err := handler.Merge(
			WebhookConfiguration{Events: []string{KindFlag}},
			WebhookConfiguration{Events: []string{KindFlag, "project"}},
		)
		require.NoError(t, err)
		assert.True(t, changed)
		assert.Equal(t, WebhookConfiguration{Events: []string{KindFlag, "project"}}, merged)
	})

	t.Run("Merge returns current when no change", func(t *testing.T) {
		current := WebhookConfiguration{Events: []string{KindFlag}}
		merged, changed, err := handler.Merge(
			current,
			WebhookConfiguration{Events: []string{KindFlag}},
		)
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, current, merged)
	})
}

func Test__LaunchDarklyWebhookHandler__Setup(t *testing.T) {
	handler := &LaunchDarklyWebhookHandler{}

	createWebhookResponse := `{"_id":"ld-webhook-abc123","url":"https://example.com/api/v1/webhooks/w1","secret":"auto-generated-secret","on":true,"sign":true}`

	t.Run("creates webhook in LaunchDarkly and stores secret", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(createWebhookResponse)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-api-key"},
		}

		webhookCtx := &testHandlerWebhookContext{
			url: "https://example.com/api/v1/webhooks/w1",
		}

		result, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
			Webhook:     webhookCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://app.launchdarkly.com/api/v2/webhooks", req.URL.String())
		assert.Equal(t, "auto-generated-secret", string(webhookCtx.secret))

		metadata, ok := result.(WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, "ld-webhook-abc123", metadata.LDWebhookID)
	})
}

func Test__LaunchDarklyWebhookHandler__Cleanup(t *testing.T) {
	handler := &LaunchDarklyWebhookHandler{}

	t.Run("deletes webhook from LaunchDarkly", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-api-key"},
		}

		webhookCtx := &testHandlerWebhookContext{
			metadata: WebhookMetadata{LDWebhookID: "ld-webhook-abc123"},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
			Webhook:     webhookCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodDelete, req.Method)
		assert.Equal(t, "https://app.launchdarkly.com/api/v2/webhooks/ld-webhook-abc123", req.URL.String())
	})

	t.Run("no-op when LDWebhookID is empty", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		webhookCtx := &testHandlerWebhookContext{
			metadata: WebhookMetadata{LDWebhookID: ""},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:    httpContext,
			Webhook: webhookCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("404 from LaunchDarkly is ignored", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message":"webhook not found"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-api-key"},
		}

		webhookCtx := &testHandlerWebhookContext{
			metadata: WebhookMetadata{LDWebhookID: "ld-webhook-gone"},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
			Webhook:     webhookCtx,
		})

		require.NoError(t, err)
	})
}
