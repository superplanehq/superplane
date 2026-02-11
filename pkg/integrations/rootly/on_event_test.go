package rootly

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnEvent__HandleWebhook(t *testing.T) {
	trigger := &OnEvent{}

	validConfig := map[string]any{}

	signatureFor := func(secret string, timestamp string, body []byte) string {
		payload := append([]byte(timestamp), body...)
		sig := computeHMACSHA256([]byte(secret), payload)
		return "t=" + timestamp + ",v1=" + sig
	}

	t.Run("missing X-Rootly-Signature -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-02-11T18:00:00Z"},"data":{"incident_event":{"id":"ie-456","event":"Investigation started","kind":"note"},"incident":{"status":"open"}}}`)
		headers := http.Header{}
		headers.Set("X-Rootly-Signature", "t=1234567890,v1=invalid")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
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
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("wrong event type -> no emit", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-123","issued_at":"2025-02-11T18:00:00Z"},"data":{"id":"inc-123","title":"Test Incident"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("valid incident event -> emit", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-02-11T18:00:00Z"},"data":{"incident_event":{"id":"ie-456","event":"Investigation started","kind":"note","occurred_at":"2025-02-11T17:59:00Z","created_at":"2025-02-11T18:00:00Z","user_display_name":"John Doe"},"incident":{"id":"inc-789","title":"Database issue","status":"open","severity":"high"}}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 1, eventContext.Count())

		payload := eventContext.Payloads[0]
		assert.Equal(t, "rootly.incident_event.created", payload.Type)

		data := payload.Data.(map[string]any)
		assert.Equal(t, "evt-123", data["event_id"])
		assert.Equal(t, "incident_event.created", data["event_type"])
		assert.Equal(t, "ie-456", data["id"])
		assert.Equal(t, "Investigation started", data["event"])
		assert.Equal(t, "note", data["kind"])
		assert.Equal(t, "John Doe", data["user_display_name"])
		assert.NotNil(t, data["incident"])
	})

	t.Run("filtered by incident status -> no emit", func(t *testing.T) {
		config := map[string]any{
			"incidentStatus": []string{"resolved"},
		}

		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-02-11T18:00:00Z"},"data":{"incident_event":{"id":"ie-456","event":"Investigation started","kind":"note"},"incident":{"status":"open"}}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: config,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, eventContext.Count()) // Filtered out
	})

	t.Run("filtered by severity -> no emit", func(t *testing.T) {
		config := map[string]any{
			"severity": []string{"critical"},
		}

		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-02-11T18:00:00Z"},"data":{"incident_event":{"id":"ie-456","event":"Investigation started","kind":"note"},"incident":{"severity":"medium"}}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: config,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, eventContext.Count()) // Filtered out
	})

	t.Run("filtered by event kind -> no emit", func(t *testing.T) {
		config := map[string]any{
			"eventKind": []string{"annotation"},
		}

		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-02-11T18:00:00Z"},"data":{"incident_event":{"id":"ie-456","event":"Investigation started","kind":"note"},"incident":{}}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: config,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, eventContext.Count()) // Filtered out
	})
}

func Test__OnEvent__Setup(t *testing.T) {
	trigger := &OnEvent{}

	ctx := core.TriggerContext{
		Configuration: map[string]any{},
		Integration: &contexts.IntegrationContext{
			WebhookRequests: []any{},
		},
	}

	err := trigger.Setup(ctx)
	assert.NoError(t, err)

	integrationCtx := ctx.Integration.(*contexts.IntegrationContext)
	assert.Len(t, integrationCtx.WebhookRequests, 1)

	webhookConfig := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
	expectedEvents := []string{"incident_event.created", "incident_event.updated"}
	assert.Equal(t, expectedEvents, webhookConfig.Events)
}