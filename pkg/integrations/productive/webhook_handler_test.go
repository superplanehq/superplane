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

func Test__WebhookHandler__CompareConfig(t *testing.T) {
	handler := &ProductiveWebhookHandler{}

	t.Run("same project and a superset of events match", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{ProjectID: "1", Events: []string{TaskCreatedEvent, TaskUpdatedEvent}},
			WebhookConfiguration{ProjectID: "1", Events: []string{TaskCreatedEvent}},
		)

		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("different projects do not match", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{ProjectID: "1", Events: []string{TaskCreatedEvent}},
			WebhookConfiguration{ProjectID: "2", Events: []string{TaskCreatedEvent}},
		)

		require.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("missing an event does not match", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{ProjectID: "1", Events: []string{TaskCreatedEvent}},
			WebhookConfiguration{ProjectID: "1", Events: []string{TaskCreatedEvent, TaskUpdatedEvent}},
		)

		require.NoError(t, err)
		assert.False(t, equal)
	})
}

func Test__WebhookHandler__Setup(t *testing.T) {
	handler := &ProductiveWebhookHandler{}

	t.Run("creates the webhook with SuperPlane's secret", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			jsonResponse(`{"data":{"id":"w1","type":"webhooks"}}`),
		}}

		metadata, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: testIntegration(nil),
			Webhook: &contexts.WebhookContext{
				URL:           "https://sp.test/hook",
				Secret:        []byte(testWebhookSecret),
				Configuration: WebhookConfiguration{ProjectID: "1", Events: []string{TaskCreatedEvent}},
			},
		})

		require.NoError(t, err)
		webhookMetadata, ok := metadata.(*WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, "w1", webhookMetadata.ID)

		body := requestBody(t, httpContext.Requests[0])
		data := body["data"].(map[string]any)
		attributes := data["attributes"].(map[string]any)
		assert.Equal(t, "https://sp.test/hook", attributes["target_url"])
		assert.Equal(t, testWebhookSecret, attributes["secret"])
	})

	t.Run("failure is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(`{}`))},
		}}

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: testIntegration(nil),
			Webhook: &contexts.WebhookContext{
				URL:           "https://sp.test/hook",
				Secret:        []byte(testWebhookSecret),
				Configuration: WebhookConfiguration{ProjectID: "1", Events: []string{TaskCreatedEvent}},
			},
		})

		require.Error(t, err)
	})
}

func Test__WebhookHandler__Cleanup(t *testing.T) {
	handler := &ProductiveWebhookHandler{}

	t.Run("deletes the webhook", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(`{}`))},
		}}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: testIntegration(nil),
			Webhook: &contexts.WebhookContext{
				Metadata:      WebhookMetadata{ID: "w1"},
				Configuration: WebhookConfiguration{ProjectID: "1", Events: []string{TaskCreatedEvent}},
			},
		})

		require.NoError(t, err)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/webhooks/w1")
	})

	// A failed Setup leaves the webhook record without a Productive.io id, so
	// there is nothing to delete remotely.
	t.Run("skips the API call when no webhook was ever created", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: testIntegration(nil),
			Webhook: &contexts.WebhookContext{
				Metadata:      WebhookMetadata{},
				Configuration: WebhookConfiguration{ProjectID: "1", Events: []string{TaskCreatedEvent}},
			},
		})

		require.NoError(t, err)
		assert.Empty(t, httpContext.Requests, "must not call Productive.io with an empty webhook id")
	})

	t.Run("delete failure is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{}`))},
		}}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: testIntegration(nil),
			Webhook: &contexts.WebhookContext{
				Metadata:      WebhookMetadata{ID: "w1"},
				Configuration: WebhookConfiguration{ProjectID: "1", Events: []string{TaskCreatedEvent}},
			},
		})

		require.Error(t, err)
	})
}
