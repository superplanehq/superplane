package firehydrant

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIncident__HandleWebhook(t *testing.T) {
	trigger := &OnIncident{}

	signatureFor := func(secret string, body []byte) string {
		return computeHMACSHA256([]byte(secret), body)
	}

	t.Run("missing fh-signature -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		body := []byte(`{"data":{"incident":{"id":"inc-123"}},"event":{"operation":"CREATED","resource_type":"incident"}}`)
		headers := http.Header{}
		headers.Set("fh-signature", "invalid-hex-signature")

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

	t.Run("no secret configured -> skip verification", func(t *testing.T) {
		body := []byte(`{"data":{"incident":{"id":"inc-123","name":"Test"}},"event":{"operation":"CREATED","resource_type":"incident"}}`)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       http.Header{},
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: ""},
			Events:        eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("invalid JSON body -> 400", func(t *testing.T) {
		body := []byte("invalid json")
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

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

	t.Run("non-incident resource_type -> no emit", func(t *testing.T) {
		body := []byte(`{"data":{},"event":{"operation":"CREATED","resource_type":"change_event"}}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

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

	t.Run("UPDATED operation -> no emit", func(t *testing.T) {
		body := []byte(`{"data":{"incident":{"id":"inc-123"}},"event":{"operation":"UPDATED","resource_type":"incident"}}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

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

	t.Run("CREATED incident -> event emitted", func(t *testing.T) {
		body := []byte(`{
			"data": {
				"incident": {
					"id": "04d9fd1a-ba9c-417d-b396-58a6e2c374de",
					"name": "API Outage",
					"number": 42,
					"severity": {"slug": "SEV1", "description": "Critical"},
					"current_milestone": "started"
				}
			},
			"event": {
				"operation": "CREATED",
				"resource_type": "incident"
			}
		}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

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
		assert.Equal(t, "firehydrant.incident.created", payload.Type)
		assert.Equal(t, "incident.created", payload.Data.(map[string]any)["event"])
		assert.Equal(t, "CREATED", payload.Data.(map[string]any)["operation"])
		assert.NotNil(t, payload.Data.(map[string]any)["incident"])
	})

	t.Run("severity filter match -> event emitted", func(t *testing.T) {
		body := []byte(`{
			"data": {
				"incident": {
					"id": "inc-123",
					"name": "DB Outage",
					"severity": {"slug": "SEV1", "description": "Critical"}
				}
			},
			"event": {
				"operation": "CREATED",
				"resource_type": "incident"
			}
		}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"severities": []any{"SEV1", "SEV0"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("severity filter mismatch -> no emit", func(t *testing.T) {
		body := []byte(`{
			"data": {
				"incident": {
					"id": "inc-123",
					"name": "Minor Issue",
					"severity": {"slug": "SEV3", "description": "Minor"}
				}
			},
			"event": {
				"operation": "CREATED",
				"resource_type": "incident"
			}
		}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"severities": []any{"SEV1"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("no severity on incident with filter -> passes through", func(t *testing.T) {
		body := []byte(`{
			"data": {
				"incident": {
					"id": "inc-456",
					"name": "Unknown Severity Incident"
				}
			},
			"event": {
				"operation": "CREATED",
				"resource_type": "incident"
			}
		}`)
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("fh-signature", signatureFor(secret, body))

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"severities": []any{"SEV1"},
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})
}

func Test__OnIncident__Setup(t *testing.T) {
	trigger := &OnIncident{}

	t.Run("valid configuration -> webhook requested", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Configuration: map[string]any{},
		})

		require.NoError(t, err)
		assert.Len(t, integrationCtx.WebhookRequests, 1)
	})
}

func Test__verifyWebhookSignature(t *testing.T) {
	t.Run("empty secret -> skip verification", func(t *testing.T) {
		err := verifyWebhookSignature("", []byte("body"), []byte{})
		require.NoError(t, err)
	})

	t.Run("missing signature with secret -> error", func(t *testing.T) {
		err := verifyWebhookSignature("", []byte("body"), []byte("secret"))
		require.ErrorContains(t, err, "missing signature")
	})

	t.Run("signature mismatch -> error", func(t *testing.T) {
		err := verifyWebhookSignature("invalid-hex", []byte("body"), []byte("secret"))
		require.ErrorContains(t, err, "signature mismatch")
	})

	t.Run("valid signature -> no error", func(t *testing.T) {
		body := []byte("test body")
		secret := []byte("test secret")
		sig := computeHMACSHA256(secret, body)

		err := verifyWebhookSignature(sig, body, secret)
		require.NoError(t, err)
	})
}
