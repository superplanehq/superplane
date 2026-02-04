package rootly

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
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
		body := []byte(`{"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"evt-timeline-123","kind":"note"}}`)
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
		body := []byte(`{"event":{"type":"incident_event.updated","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"evt-timeline-123","kind":"note"}}`)
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
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{
				"id":"evt-timeline-123",
				"kind":"note",
				"occurred_at":"2025-01-01T00:00:00Z",
				"created_at":"2025-01-01T00:01:00Z",
				"user_display_name":"John Doe",
				"body":"Investigation started",
				"source":"web",
				"visibility":"internal",
				"incident":{"id":"inc-123","title":"Test Incident","status":"started","severity":"sev2"}
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

		data := payload.Data.(map[string]any)
		assert.Equal(t, "incident_event.created", data["event"])
		assert.Equal(t, "evt-123", data["event_id"])
		assert.Equal(t, "evt-timeline-123", data["id"])
		assert.Equal(t, "note", data["kind"])
		assert.Equal(t, "John Doe", data["user_display_name"])
		assert.Equal(t, "Investigation started", data["body"])
		assert.Equal(t, "web", data["source"])
		assert.Equal(t, "internal", data["visibility"])
		assert.NotNil(t, data["incident"])
	})

	t.Run("multiple events configured -> matching event emitted", func(t *testing.T) {
		multiEventConfig := map[string]any{
			"events": []string{"incident_event.created", "incident_event.updated"},
		}

		body := []byte(`{
			"event":{"type":"incident_event.updated","id":"evt-456","issued_at":"2025-01-01T12:00:00Z"},
			"data":{"id":"evt-timeline-456","kind":"status_update"}
		}`)
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
}

func Test__OnEvent__HandleWebhook_Filters(t *testing.T) {
	trigger := &OnEvent{}

	signatureFor := func(secret string, timestamp string, body []byte) string {
		payload := append([]byte(timestamp), body...)
		sig := computeHMACSHA256([]byte(secret), payload)
		return "t=" + timestamp + ",v1=" + sig
	}

	secret := "test-secret"
	timestamp := "1234567890"

	t.Run("filter by incident status - matching", func(t *testing.T) {
		config := map[string]any{
			"events":   []string{"incident_event.created"},
			"statuses": []string{"started"},
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{"id":"evt-123","kind":"note","incident":{"status":"started"}}
		}`)

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

	t.Run("filter by incident status - not matching", func(t *testing.T) {
		config := map[string]any{
			"events":   []string{"incident_event.created"},
			"statuses": []string{"resolved"},
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{"id":"evt-123","kind":"note","incident":{"status":"started"}}
		}`)

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
		require.Equal(t, 0, eventContext.Count())
	})

	t.Run("filter by severity - matching", func(t *testing.T) {
		config := map[string]any{
			"events":     []string{"incident_event.created"},
			"severities": []string{"sev1", "sev2"},
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{"id":"evt-123","kind":"note","incident":{"severity":"sev2"}}
		}`)

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

	t.Run("filter by severity - not matching", func(t *testing.T) {
		config := map[string]any{
			"events":     []string{"incident_event.created"},
			"severities": []string{"sev0"},
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{"id":"evt-123","kind":"note","incident":{"severity":"sev3"}}
		}`)

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
		require.Equal(t, 0, eventContext.Count())
	})

	t.Run("filter by event kind - matching", func(t *testing.T) {
		config := map[string]any{
			"events":      []string{"incident_event.created"},
			"event_kinds": []string{"note", "status_update"},
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{"id":"evt-123","kind":"note"}
		}`)

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

	t.Run("filter by event kind - not matching", func(t *testing.T) {
		config := map[string]any{
			"events":      []string{"incident_event.created"},
			"event_kinds": []string{"alert"},
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{"id":"evt-123","kind":"note"}
		}`)

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
		require.Equal(t, 0, eventContext.Count())
	})

	t.Run("filter by visibility - matching", func(t *testing.T) {
		config := map[string]any{
			"events":     []string{"incident_event.created"},
			"visibility": "internal",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{"id":"evt-123","kind":"note","visibility":"internal"}
		}`)

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

	t.Run("filter by visibility - not matching", func(t *testing.T) {
		config := map[string]any{
			"events":     []string{"incident_event.created"},
			"visibility": "external",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{"id":"evt-123","kind":"note","visibility":"internal"}
		}`)

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
		require.Equal(t, 0, eventContext.Count())
	})

	t.Run("filter by source - matching", func(t *testing.T) {
		config := map[string]any{
			"events":  []string{"incident_event.created"},
			"sources": []string{"web", "api"},
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{"id":"evt-123","kind":"note","source":"web"}
		}`)

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

	t.Run("filter by source - not matching", func(t *testing.T) {
		config := map[string]any{
			"events":  []string{"incident_event.created"},
			"sources": []string{"email"},
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{"id":"evt-123","kind":"note","source":"web"}
		}`)

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
		require.Equal(t, 0, eventContext.Count())
	})

	t.Run("filter by services - matching", func(t *testing.T) {
		config := map[string]any{
			"events":   []string{"incident_event.created"},
			"services": "api-gateway",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{"id":"evt-123","kind":"note","incident":{"services":[{"name":"api-gateway"}]}}
		}`)

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

	t.Run("filter by services - not matching", func(t *testing.T) {
		config := map[string]any{
			"events":   []string{"incident_event.created"},
			"services": "database",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{"id":"evt-123","kind":"note","incident":{"services":[{"name":"api-gateway"}]}}
		}`)

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
		require.Equal(t, 0, eventContext.Count())
	})

	t.Run("filter by teams - matching", func(t *testing.T) {
		config := map[string]any{
			"events": []string{"incident_event.created"},
			"teams":  "platform",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{"id":"evt-123","kind":"note","incident":{"teams":[{"name":"platform"}]}}
		}`)

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

	t.Run("filter by teams - not matching", func(t *testing.T) {
		config := map[string]any{
			"events": []string{"incident_event.created"},
			"teams":  "infrastructure",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{"id":"evt-123","kind":"note","incident":{"teams":[{"name":"platform"}]}}
		}`)

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
		require.Equal(t, 0, eventContext.Count())
	})

	t.Run("multiple filters - all matching", func(t *testing.T) {
		config := map[string]any{
			"events":      []string{"incident_event.created"},
			"statuses":    []string{"started"},
			"severities":  []string{"sev2"},
			"event_kinds": []string{"note"},
			"visibility":  "internal",
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{
				"id":"evt-123",
				"kind":"note",
				"visibility":"internal",
				"incident":{"status":"started","severity":"sev2"}
			}
		}`)

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

	t.Run("multiple filters - one not matching", func(t *testing.T) {
		config := map[string]any{
			"events":      []string{"incident_event.created"},
			"statuses":    []string{"started"},
			"severities":  []string{"sev1"}, // Different severity
			"event_kinds": []string{"note"},
		}

		body := []byte(`{
			"event":{"type":"incident_event.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},
			"data":{
				"id":"evt-123",
				"kind":"note",
				"incident":{"status":"started","severity":"sev2"}
			}
		}`)

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
		require.Equal(t, 0, eventContext.Count())
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

	t.Run("configuration with filters -> webhook request", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Configuration: OnEventConfiguration{
				Events:     []string{"incident_event.created"},
				Statuses:   []string{"started"},
				Severities: []string{"sev1"},
				EventKinds: []string{"note"},
				Visibility: "internal",
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
	})
}

func Test__OnEvent__Metadata(t *testing.T) {
	trigger := &OnEvent{}

	t.Run("Name returns correct value", func(t *testing.T) {
		assert.Equal(t, "rootly.onEvent", trigger.Name())
	})

	t.Run("Label returns correct value", func(t *testing.T) {
		assert.Equal(t, "On Event", trigger.Label())
	})

	t.Run("Description returns correct value", func(t *testing.T) {
		assert.Equal(t, "Listen to incident timeline events", trigger.Description())
	})

	t.Run("Icon returns correct value", func(t *testing.T) {
		assert.Equal(t, "message-square", trigger.Icon())
	})

	t.Run("Color returns correct value", func(t *testing.T) {
		assert.Equal(t, "gray", trigger.Color())
	})

	t.Run("Documentation is not empty", func(t *testing.T) {
		assert.NotEmpty(t, trigger.Documentation())
	})

	t.Run("Configuration returns expected fields", func(t *testing.T) {
		config := trigger.Configuration()
		assert.NotEmpty(t, config)

		// Check events field exists and is required
		var eventsField *configuration.Field
		for i := range config {
			if config[i].Name == "events" {
				eventsField = &config[i]
				break
			}
		}
		require.NotNil(t, eventsField)
		assert.True(t, eventsField.Required)

		// Check optional filter fields
		fieldNames := make([]string, len(config))
		for i, f := range config {
			fieldNames[i] = f.Name
		}
		assert.Contains(t, fieldNames, "statuses")
		assert.Contains(t, fieldNames, "severities")
		assert.Contains(t, fieldNames, "services")
		assert.Contains(t, fieldNames, "teams")
		assert.Contains(t, fieldNames, "sources")
		assert.Contains(t, fieldNames, "visibility")
		assert.Contains(t, fieldNames, "event_kinds")
	})

	t.Run("Actions returns empty slice", func(t *testing.T) {
		assert.Empty(t, trigger.Actions())
	})

	t.Run("Cleanup returns nil", func(t *testing.T) {
		err := trigger.Cleanup(core.TriggerContext{})
		assert.NoError(t, err)
	})

	t.Run("ExampleData returns valid structure", func(t *testing.T) {
		example := trigger.ExampleData()
		assert.NotNil(t, example)
		assert.Equal(t, "rootly.onEvent", example["type"])
		assert.NotNil(t, example["data"])
		assert.NotNil(t, example["timestamp"])

		data, ok := example["data"].(map[string]any)
		require.True(t, ok)
		assert.NotEmpty(t, data["event"])
		assert.NotEmpty(t, data["kind"])
		assert.NotNil(t, data["incident"])
	})
}

func Test__matchesEventFilters(t *testing.T) {
	t.Run("nil data with no filters returns true", func(t *testing.T) {
		config := OnEventConfiguration{}
		assert.True(t, matchesEventFilters(nil, config))
	})

	t.Run("nil data with filters returns false", func(t *testing.T) {
		config := OnEventConfiguration{Statuses: []string{"started"}}
		assert.False(t, matchesEventFilters(nil, config))
	})

	t.Run("empty config matches everything", func(t *testing.T) {
		config := OnEventConfiguration{}
		data := map[string]any{
			"kind":       "note",
			"visibility": "internal",
			"incident":   map[string]any{"status": "started"},
		}
		assert.True(t, matchesEventFilters(data, config))
	})

	t.Run("missing incident data when filtering by status", func(t *testing.T) {
		config := OnEventConfiguration{Statuses: []string{"started"}}
		data := map[string]any{"kind": "note"}
		// Should not match because incident is missing
		assert.False(t, matchesEventFilters(data, config))
	})
}

// Ensure configuration package is used for field type assertions
var _ configuration.Field
