package productive

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

func Test__ProductiveWebhookHandler__CompareConfig(t *testing.T) {
	handler := &ProductiveWebhookHandler{}

	t.Run("same project matches", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{ProjectID: "1"},
			WebhookConfiguration{ProjectID: "1"},
		)

		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("different projects do not match", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{ProjectID: "1"},
			WebhookConfiguration{ProjectID: "2"},
		)

		require.NoError(t, err)
		assert.False(t, equal)
	})
}

func Test__ProductiveWebhookHandler__Setup(t *testing.T) {
	handler := &ProductiveWebhookHandler{}

	t.Run("creates the webhook with SuperPlane's secret", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"id":"555","type":"webhooks"}}`),
			},
		}

		metadata, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: authorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				URL:           "https://sp.test/hook",
				Secret:        []byte("s3cr3t"),
				Configuration: WebhookConfiguration{ProjectID: "1"},
			},
		})

		require.NoError(t, err)
		webhookMetadata, ok := metadata.(*WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, "555", webhookMetadata.ID)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	})

	t.Run("webhooks_limit_exceeded is surfaced as a plan limitation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body: io.NopCloser(strings.NewReader(
						`{"errors":[{"status":"403","code":"webhooks_limit_exceeded","title":"Webhooks are not available on your plan"}]}`,
					)),
				},
			},
		}

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: authorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				URL:           "https://sp.test/hook",
				Secret:        []byte("s3cr3t"),
				Configuration: WebhookConfiguration{ProjectID: "1"},
			},
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrWebhooksLimitExceeded)
		assert.Contains(t, err.Error(), "does not offer webhooks on this plan")
	})
}

func Test__ProductiveWebhookHandler__Cleanup(t *testing.T) {
	handler := &ProductiveWebhookHandler{}

	t.Run("deletes the webhook", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{jsonResponse(`{}`)},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: authorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				Metadata:      WebhookMetadata{ID: "555"},
				Configuration: WebhookConfiguration{ProjectID: "1"},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	})

	// A failed Setup (e.g. webhooks_limit_exceeded) leaves the webhook record
	// without a Productive.io id, so there is nothing to delete remotely.
	t.Run("skips the API call when no webhook was ever created", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: authorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				Metadata:      WebhookMetadata{},
				Configuration: WebhookConfiguration{ProjectID: "1"},
			},
		})

		require.NoError(t, err)
		assert.Empty(t, httpContext.Requests, "must not call Productive.io with an empty webhook id")
	})

	t.Run("delete failure is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"errors":[{"title":"Not found"}]}`)),
				},
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: authorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				Metadata:      WebhookMetadata{ID: "555"},
				Configuration: WebhookConfiguration{ProjectID: "1"},
			},
		})

		require.Error(t, err)
	})
}
