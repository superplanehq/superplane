package rootly

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnEvent__HandleWebhook(t *testing.T) {
	trigger := &OnEvent{}

	signatureFor := func(secret string, timestamp string, body []byte) string {
		payload := append([]byte(timestamp), body...)
		sig := computeHMACSHA256([]byte(secret), payload)
		return "t=" + timestamp + ",v1=" + sig
	}

	baseConfig := map[string]any{}

	payloadBody := []byte(`{
  "event": {"type": "incident.updated", "id": "evt-123", "issued_at": "2026-01-01T02:00:00Z"},
  "data": {
    "id": "inc-1",
    "title": "API latency spike",
    "status": "started",
    "severity": "sev2",
    "services": [{"id": "svc-1", "name": "API", "slug": "api"}],
    "groups": [{"id": "team-1", "name": "Core"}],
    "events": [
      {"id": "ev-1", "event": "First note", "kind": "note", "source": "web", "visibility": "internal", "occurred_at": "2026-01-01T01:00:00Z", "created_at": "2026-01-01T01:00:00Z"},
      {"id": "ev-2", "event": "Second note", "kind": "note", "source": "api", "visibility": "external", "occurred_at": "2026-01-01T02:00:00Z", "created_at": "2026-01-01T02:00:00Z"}
    ]
  }
}`)

	t.Run("missing X-Rootly-Signature -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Configuration: baseConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("invalid JSON body -> 400", func(t *testing.T) {
		body := []byte("invalid json")
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: baseConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("incident status filter -> no emit", func(t *testing.T) {
		secret := "test-secret"
		timestamp := "1234567890"
		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, payloadBody))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    payloadBody,
			Headers: headers,
			Configuration: map[string]any{
				"incidentStatuses": []string{"resolved"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("event filters select matching event", func(t *testing.T) {
		secret := "test-secret"
		timestamp := "1234567890"
		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, payloadBody))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    payloadBody,
			Headers: headers,
			Configuration: map[string]any{
				"eventKinds":   []string{"note"},
				"eventSources": []string{"api"},
				"visibilities": []string{"external"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())

		payload := eventContext.Payloads[0]
		assert.Equal(t, "rootly.onEvent", payload.Type)
		data := payload.Data.(map[string]any)
		assert.Equal(t, "ev-2", data["id"])
		assert.Equal(t, "Second note", data["event"])
		assert.Equal(t, "note", data["kind"])
	})
}

func Test__OnEvent__Setup(t *testing.T) {
	trigger := &OnEvent{}

	t.Run("requests webhook with incident events", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)

		webhookConfig := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, onEventWebhookEvents, webhookConfig.Events)
	})
}
