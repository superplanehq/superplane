package dash0

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnAlertEvent__Setup(t *testing.T) {
	trigger := OnAlertEvent{}

	t.Run("requires at least one event type", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"eventTypes": []string{},
			},
			Integration: &contexts.IntegrationContext{},
		})

		require.ErrorContains(t, err, "at least one event type must be selected")
	})

	t.Run("requests webhook with configured event types", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"eventTypes": []string{"fired", "resolved"},
			},
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)

		configuration, ok := integrationCtx.WebhookRequests[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, []string{"fired", "resolved"}, configuration["eventTypes"])
	})
}

func Test__OnAlertEvent__HandleWebhook(t *testing.T) {
	trigger := OnAlertEvent{}

	t.Run("emits event for matching fired payload", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{
				"event_type":"fired",
				"check":{"id":"check-123","name":"Checkout latency","severity":"critical","labels":{"service":"checkout"}},
				"summary":"Latency above threshold",
				"description":"p95 latency above threshold",
				"timestamp":"2026-02-09T12:00:00Z"
			}`),
			Configuration: map[string]any{
				"eventTypes": []string{"fired"},
			},
			Events: eventCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		require.Len(t, eventCtx.Payloads, 1)
		assert.Equal(t, OnAlertEventPayloadType, eventCtx.Payloads[0].Type)

		payload, ok := eventCtx.Payloads[0].Data.(AlertEventPayload)
		require.True(t, ok)
		assert.Equal(t, "fired", payload.EventType)
		assert.Equal(t, "check-123", payload.CheckID)
		assert.Equal(t, "Checkout latency", payload.CheckName)
		assert.Equal(t, "critical", payload.Severity)
		assert.Equal(t, "Latency above threshold", payload.Summary)
	})

	t.Run("ignores event when type is filtered out", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{"event_type":"resolved","check":{"id":"check-123"}}`),
			Configuration: map[string]any{
				"eventTypes": []string{"fired"},
			},
			Events: eventCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, 0, eventCtx.Count())
	})

	t.Run("missing check id returns error", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		status, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{"event_type":"fired","check":{"name":"missing id"}}`),
			Configuration: map[string]any{
				"eventTypes": []string{"fired"},
			},
			Events: eventCtx,
		})

		require.ErrorContains(t, err, "check id is required")
		assert.Equal(t, http.StatusBadRequest, status)
		assert.Equal(t, 0, eventCtx.Count())
	})
}
