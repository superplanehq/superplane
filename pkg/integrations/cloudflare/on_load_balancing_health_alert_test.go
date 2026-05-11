package cloudflare

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnLoadBalancingHealthAlert__Setup(t *testing.T) {
	trigger := &OnLoadBalancingHealthAlert{}
	integration := &contexts.IntegrationContext{}

	err := trigger.Setup(core.TriggerContext{
		Configuration: map[string]any{
			"pool":        "pool123",
			"newHealth":   []string{"Unhealthy"},
			"eventSource": []string{"pool"},
		},
		Integration: integration,
	})

	require.NoError(t, err)
	require.Len(t, integration.WebhookRequests, 1)
	assert.Equal(t, OnLoadBalancingHealthAlertSpec{
		Pool:        "pool123",
		NewHealth:   []string{"Unhealthy"},
		EventSource: []string{"pool"},
	}, integration.WebhookRequests[0])
}

func Test__OnLoadBalancingHealthAlert__HandleWebhook(t *testing.T) {
	trigger := &OnLoadBalancingHealthAlert{}

	t.Run("rejects missing auth header", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
			Webhook: &contexts.NodeWebhookContext{
				Secret: "secret123",
			},
		})

		require.ErrorContains(t, err, "missing cf-webhook-auth")
		assert.Equal(t, http.StatusUnauthorized, code)
	})

	t.Run("emits valid alert payload", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"new_health":"Unhealthy","event_source":"origin","pool_id":"pool123"}`),
			Headers: http.Header{"cf-webhook-auth": []string{"secret123"}},
			Webhook: &contexts.NodeWebhookContext{
				Secret: "secret123",
			},
			Events: events,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, events.Payloads, 1)
		assert.Equal(t, LoadBalancingHealthAlertPayloadType, events.Payloads[0].Type)
	})

	t.Run("skips emit when pool filter does not match payload", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"new_health":"Unhealthy","event_source":"origin","pool_id":"pool123"}`),
			Headers: http.Header{"cf-webhook-auth": []string{"secret123"}},
			Webhook: &contexts.NodeWebhookContext{
				Secret: "secret123",
			},
			Events: events,
			Configuration: map[string]any{
				"pool":        "other-pool",
				"newHealth":   []string{"Unhealthy"},
				"eventSource": []string{"origin"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, events.Payloads)
	})

	t.Run("skips emit when new_health not allowed by trigger", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"new_health":"Healthy","event_source":"origin","pool_id":"pool123"}`),
			Headers: http.Header{"cf-webhook-auth": []string{"secret123"}},
			Webhook: &contexts.NodeWebhookContext{
				Secret: "secret123",
			},
			Events: events,
			Configuration: map[string]any{
				"newHealth":   []string{"Unhealthy"},
				"eventSource": []string{"origin"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, events.Payloads)
	})

	t.Run("matches nested data envelope", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{
				"data": {
					"new_health": "Unhealthy",
					"event_source": "pool",
					"pool_id": "pool123"
				}
			}`),
			Headers: http.Header{"cf-webhook-auth": []string{"secret123"}},
			Webhook: &contexts.NodeWebhookContext{
				Secret: "secret123",
			},
			Events: events,
			Configuration: map[string]any{
				"pool":        "pool123",
				"newHealth":   []string{"Unhealthy"},
				"eventSource": []string{"pool"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, events.Payloads, 1)
	})
}
