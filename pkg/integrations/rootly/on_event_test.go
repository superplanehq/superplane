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

	t.Run("invalid signature format -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Rootly-Signature", "invalid")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2026-01-01T00:00:00Z"},"data":{"id":"ie-123","event":"Investigation started"}}`)
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
		body := []byte(`{"event":{"type":"incident_event.updated","id":"evt-123","issued_at":"2026-01-01T00:00:00Z"},"data":{"id":"ie-123","event":"Note updated"}}`)
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
		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2026-01-01T00:00:00Z"},
			"data":{
				"id":"ie-123",
				"event":"Investigation started",
				"kind":"note",
				"visibility":"internal",
				"occurred_at":"2026-01-01T00:00:00Z",
				"created_at":"2026-01-01T00:00:00Z",
				"user":{"full_name":"Jane Smith"},
				"incident":{"id":"inc-123","title":"API Outage","status":"started","severity":"sev1"}
			}
		}`)
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
		assert.Equal(t, "ie-123", payload.Data.(map[string]any)["id"])
		assert.Equal(t, "Investigation started", payload.Data.(map[string]any)["event_content"])
		assert.Equal(t, "note", payload.Data.(map[string]any)["kind"])
		assert.Equal(t, "internal", payload.Data.(map[string]any)["visibility"])
		assert.Equal(t, "Jane Smith", payload.Data.(map[string]any)["user_display_name"])
		assert.NotNil(t, payload.Data.(map[string]any)["incident"])
	})

	t.Run("multiple events configured -> matching event emitted", func(t *testing.T) {
		multiEventConfig := map[string]any{
			"events": []string{"incident_event.created", "incident_event.updated"},
		}

		body := []byte(`{"event":{"type":"incident_event.updated","id":"evt-456","issued_at":"2026-01-01T12:00:00Z"},"data":{"id":"ie-456","event":"Note updated"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: multiEventConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())

		payload := eventContext.Payloads[0]
		assert.Equal(t, "rootly.incident_event.updated", payload.Type)
	})

	t.Run("visibility filter matches -> event emitted", func(t *testing.T) {
		config := map[string]any{
			"events":     []string{"incident_event.created"},
			"visibility": "internal",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-789","issued_at":"2026-01-01T00:00:00Z"},
			"data":{"id":"ie-789","event":"Internal note","visibility":"internal","kind":"note"}
		}`)
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

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})

	t.Run("visibility filter does not match -> no emit", func(t *testing.T) {
		config := map[string]any{
			"events":     []string{"incident_event.created"},
			"visibility": "external",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-789","issued_at":"2026-01-01T00:00:00Z"},
			"data":{"id":"ie-789","event":"Internal note","visibility":"internal","kind":"note"}
		}`)
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

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("event kind filter matches -> event emitted", func(t *testing.T) {
		config := map[string]any{
			"events":    []string{"incident_event.created"},
			"eventKind": "note",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-100","issued_at":"2026-01-01T00:00:00Z"},
			"data":{"id":"ie-100","event":"A note","kind":"note"}
		}`)
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

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})

	t.Run("incident status filter matches -> event emitted", func(t *testing.T) {
		config := map[string]any{
			"events":         []string{"incident_event.created"},
			"incidentStatus": "started",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-200","issued_at":"2026-01-01T00:00:00Z"},
			"data":{"id":"ie-200","event":"Note added","incident":{"id":"inc-200","status":"started","severity":"sev1"}}
		}`)
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

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})

	t.Run("incident status filter does not match -> no emit", func(t *testing.T) {
		config := map[string]any{
			"events":         []string{"incident_event.created"},
			"incidentStatus": "resolved",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-201","issued_at":"2026-01-01T00:00:00Z"},
			"data":{"id":"ie-201","event":"Note added","incident":{"id":"inc-201","status":"started","severity":"sev1"}}
		}`)
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

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("severity filter matches -> event emitted", func(t *testing.T) {
		config := map[string]any{
			"events":   []string{"incident_event.created"},
			"severity": "sev1",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-300","issued_at":"2026-01-01T00:00:00Z"},
			"data":{"id":"ie-300","event":"Note","incident":{"id":"inc-300","status":"started","severity":"sev1"}}
		}`)
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

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})

	t.Run("service filter matches -> event emitted", func(t *testing.T) {
		config := map[string]any{
			"events":  []string{"incident_event.created"},
			"service": "api-gateway",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-400","issued_at":"2026-01-01T00:00:00Z"},
			"data":{"id":"ie-400","event":"Note","incident":{"id":"inc-400","status":"started","services":[{"name":"API Gateway","slug":"api-gateway"}]}}
		}`)
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

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})

	t.Run("service filter does not match -> no emit", func(t *testing.T) {
		config := map[string]any{
			"events":  []string{"incident_event.created"},
			"service": "database",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-401","issued_at":"2026-01-01T00:00:00Z"},
			"data":{"id":"ie-401","event":"Note","incident":{"id":"inc-401","status":"started","services":[{"name":"API Gateway","slug":"api-gateway"}]}}
		}`)
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

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("team filter matches -> event emitted", func(t *testing.T) {
		config := map[string]any{
			"events": []string{"incident_event.created"},
			"team":   "platform",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-500","issued_at":"2026-01-01T00:00:00Z"},
			"data":{"id":"ie-500","event":"Note","incident":{"id":"inc-500","status":"started","groups":[{"name":"Platform","slug":"platform"}]}}
		}`)
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

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})

	t.Run("multiple filters combined -> all must match", func(t *testing.T) {
		config := map[string]any{
			"events":         []string{"incident_event.created"},
			"visibility":     "internal",
			"incidentStatus": "started",
			"severity":       "sev1",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-600","issued_at":"2026-01-01T00:00:00Z"},
			"data":{"id":"ie-600","event":"Note","visibility":"internal","incident":{"id":"inc-600","status":"started","severity":"sev1"}}
		}`)
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

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})

	t.Run("multiple filters combined -> one does not match -> no emit", func(t *testing.T) {
		config := map[string]any{
			"events":         []string{"incident_event.created"},
			"visibility":     "internal",
			"incidentStatus": "resolved", // does not match
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-601","issued_at":"2026-01-01T00:00:00Z"},
			"data":{"id":"ie-601","event":"Note","visibility":"internal","incident":{"id":"inc-601","status":"started","severity":"sev1"}}
		}`)
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

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
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

	t.Run("valid configuration with filters -> webhook request", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Configuration: OnEventConfiguration{
				Events:     []string{"incident_event.created", "incident_event.updated"},
				Visibility: "internal",
				EventKind:  "note",
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)

		webhookConfig := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Len(t, webhookConfig.Events, 2)
	})
}
