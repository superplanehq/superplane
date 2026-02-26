package launchdarkly

import (
	"encoding/json"
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

	t.Run("same project and URL -> true", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{ProjectKey: "default", WebhooksBaseURL: "https://example.com"},
			WebhookConfiguration{ProjectKey: "default", WebhooksBaseURL: "https://example.com"},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("different project -> false", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{ProjectKey: "default", WebhooksBaseURL: "https://example.com"},
			WebhookConfiguration{ProjectKey: "other", WebhooksBaseURL: "https://example.com"},
		)
		require.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("different URL -> false", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{ProjectKey: "default", WebhooksBaseURL: "https://old.ngrok-free.app"},
			WebhookConfiguration{ProjectKey: "default", WebhooksBaseURL: "https://new.ngrok-free.app"},
		)
		require.NoError(t, err)
		assert.False(t, equal)
	})
}

func Test__LaunchDarklyWebhookHandler__Merge(t *testing.T) {
	handler := &LaunchDarklyWebhookHandler{}

	t.Run("always returns current unchanged", func(t *testing.T) {
		current := WebhookConfiguration{ProjectKey: "default"}
		merged, changed, err := handler.Merge(current, WebhookConfiguration{ProjectKey: "default"})
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
			url:           "https://example.com/api/v1/webhooks/w1",
			configuration: WebhookConfiguration{ProjectKey: "default", WebhooksBaseURL: "https://example.com"},
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

		// Verify statement scopes to the selected project (all flags)
		bodyBytes, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(bodyBytes, &body))
		statements, ok := body["statements"].([]any)
		require.True(t, ok)
		require.Len(t, statements, 1)
		stmt := statements[0].(map[string]any)
		resources := stmt["resources"].([]any)
		assert.Equal(t, "proj/default:env/*:flag/*", resources[0])

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
