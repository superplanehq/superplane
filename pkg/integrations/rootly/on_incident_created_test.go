package rootly

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIncidentCreated__HandleWebhook(t *testing.T) {
	trigger := &OnIncidentCreated{}

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

	t.Run("invalid signature format -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Rootly-Signature", "invalid")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-123","title":"Test Incident"}}`)
		headers := http.Header{}
		headers.Set("X-Rootly-Signature", "t=1234567890,v1=invalid")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
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
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("event type not incident.created -> no emit", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.resolved","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-123","title":"Test Incident"}}`)
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

	t.Run("valid signature and incident.created -> event is emitted", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-123","title":"Test Incident","status":"started"}}`)
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
		assert.Equal(t, "rootly.incident.created", payload.Type)
		assert.Equal(t, "incident.created", payload.Data.(map[string]any)["event"])
		assert.Equal(t, "evt-123", payload.Data.(map[string]any)["event_id"])
		assert.NotNil(t, payload.Data.(map[string]any)["incident"])
	})

	t.Run("severity filter no match -> no emit", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-123","title":"Test","attributes":{"severity":{"slug":"sev1"}}}}`)
		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor("test-secret", "1234567890", body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"severityFilter": []string{"sev2"}},
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("severity filter match -> event emitted", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-123","title":"Test","attributes":{"severity":{"slug":"sev2"}}}}`)
		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor("test-secret", "1234567890", body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"severityFilter": []string{"sev2"}},
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})
}

func Test__OnIncidentCreated__Setup(t *testing.T) {
	trigger := &OnIncidentCreated{}

	t.Run("webhook request with incident.created", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Configuration: map[string]any{},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)

		webhookConfig := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, []string{"incident.created"}, webhookConfig.Events)
	})
}
