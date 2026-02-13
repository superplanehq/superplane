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
		"events": []string{"incident.created", "incident.updated"},
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
		body := []byte(`{"event":{"type":"incident.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-123","title":"Test Incident"}}`)
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
		body := []byte(`{"event":{"type":"incident.resolved","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-123","title":"Test Incident"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig, // Only "incident.created" and "incident.updated" configured
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("valid signature and matching event -> event is emitted", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-123","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-123","title":"Test Incident","status":"started"}}`)
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
		assert.Equal(t, "rootly.incident.created", payload.Type)
		assert.Equal(t, "incident.created", payload.Data.(map[string]any)["event"])
		assert.Equal(t, "evt-123", payload.Data.(map[string]any)["event_id"])
		assert.NotNil(t, payload.Data.(map[string]any)["incident"])
	})

	t.Run("status filter matches -> event is emitted", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-456","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-456","title":"Test Incident","status":"started"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		configWithStatus := map[string]any{
			"events": []string{"incident.created"},
			"status": "started",
		}

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

	t.Run("status filter does not match -> no emit", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-789","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-789","title":"Test Incident","status":"started"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		configWithStatus := map[string]any{
			"events": []string{"incident.created"},
			"status": "resolved",
		}

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

	t.Run("severity filter with string value -> matches", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.updated","id":"evt-sev","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-sev","title":"Severity Test","status":"started","severity":"sev1"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		configWithSeverity := map[string]any{
			"events":   []string{"incident.updated"},
			"severity": "sev1",
		}

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

	t.Run("severity filter with object value -> matches name", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.updated","id":"evt-sev2","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-sev2","title":"Severity Object Test","status":"started","severity":{"name":"sev2","slug":"sev2"}}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		configWithSeverity := map[string]any{
			"events":   []string{"incident.updated"},
			"severity": "sev2",
		}

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
		body := []byte(`{"event":{"type":"incident.updated","id":"evt-sev3","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-sev3","title":"Severity Mismatch","status":"started","severity":"sev1"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		configWithSeverity := map[string]any{
			"events":   []string{"incident.updated"},
			"severity": "sev2",
		}

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

	t.Run("service filter matches -> event is emitted", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-svc","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-svc","title":"Service Test","status":"started","services":[{"name":"Production API","slug":"production-api"}]}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		configWithService := map[string]any{
			"events":  []string{"incident.created"},
			"service": "Production API",
		}

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: configWithService,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})

	t.Run("service filter does not match -> no emit", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-svc2","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-svc2","title":"Service Mismatch","status":"started","services":[{"name":"Staging API","slug":"staging-api"}]}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		configWithService := map[string]any{
			"events":  []string{"incident.created"},
			"service": "Production API",
		}

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: configWithService,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("team filter matches -> event is emitted", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-team","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-team","title":"Team Test","status":"started","teams":[{"name":"Platform","slug":"platform"}]}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		configWithTeam := map[string]any{
			"events": []string{"incident.created"},
			"team":   "Platform",
		}

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: configWithTeam,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
	})

	t.Run("team filter does not match -> no emit", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-team2","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-team2","title":"Team Mismatch","status":"started","teams":[{"name":"Backend","slug":"backend"}]}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		configWithTeam := map[string]any{
			"events": []string{"incident.created"},
			"team":   "Platform",
		}

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: configWithTeam,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("visibility filter on timeline events -> matches", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.updated","id":"evt-vis","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-vis","title":"Visibility Test","status":"started","events":[{"kind":"event","source":"web","visibility":"internal"}]}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		configWithVisibility := map[string]any{
			"events":     []string{"incident.updated"},
			"visibility": "internal",
		}

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

	t.Run("kind filter on timeline events -> no matching event", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.updated","id":"evt-kind","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-kind","title":"Kind Test","status":"started","events":[{"kind":"trail","source":"web","visibility":"internal"}]}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		configWithKind := map[string]any{
			"events": []string{"incident.updated"},
			"kind":   "event",
		}

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: configWithKind,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("multiple filters combined -> all must match", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-multi","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-multi","title":"Multi Filter","status":"started","severity":"sev1","services":[{"name":"Production API"}],"teams":[{"name":"Platform"}],"events":[{"kind":"event","source":"web","visibility":"internal"}]}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		multiConfig := map[string]any{
			"events":     []string{"incident.created"},
			"status":     "started",
			"severity":   "sev1",
			"service":    "Production API",
			"team":       "Platform",
			"visibility": "internal",
			"kind":       "event",
			"source":     "web",
		}

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: multiConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())

		payload := eventContext.Payloads[0]
		assert.Equal(t, "rootly.incident.created", payload.Type)
	})

	t.Run("no timeline event filters with missing events array -> passes", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-noevt","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-noevt","title":"No Events Array","status":"started"}}`)
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
	})

	t.Run("status filter with missing status field in webhook data -> no emit", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-nostatus","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-nostatus","title":"No Status Field","severity":"sev1"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		configWithStatus := map[string]any{
			"events": []string{"incident.created"},
			"status": "started",
		}

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

	t.Run("severity filter with missing severity field in webhook data -> no emit", func(t *testing.T) {
		body := []byte(`{"event":{"type":"incident.created","id":"evt-noseverity","issued_at":"2025-01-01T00:00:00Z"},"data":{"id":"inc-noseverity","title":"No Severity Field","status":"started"}}`)
		secret := "test-secret"
		timestamp := "1234567890"

		headers := http.Header{}
		headers.Set("X-Rootly-Signature", signatureFor(secret, timestamp, body))

		configWithSeverity := map[string]any{
			"events":   []string{"incident.created"},
			"severity": "sev1",
		}

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
			Configuration: OnEventConfiguration{Events: []string{"incident.created"}},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)

		webhookConfig := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, []string{"incident.created"}, webhookConfig.Events)
	})

	t.Run("multiple events configured -> webhook request with all events", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration: integrationCtx,
			Configuration: OnEventConfiguration{
				Events: []string{"incident.created", "incident.updated", "incident.mitigated"},
			},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)

		webhookConfig := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Len(t, webhookConfig.Events, 3)
	})
}

func Test__matchesEventFilters(t *testing.T) {
	t.Run("nil data with filters configured -> fails filter check", func(t *testing.T) {
		result := matchesEventFilters(nil, OnEventConfiguration{
			Status:   "started",
			Severity: "sev1",
		})
		assert.False(t, result)
	})

	t.Run("empty config -> passes all filters", func(t *testing.T) {
		data := map[string]any{
			"status":   "started",
			"severity": "sev1",
		}
		result := matchesEventFilters(data, OnEventConfiguration{})
		assert.True(t, result)
	})

	t.Run("service with no services array -> does not match", func(t *testing.T) {
		data := map[string]any{
			"status": "started",
		}
		result := matchesEventFilters(data, OnEventConfiguration{
			Service: "Production API",
		})
		assert.False(t, result)
	})

	t.Run("team with no teams array -> does not match", func(t *testing.T) {
		data := map[string]any{
			"status": "started",
		}
		result := matchesEventFilters(data, OnEventConfiguration{
			Team: "Platform",
		})
		assert.False(t, result)
	})

	t.Run("timeline filters with no events array -> passes", func(t *testing.T) {
		data := map[string]any{
			"status": "started",
		}
		result := matchesEventFilters(data, OnEventConfiguration{
			Visibility: "internal",
		})
		assert.True(t, result)
	})

	t.Run("status filter with missing status field -> does not match", func(t *testing.T) {
		data := map[string]any{
			"severity": "sev1",
		}
		result := matchesEventFilters(data, OnEventConfiguration{
			Status: "started",
		})
		assert.False(t, result)
	})

	t.Run("severity filter with missing severity field -> does not match", func(t *testing.T) {
		data := map[string]any{
			"status": "started",
		}
		result := matchesEventFilters(data, OnEventConfiguration{
			Severity: "sev1",
		})
		assert.False(t, result)
	})
}
