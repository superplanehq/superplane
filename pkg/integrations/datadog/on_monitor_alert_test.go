package datadog

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnMonitorAlert__HandleWebhook(t *testing.T) {
	trigger := &OnMonitorAlert{}

	validConfig := map[string]any{
		"alertTransitions": []string{"Triggered", "Recovered"},
	}

	signatureFor := func(secret string, body []byte) string {
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		return fmt.Sprintf("%x", h.Sum(nil))
	}

	t.Run("missing X-Superplane-Signature-256 -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "missing X-Superplane-Signature-256 header")
	})

	t.Run("invalid signature format (empty after prefix) -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Superplane-Signature-256", "sha256=")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Events:  &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature format")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		body := []byte(`{"alert_transition":"Triggered"}`)
		headers := http.Header{}
		headers.Set("X-Superplane-Signature-256", "sha256=invalidsignature")

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

		headers := http.Header{}
		headers.Set("X-Superplane-Signature-256", "sha256="+signatureFor(secret, body))

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

	t.Run("alert transition not configured -> no emit", func(t *testing.T) {
		body := []byte(`{"alert_transition":"Warn","monitor_name":"CPU High"}`)
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Superplane-Signature-256", "sha256="+signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: validConfig,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("tags filter not matching -> no emit", func(t *testing.T) {
		body := []byte(`{"alert_transition":"Triggered","monitor_name":"CPU High","tags":["env:staging"]}`)
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Superplane-Signature-256", "sha256="+signatureFor(secret, body))

		config := map[string]any{
			"alertTransitions": []string{"Triggered"},
			"tags":             "env:prod",
		}

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: config,
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("valid signature and matching transition -> event is emitted", func(t *testing.T) {
		body := []byte(`{
			"id": "event-123",
			"event_type": "query_alert_monitor",
			"alert_type": "error",
			"alert_transition": "Triggered",
			"monitor_id": 12345,
			"monitor_name": "CPU High",
			"title": "[Triggered] CPU High",
			"body": "CPU exceeded threshold",
			"date": 1704067200,
			"tags": ["env:prod", "service:web"]
		}`)
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Superplane-Signature-256", "sha256="+signatureFor(secret, body))

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
		assert.Equal(t, "datadog.monitor.alert", payload.Type)

		data := payload.Data.(map[string]any)
		assert.Equal(t, "event-123", data["id"])
		assert.Equal(t, "Triggered", data["alert_transition"])
		assert.Equal(t, "CPU High", data["monitor_name"])
		assert.Equal(t, int64(12345), data["monitor_id"])
	})

	t.Run("matching tags filter -> event is emitted", func(t *testing.T) {
		body := []byte(`{"alert_transition":"Triggered","monitor_name":"CPU High","tags":["env:prod","service:web"]}`)
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Superplane-Signature-256", "sha256="+signatureFor(secret, body))

		config := map[string]any{
			"alertTransitions": []string{"Triggered"},
			"tags":             "env:prod",
		}

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
		assert.Equal(t, 1, eventContext.Count())
	})
}

func Test__OnMonitorAlert__Setup(t *testing.T) {
	trigger := &OnMonitorAlert{}

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("empty alert transitions -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"alertTransitions": []string{},
			},
		})

		require.ErrorContains(t, err, "at least one alert transition must be chosen")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"alertTransitions": []string{"Triggered"},
			},
		})

		require.NoError(t, err)
	})
}

func Test__parseTags(t *testing.T) {
	t.Run("empty string -> empty slice", func(t *testing.T) {
		result := parseTags("")
		assert.Empty(t, result)
	})

	t.Run("single tag -> single element slice", func(t *testing.T) {
		result := parseTags("env:prod")
		assert.Equal(t, []string{"env:prod"}, result)
	})

	t.Run("multiple tags -> multiple element slice", func(t *testing.T) {
		result := parseTags("env:prod, service:web, team:backend")
		assert.Equal(t, []string{"env:prod", "service:web", "team:backend"}, result)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		result := parseTags("  env:prod  ,  service:web  ")
		assert.Equal(t, []string{"env:prod", "service:web"}, result)
	})

	t.Run("ignores empty segments", func(t *testing.T) {
		result := parseTags("env:prod,,service:web")
		assert.Equal(t, []string{"env:prod", "service:web"}, result)
	})
}

func Test__matchesTags(t *testing.T) {
	t.Run("empty payload tags -> false", func(t *testing.T) {
		result := matchesTags([]string{}, "env:prod")
		assert.False(t, result)
	})

	t.Run("matching tag -> true", func(t *testing.T) {
		result := matchesTags([]string{"env:prod", "service:web"}, "env:prod")
		assert.True(t, result)
	})

	t.Run("no matching tag -> false", func(t *testing.T) {
		result := matchesTags([]string{"env:staging", "service:web"}, "env:prod")
		assert.False(t, result)
	})

	t.Run("any match from filter -> true", func(t *testing.T) {
		result := matchesTags([]string{"env:staging", "service:web"}, "env:prod, service:web")
		assert.True(t, result)
	})
}
