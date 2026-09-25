package datadog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnErrorTrackingAlert__OnIntegrationMessage(t *testing.T) {
	trigger := &OnErrorTrackingAlert{}

	t.Run("emits triggered error tracking alerts", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionTriggered,
				"title":            "[Triggered] checkout new issues",
				"body":             "InventoryTimeout: checkout failed",
			},
			Events: events,
		})
		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
		assert.Equal(t, ErrorTrackingAlertPayloadType, events.Payloads[0].Type)
	})

	t.Run("ignores recovered alerts", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": "Recovered",
			},
			Events: events,
		})
		require.NoError(t, err)
		assert.Empty(t, events.Payloads)
	})

	t.Run("ignores non error tracking events", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"event_type":       "query_alert_monitor",
				"alert_transition": AlertTransitionTriggered,
			},
			Events: events,
		})
		require.NoError(t, err)
		assert.Empty(t, events.Payloads)
	})
}

func Test__OnErrorTrackingAlert__ExampleDataMatchesTrigger(t *testing.T) {
	trigger := &OnErrorTrackingAlert{}
	example := trigger.ExampleData()

	events := &contexts.EventContext{}
	err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
		Message: example,
		Events:  events,
	})
	require.NoError(t, err)
	require.Len(t, events.Payloads, 1)
}
