package posthog

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

func Test__PostHogWebhookHandler__CompareConfig(t *testing.T) {
	handler := &PostHogWebhookHandler{}

	t.Run("same project and events -> true", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{ProjectID: "1", Events: []string{"signup"}, FilterTestAccounts: true},
			WebhookConfiguration{ProjectID: "1", Events: []string{"signup"}, FilterTestAccounts: true},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("event order does not split a reusable webhook", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{ProjectID: "1", Events: []string{"signup", "purchase"}},
			WebhookConfiguration{ProjectID: "1", Events: []string{"purchase", "signup"}},
		)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("different project -> false", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{ProjectID: "1"},
			WebhookConfiguration{ProjectID: "2"},
		)
		require.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("different events -> false", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{ProjectID: "1", Events: []string{"signup"}},
			WebhookConfiguration{ProjectID: "1", Events: []string{"purchase"}},
		)
		require.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("all events differs from a filtered subset", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{ProjectID: "1", Events: []string{}},
			WebhookConfiguration{ProjectID: "1", Events: []string{"signup"}},
		)
		require.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("different test account filter -> false", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{ProjectID: "1", FilterTestAccounts: true},
			WebhookConfiguration{ProjectID: "1", FilterTestAccounts: false},
		)
		require.NoError(t, err)
		assert.False(t, equal)
	})
}

func Test__PostHogWebhookHandler__Merge(t *testing.T) {
	handler := &PostHogWebhookHandler{}

	t.Run("always returns current unchanged", func(t *testing.T) {
		current := WebhookConfiguration{ProjectID: "1"}
		merged, changed, err := handler.Merge(current, WebhookConfiguration{ProjectID: "2"})
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, current, merged)
	})
}

func Test__PostHogWebhookHandler__Setup(t *testing.T) {
	handler := &PostHogWebhookHandler{}

	createResponse := `{"id":"01912f3a-aaaa-bbbb-cccc-1234567890ab","name":"SuperPlane","enabled":true}`

	newHTTPContext := func() *contexts.HTTPContext {
		return &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(createResponse)),
				},
			},
		}
	}

	integrationContext := func() *contexts.IntegrationContext {
		return &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "phx_test", "host": "https://eu.posthog.com"},
		}
	}

	t.Run("creates destination, stores secret, returns metadata", func(t *testing.T) {
		httpContext := newHTTPContext()
		webhookContext := &contexts.WebhookContext{
			URL: "https://superplane.example.com/api/v1/webhooks/w1",
			Configuration: WebhookConfiguration{
				ProjectID:          "12345",
				Events:             []string{"signup", "purchase"},
				FilterTestAccounts: true,
			},
		}

		result, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integrationContext(),
			Webhook:     webhookContext,
		})
		require.NoError(t, err)

		metadata, ok := result.(WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, "01912f3a-aaaa-bbbb-cccc-1234567890ab", metadata.HogFunctionID)
		assert.Equal(t, "12345", metadata.ProjectID)

		//
		// The secret must be stored, since it is the only thing authenticating
		// deliveries from PostHog.
		//
		require.NotEmpty(t, webhookContext.Secret)

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "https://eu.posthog.com/api/projects/12345/hog_functions/", request.URL.String())

		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)

		sent := CreateHogFunctionRequest{}
		require.NoError(t, json.Unmarshal(body, &sent))

		assert.Equal(t, HogFunctionTypeDestination, sent.Type)
		assert.Equal(t, WebhookTemplateID, sent.TemplateID)
		assert.True(t, sent.Enabled)
		assert.Equal(t, "https://superplane.example.com/api/v1/webhooks/w1", sent.Inputs["url"].Value)
		assert.Equal(t, http.MethodPost, sent.Inputs["method"].Value)

		headers, ok := sent.Inputs["headers"].Value.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, string(webhookContext.Secret), headers[WebhookTokenHeader])

		assert.True(t, sent.Filters.FilterTestAccounts)
		require.Len(t, sent.Filters.Events, 2)
		assert.Equal(t, HogFunctionEventFilter{ID: "signup", Type: "events"}, sent.Filters.Events[0])
		assert.Equal(t, HogFunctionEventFilter{ID: "purchase", Type: "events"}, sent.Filters.Events[1])
	})

	t.Run("no configured events sends an empty filter so every event matches", func(t *testing.T) {
		httpContext := newHTTPContext()

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integrationContext(),
			Webhook: &contexts.WebhookContext{
				URL:           "https://superplane.example.com/api/v1/webhooks/w1",
				Configuration: WebhookConfiguration{ProjectID: "12345"},
			},
		})
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 1)
		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		sent := CreateHogFunctionRequest{}
		require.NoError(t, json.Unmarshal(body, &sent))
		assert.Empty(t, sent.Filters.Events)
	})

	t.Run("missing project ID returns error before API call", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integrationContext(),
			Webhook: &contexts.WebhookContext{
				URL:           "https://superplane.example.com/api/v1/webhooks/w1",
				Configuration: WebhookConfiguration{},
			},
		})

		require.ErrorContains(t, err, "project ID is required")
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("PostHog error is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"detail":"missing hog_function:write scope"}`)),
				},
			},
		}

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integrationContext(),
			Webhook: &contexts.WebhookContext{
				URL:           "https://superplane.example.com/api/v1/webhooks/w1",
				Configuration: WebhookConfiguration{ProjectID: "12345"},
			},
		})

		require.ErrorContains(t, err, "failed to create webhook destination in PostHog")
	})
}

func Test__PostHogWebhookHandler__Cleanup(t *testing.T) {
	handler := &PostHogWebhookHandler{}

	integrationContext := func() *contexts.IntegrationContext {
		return &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "phx_test", "host": "https://eu.posthog.com"},
		}
	}

	t.Run("deletes the destination", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integrationContext(),
			Webhook: &contexts.WebhookContext{
				Metadata: WebhookMetadata{ProjectID: "12345", HogFunctionID: "fn-1"},
			},
		})
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, http.MethodDelete, request.Method)
		assert.Equal(t, "https://eu.posthog.com/api/projects/12345/hog_functions/fn-1/", request.URL.String())
	})

	t.Run("already deleted in PostHog is not an error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"detail":"Not found."}`)),
				},
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integrationContext(),
			Webhook: &contexts.WebhookContext{
				Metadata: WebhookMetadata{ProjectID: "12345", HogFunctionID: "fn-1"},
			},
		})

		require.NoError(t, err)
	})

	t.Run("nothing provisioned means nothing to clean up", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integrationContext(),
			Webhook:     &contexts.WebhookContext{Metadata: WebhookMetadata{}},
		})

		require.NoError(t, err)
		assert.Empty(t, httpContext.Requests)
	})
}
