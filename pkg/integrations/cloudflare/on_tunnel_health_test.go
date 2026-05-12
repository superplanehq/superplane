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

	t.Run("skips when tunnel filter does not match", func(t *testing.T) {
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"new_status":"Down","tunnel_id":"tun123"}`),
			Headers: http.Header{"cf-webhook-auth": []string{"secret123"}},
			Webhook: &contexts.NodeWebhookContext{
				Secret: "secret123",
			},
			Events: events,
			Configuration: map[string]any{
				"tunnel":    "other",
				"newStatus": []string{"TUNNEL_STATUS_TYPE_DOWN"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Len(t, events.Payloads, 0)
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
