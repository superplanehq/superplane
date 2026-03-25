package logfire

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

func TestLogfireWebhookHandler_CompareConfig(t *testing.T) {
	t.Parallel()

	handler := &LogfireWebhookHandler{}

	same, err := handler.CompareConfig(
		map[string]any{"eventType": " Alert.Received ", "resource": " Alerts ", "projectId": "proj_1", "alertId": "alt_1"},
		map[string]any{"eventType": "alert.received", "resource": "alerts", "projectId": "proj_1", "alertId": "alt_1"},
	)
	require.NoError(t, err)
	assert.True(t, same)

	same, err = handler.CompareConfig(
		map[string]any{"eventType": "alert.received", "resource": "alerts", "projectId": "proj_1", "alertId": "alt_1"},
		map[string]any{"eventType": "alert.resolved", "resource": "alerts", "projectId": "proj_1", "alertId": "alt_1"},
	)
	require.NoError(t, err)
	assert.False(t, same)
}

func TestLogfireWebhookHandler_Setup_Success(t *testing.T) {
	t.Parallel()

	handler := &LogfireWebhookHandler{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "lf_api_us_123"},
	}
	webhookCtx := &contexts.WebhookContext{
		ID:     "node_123",
		URL:    "https://example.com/webhook",
		Secret: []byte("secret-123"),
		Configuration: map[string]any{
			"eventType": onAlertReceivedEventType,
			"resource":  onAlertReceivedResource,
			"projectId": "proj_123",
			"alertId":   "alt_456",
		},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP: &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"channel_ids":[]}`)),
				},
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"id":"channel_123","label":"superplane-webhook-node_123"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"channel_ids":[]}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		},
		Integration: integrationCtx,
		Webhook:     webhookCtx,
	})
	require.NoError(t, err)

	result, ok := metadata.(LogfireWebhookMetadata)
	require.True(t, ok)
	assert.True(t, result.ManagedChannel)
	assert.True(t, result.SupportsWebhookSetup)
	assert.Equal(t, "channel_123", result.ChannelID)
}

func TestLogfireWebhookHandler_Setup_UnsupportedProvisioning(t *testing.T) {
	t.Parallel()

	handler := &LogfireWebhookHandler{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "lf_api_us_123"},
	}

	metadata, err := handler.Setup(core.WebhookHandlerContext{
		HTTP: &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"channel_ids":[]}`)),
				},
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error":"not found"}`)),
				},
			},
		},
		Integration: integrationCtx,
		Webhook: &contexts.WebhookContext{
			ID:     "node_123",
			URL:    "https://example.com/webhook",
			Secret: []byte("secret-123"),
			Configuration: map[string]any{
				"eventType": onAlertReceivedEventType,
				"resource":  onAlertReceivedResource,
				"projectId": "proj_123",
				"alertId":   "alt_456",
			},
		},
	})
	require.NoError(t, err)

	result, ok := metadata.(LogfireWebhookMetadata)
	require.True(t, ok)
	assert.False(t, result.ManagedChannel)
	assert.False(t, result.SupportsWebhookSetup)

	integrationMetadata, ok := integrationCtx.Metadata.(Metadata)
	require.True(t, ok)
	assert.False(t, integrationMetadata.SupportsWebhookSetup)
	assert.True(t, integrationMetadata.SupportsQueryAPI)
}

func TestLogfireWebhookHandler_Cleanup(t *testing.T) {
	t.Parallel()

	handler := &LogfireWebhookHandler{}

	t.Run("no managed channel is no-op", func(t *testing.T) {
		t.Parallel()

		err := handler.Cleanup(core.WebhookHandlerContext{
			Webhook: &contexts.WebhookContext{
				Metadata: LogfireWebhookMetadata{ManagedChannel: false},
			},
		})
		require.NoError(t, err)
	})

	t.Run("managed channel deletes alert channel", func(t *testing.T) {
		t.Parallel()

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "lf_api_us_123"},
			},
			Webhook: &contexts.WebhookContext{
				Metadata: LogfireWebhookMetadata{
					ManagedChannel: true,
					ChannelID:      "channel_123",
					ChannelsPath:   "/api/v1/channels/",
				},
			},
		})
		require.NoError(t, err)
	})
}
