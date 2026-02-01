package rootly

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIncidentResolved__HandleWebhook(t *testing.T) {
	trigger := &OnIncidentResolved{}

	signatureFor := func(secret string, timestamp string, body []byte) string {
		payload := append([]byte(timestamp), body...)
		sig := computeHMACSHA256([]byte(secret), payload)
		return "t=" + timestamp + ",v1=" + sig
	}

	t.Run("missing X-Rootly-Signature -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("event type not incident.resolved -> no emit", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-123","title":"Test Incident"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("severity filter does not match -> no emit", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.resolved","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-123","title":"Test Incident","severity":{"id":"sev-1","name":"SEV1"}}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"severityFilter": []string{"sev-2"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("valid signature, incident.resolved -> event emitted with incident payload", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.resolved","id":"evt-999","issued_at":"2026-01-19T12:45:00Z"},"data":{"id":"inc-999","sequential_id":42,"title":"Resolved Incident","slug":"resolved-incident","status":"resolved","resolution_message":"fixed","resolved_at":"2026-01-19T12:44:30Z","resolved_by":{"id":"usr-1","name":"Jane"},"url":"https://app.rootly.com/incidents/inc-999"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())

		payload := eventContext.Payloads[0]
		assert.Equal(t, "rootly.incident.resolved", payload.Type)

		data := payload.Data.(map[string]any)
		assert.Equal(t, "incident.resolved", data["event"])
		incident := data["incident"].(map[string]any)
		assert.Equal(t, "inc-999", incident["id"])
		assert.Equal(t, float64(42), incident["sequential_id"]) // JSON numbers unmarshal as float64
		assert.Equal(t, "Resolved Incident", incident["title"])
		assert.Equal(t, "resolved", incident["status"])
		assert.Equal(t, "fixed", incident["resolution_message"])
		assert.Equal(t, "2026-01-19T12:44:30Z", incident["resolved_at"])
		assert.Equal(t, "https://app.rootly.com/incidents/inc-999", incident["url"])
	})
}

func Test__OnIncidentResolved__Setup(t *testing.T) {
	trigger := &OnIncidentResolved{}

	integrationCtx := &contexts.IntegrationContext{}
	err := trigger.Setup(core.TriggerContext{
		Integration: integrationCtx,
	})

	require.NoError(t, err)
	require.Len(t, integrationCtx.WebhookRequests, 1)

	webhookConfig := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
	assert.Equal(t, []string{"incident.resolved"}, webhookConfig.Events)
}

