package cloudflare

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

func Test__OnTunnelHealth__Setup(t *testing.T) {
	trigger := &OnTunnelHealth{}
	integration := &contexts.IntegrationContext{}

	err := trigger.Setup(core.TriggerContext{
		Configuration: map[string]any{
			"tunnel":    "tun123",
			"newStatus": []string{"Down"},
		},
		Integration: integration,
	})

	require.NoError(t, err)
	require.Len(t, integration.WebhookRequests, 1)
	assert.Equal(t, OnTunnelHealthSpec{
		Tunnel:    "tun123",
		NewStatus: []string{"TUNNEL_STATUS_TYPE_DOWN"},
	}, integration.WebhookRequests[0])
}

func Test__OnTunnelHealth__HandleWebhook(t *testing.T) {
	trigger := &OnTunnelHealth{}

	t.Run("emits valid tunnel health payload", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"new_status":"Down","tunnel_id":"tun123"}`),
			Headers: http.Header{"cf-webhook-auth": []string{"secret123"}},
			Webhook: &contexts.NodeWebhookContext{
				Secret: "secret123",
			},
			Events: events,
			Configuration: map[string]any{
				"tunnel":    "tun123",
				"newStatus": []string{"TUNNEL_STATUS_TYPE_DOWN", "TUNNEL_STATUS_TYPE_DEGRADED"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, events.Payloads, 1)
		assert.Equal(t, TunnelHealthEventPayloadType, events.Payloads[0].Type)
	})

	t.Run("skips emit when tunnel_id does not match spec tunnel", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{
				"alert_type": "tunnel_health_event",
				"data": {
					"new_status": "TUNNEL_STATUS_TYPE_DEGRADED",
					"tunnel_id": "wrong-id",
					"tunnel_name": "tunnel-name"
				}
			}`),
			Headers: http.Header{"cf-webhook-auth": []string{"secret123"}},
			Webhook: &contexts.NodeWebhookContext{
				Secret: "secret123",
			},
			Events: events,
			Configuration: map[string]any{
				"tunnel":    "60d33a5d-1228-47e0-813d-a5297cc0624b",
				"newStatus": []string{"TUNNEL_STATUS_TYPE_DEGRADED"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Len(t, events.Payloads, 0)
	})

	t.Run("matches tunnel_id case-insensitively", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body: []byte(`{
				"data": {
					"new_status": "TUNNEL_STATUS_TYPE_DEGRADED",
					"tunnel_id": "60D33A5D-1228-47E0-813D-A5297CC0624B"
				}
			}`),
			Headers: http.Header{"cf-webhook-auth": []string{"secret123"}},
			Webhook: &contexts.NodeWebhookContext{
				Secret: "secret123",
			},
			Events: events,
			Configuration: map[string]any{
				"tunnel":    "60d33a5d-1228-47e0-813d-a5297cc0624b",
				"newStatus": []string{"TUNNEL_STATUS_TYPE_DEGRADED"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, events.Payloads, 1)
	})

	t.Run("new_status with suffix still matches", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"new_status":"TUNNEL_STATUS_TYPE_DEGRADED (status change)","tunnel_id":"t1"}`),
			Headers: http.Header{"cf-webhook-auth": []string{"secret123"}},
			Webhook: &contexts.NodeWebhookContext{
				Secret: "secret123",
			},
			Events: events,
			Configuration: map[string]any{
				"newStatus": []string{"TUNNEL_STATUS_TYPE_DEGRADED"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Len(t, events.Payloads, 1)
	})
}

func Test__OnTunnelHealth__SetupResolvesTunnelMetadata(t *testing.T) {
	trigger := &OnTunnelHealth{}
	metadata := &contexts.MetadataContext{}
	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiToken": "token123"},
		Metadata:      Metadata{AccountID: "acc123"},
	}

	err := trigger.Setup(core.TriggerContext{
		Configuration: map[string]any{
			"tunnel": "tun123",
		},
		HTTP: &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": {"id": "tun123", "name": "Edge tunnel"}
					}`)),
				},
			},
		},
		Integration: integration,
		Metadata:    metadata,
	})

	require.NoError(t, err)
	tm, ok := metadata.Metadata.(TunnelNodeMetadata)
	require.True(t, ok)
	assert.Equal(t, "Edge tunnel", tm.TunnelName)
}
