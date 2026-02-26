package terraform

import (
	"crypto/hmac"
	"crypto/sha512"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Webhook__ParseAndValidateWebhook(t *testing.T) {
	secret := "my-secret-key"
	body := []byte(`{"payload_version": 1, "run_id": "run-1234", "workspace_id": "ws-5678", "notifications": [{"trigger": "run:completed", "run_status": "applied"}]}`)

	h := hmac.New(sha512.New, []byte(secret))
	h.Write(body)
	signature := fmt.Sprintf("%x", h.Sum(nil))

	t.Run("missing signature -> 401", func(t *testing.T) {
		headers := http.Header{}

		_, code, err := ParseAndValidateWebhook(core.WebhookRequestContext{
			Headers:     headers,
			Body:        body,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
		})

		assert.Equal(t, http.StatusUnauthorized, code)
		assert.ErrorContains(t, err, "missing signature header")
	})

	t.Run("invalid signature -> 401", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", "invalid-hash")

		_, code, err := ParseAndValidateWebhook(core.WebhookRequestContext{
			Headers:     headers,
			Body:        body,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
		})

		assert.Equal(t, http.StatusUnauthorized, code)
		assert.ErrorContains(t, err, "invalid HMAC-SHA512 signature")
	})

	t.Run("valid v1 payload parsed successfully", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		payload, code, err := ParseAndValidateWebhook(core.WebhookRequestContext{
			Headers:     headers,
			Body:        body,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.NotNil(t, payload)

		// It should normalize run:completed
		assert.Equal(t, "run-1234", payload["runId"])
		assert.Equal(t, "ws-5678", payload["workspaceId"])
		assert.Equal(t, "run:completed", payload["action"])
	})

	t.Run("verification payload (handshake) -> 200 OK without events", func(t *testing.T) {
		handshakeBody := []byte(`{"payload_version": 1, "notifications": [{"trigger": "verification"}]}`)
		h2 := hmac.New(sha512.New, []byte(secret))
		h2.Write(handshakeBody)
		sig2 := fmt.Sprintf("%x", h2.Sum(nil))

		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", sig2)

		payload, code, err := ParseAndValidateWebhook(core.WebhookRequestContext{
			Headers:     headers,
			Body:        handshakeBody,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Nil(t, payload) // No event data should be returned for a handshake
	})
}

func Test__TerraformRunEvent__HandleWebhook(t *testing.T) {
	trigger := &TerraformRunEvent{}

	secret := "test-secret"
	body := []byte(`{"payload_version": 1, "run_id": "run-111", "workspace_id": "ws-222", "notifications": [{"trigger": "run:created", "run_status": "planned"}]}`)

	h := hmac.New(sha512.New, []byte(secret))
	h.Write(body)
	signature := fmt.Sprintf("%x", h.Sum(nil))

	t.Run("workspace mismatch -> ignored", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				// the node configuring workspace but it differs from payload's
				"workspaceId": "ws-DIFFERENT",
				"events":      []any{"run:created"},
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
			Events:      eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("action not configured -> ignored", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"workspaceId": "ws-222",
				"events":      []any{"run:completed"}, // waiting for completed, got created
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
			Events:      eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("workspace match + action match -> emitted", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"workspaceId": "ws-222",
				"events":      []any{"run:created"},
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
			Events:      eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())

		emittedData := eventContext.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "run-111", emittedData["runId"])
		assert.Equal(t, "ws-222", emittedData["workspaceId"])
	})
}
