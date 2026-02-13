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

	validConfig := map[string]any{
		"events": []string{"incident_event.created"},
	}

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
		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2026-01-19T00:00:00Z"},"data":{"id":"ie-123","event":"Note added"}}`)
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

	t.Run("event type not configured -> no emit", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident_event.updated","id":"evt-123","issued_at":"2026-01-19T00:00:00Z"},"data":{"id":"ie-123","event":"Note updated"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig, // Only "incident_event.created" is configured
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("valid signature and matching event -> event is emitted", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2026-01-19T00:00:00Z"},"data":{"id":"ie-123","event":"Investigation update: database connections stabilized.","kind":"note","visibility":"internal","occurred_at":"2026-01-19T12:00:00Z","created_at":"2026-01-19T12:00:00Z","user_display_name":"John Doe","incident":{"id":"inc-456","title":"API latency spike","status":"started","severity":"sev1"}}}`)
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

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())

		payload := eventContext.Payloads[0]
		assert.Equal(t, "rootly.incident_event.created", payload.Type)
		assert.Equal(t, "incident_event.created", payload.Data.(map[string]any)["event"])
		assert.Equal(t, "evt-123", payload.Data.(map[string]any)["event_id"])
		assert.NotNil(t, payload.Data.(map[string]any)["incident_event"])
		assert.NotNil(t, payload.Data.(map[string]any)["incident"])
	})

	t.Run("visibility filter matches -> event is emitted", func(t *testing.T) {
		configWithVisibility := map[string]any{
			"events":     []string{"incident_event.created"},
			"visibility": []string{"internal"},
		}

		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2026-01-19T00:00:00Z"},"data":{"id":"ie-123","event":"Internal note","visibility":"internal","incident":{"id":"inc-456","title":"Test","status":"started","severity":"sev1"}}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: configWithVisibility,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})

	t.Run("visibility filter does not match -> no emit", func(t *testing.T) {
		configWithVisibility := map[string]any{
			"events":     []string{"incident_event.created"},
			"visibility": []string{"external"},
		}

		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2026-01-19T00:00:00Z"},"data":{"id":"ie-123","event":"Internal note","visibility":"internal","incident":{"id":"inc-456","title":"Test","status":"started","severity":"sev1"}}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: configWithVisibility,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("incident status filter matches -> event is emitted", func(t *testing.T) {
		configWithStatus := map[string]any{
			"events": []string{"incident_event.created"},
			"status": []string{"started", "mitigated"},
		}

		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2026-01-19T00:00:00Z"},"data":{"id":"ie-123","event":"Note","visibility":"internal","incident":{"id":"inc-456","title":"Test","status":"started","severity":"sev1"}}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: configWithStatus,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})

	t.Run("incident status filter does not match -> no emit", func(t *testing.T) {
		configWithStatus := map[string]any{
			"events": []string{"incident_event.created"},
			"status": []string{"resolved"},
		}

		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2026-01-19T00:00:00Z"},"data":{"id":"ie-123","event":"Note","visibility":"internal","incident":{"id":"inc-456","title":"Test","status":"started","severity":"sev1"}}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: configWithStatus,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("severity filter matches -> event is emitted", func(t *testing.T) {
		configWithSeverity := map[string]any{
			"events":   []string{"incident_event.created"},
			"severity": []string{"sev1"},
		}

		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2026-01-19T00:00:00Z"},"data":{"id":"ie-123","event":"Note","visibility":"internal","incident":{"id":"inc-456","title":"Test","status":"started","severity":"sev1"}}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: configWithSeverity,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})

	t.Run("severity filter does not match -> no emit", func(t *testing.T) {
		configWithSeverity := map[string]any{
			"events":   []string{"incident_event.created"},
			"severity": []string{"sev0"},
		}

		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2026-01-19T00:00:00Z"},"data":{"id":"ie-123","event":"Note","visibility":"internal","incident":{"id":"inc-456","title":"Test","status":"started","severity":"sev1"}}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: configWithSeverity,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("multiple filters combined -> all must match", func(t *testing.T) {
		configWithMultiple := map[string]any{
			"events":     []string{"incident_event.created"},
			"status":     []string{"started"},
			"severity":   []string{"sev1"},
			"visibility": []string{"internal"},
		}

		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2026-01-19T00:00:00Z"},"data":{"id":"ie-123","event":"Note","visibility":"internal","incident":{"id":"inc-456","title":"Test","status":"started","severity":"sev1"}}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: configWithMultiple,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})
}

func Test__OnEvent__Setup(t *testing.T) {
	trigger := &OnEvent{}

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("at least one event required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Configuration: OnEventConfiguration{Events: []string{}},
		})

		require.ErrorContains(t, err, "at least one event type must be chosen")
	})

	t.Run("valid configuration -> webhook request", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Configuration: OnEventConfiguration{Events: []string{"incident_event.created"}},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)

		webhookConfig := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, []string{"incident_event.created"}, webhookConfig.Events)
	})

	t.Run("multiple events configured -> webhook request with all events", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Configuration: OnEventConfiguration{
				Events: []string{"incident_event.created", "incident_event.updated"},
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)

		webhookConfig := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Len(t, webhookConfig.Events, 2)
	})
}
