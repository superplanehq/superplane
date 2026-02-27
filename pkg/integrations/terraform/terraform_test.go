package terraform

import (
	"crypto/hmac"
	"crypto/sha512"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func signPayload(secret string, body []byte) string {
	h := hmac.New(sha512.New, []byte(secret))
	h.Write(body)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func Test__ParseAndValidateWebhook(t *testing.T) {
	secret := "my-secret-key"

	t.Run("missing signature with secret configured -> 401", func(t *testing.T) {
		body := []byte(`{"payload_version": 1}`)
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
		body := []byte(`{"payload_version": 1}`)
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

	t.Run("valid signature -> 200", func(t *testing.T) {
		body := []byte(`{"payload_version": 1, "run_id": "run-123", "workspace_id": "ws-456", "notifications": [{"trigger": "run:completed", "run_status": "applied"}]}`)
		signature := signPayload(secret, body)

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
		assert.Equal(t, "run-123", payload["runId"])
		assert.Equal(t, "ws-456", payload["workspaceId"])
		assert.Equal(t, "run:completed", payload["action"])
		assert.Equal(t, "applied", payload["runStatus"])
	})

	t.Run("missing secret configuration -> 500", func(t *testing.T) {
		body := []byte(`{"payload_version": 1, "run_id": "run-123", "workspace_id": "ws-456", "notifications": [{"trigger": "run:created", "run_status": "pending"}]}`)
		headers := http.Header{}

		_, code, err := ParseAndValidateWebhook(core.WebhookRequestContext{
			Headers:     headers,
			Body:        body,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{}},
		})

		assert.Equal(t, http.StatusInternalServerError, code)
		assert.ErrorContains(t, err, "failed to get webhook secret")
	})

	t.Run("empty secret configuration -> 500", func(t *testing.T) {
		body := []byte(`{"payload_version": 1, "run_id": "run-123", "workspace_id": "ws-456", "notifications": [{"trigger": "run:created", "run_status": "pending"}]}`)
		headers := http.Header{}

		_, code, err := ParseAndValidateWebhook(core.WebhookRequestContext{
			Headers:     headers,
			Body:        body,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": ""}},
		})

		assert.Equal(t, http.StatusInternalServerError, code)
		assert.ErrorContains(t, err, "webhook secret is not configured")
	})

	t.Run("verification handshake -> 200 with nil payload", func(t *testing.T) {
		body := []byte(`{"payload_version": 1, "notifications": [{"trigger": "verification"}]}`)
		signature := signPayload(secret, body)

		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		payload, code, err := ParseAndValidateWebhook(core.WebhookRequestContext{
			Headers:     headers,
			Body:        body,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Nil(t, payload)
	})

	t.Run("invalid JSON -> 400", func(t *testing.T) {
		body := []byte(`not valid json`)
		signature := signPayload(secret, body)

		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		_, code, err := ParseAndValidateWebhook(core.WebhookRequestContext{
			Headers:     headers,
			Body:        body,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "invalid json")
	})

	t.Run("missing payload_version -> 400", func(t *testing.T) {
		body := []byte(`{"notifications": []}`)
		signature := signPayload(secret, body)

		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		_, code, err := ParseAndValidateWebhook(core.WebhookRequestContext{
			Headers:     headers,
			Body:        body,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "payload_version")
	})

	t.Run("payload version 2 (drift detection) -> parsed correctly", func(t *testing.T) {
		body := []byte(`{"payload_version": 2, "workspace_id": "ws-789", "workspace_name": "prod", "organization_name": "acme", "notifications": [{"trigger": "assessment:drifted", "run_status": ""}]}`)
		signature := signPayload(secret, body)

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
		assert.Equal(t, "ws-789", payload["workspaceId"])
		assert.Equal(t, "assessment:drifted", payload["action"])
	})
}

func Test__TerraformRunEvent__HandleWebhook(t *testing.T) {
	trigger := &RunEvent{}
	secret := "test-secret"

	makeBody := func(runID, workspaceID, action, status string) []byte {
		return []byte(fmt.Sprintf(`{"payload_version": 1, "run_id": "%s", "workspace_id": "%s", "workspace_name": "my-ws", "organization_name": "my-org", "notifications": [{"trigger": "%s", "run_status": "%s"}]}`, runID, workspaceID, action, status))
	}

	t.Run("workspace ID mismatch -> event not emitted", func(t *testing.T) {
		body := makeBody("run-111", "ws-222", "run:created", "pending")
		signature := signPayload(secret, body)

		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
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

	t.Run("workspace org/name format match -> event emitted", func(t *testing.T) {
		body := makeBody("run-111", "ws-222", "run:created", "pending")
		signature := signPayload(secret, body)

		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"workspaceId": "my-org/my-ws",
				"events":      []any{"run:created"},
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
			Events:      eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("action not in configured events list -> event not emitted", func(t *testing.T) {
		body := makeBody("run-111", "ws-222", "run:created", "pending")
		signature := signPayload(secret, body)

		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"workspaceId": "ws-222",
				"events":      []any{"run:completed", "run:errored"},
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
			Events:      eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("workspace and action match -> event emitted with correct data", func(t *testing.T) {
		body := makeBody("run-111", "ws-222", "run:completed", "applied")
		signature := signPayload(secret, body)

		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"workspaceId": "ws-222",
				"events":      []any{"run:completed"},
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
			Events:      eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())

		emittedData := eventContext.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "run-111", emittedData["runId"])
		assert.Equal(t, "ws-222", emittedData["workspaceId"])
		assert.Equal(t, "run:completed", emittedData["action"])
		assert.Equal(t, "applied", emittedData["runStatus"])
		assert.Equal(t, "my-ws", emittedData["workspaceName"])
		assert.Equal(t, "my-org", emittedData["organizationName"])
	})

	t.Run("SuperPlane-initiated run (gear emoji) -> event not emitted by default", func(t *testing.T) {
		body := []byte(`{"payload_version": 1, "run_id": "run-111", "run_message": "⚙ Triggered by SuperPlane", "workspace_id": "ws-222", "notifications": [{"trigger": "run:created", "run_status": "pending"}]}`)
		signature := signPayload(secret, body)

		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"workspaceId":           "ws-222",
				"events":                []any{"run:created"},
				"includeSuperPlaneRuns": false,
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
			Events:      eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("SuperPlane-initiated run with includeSuperPlaneRuns=true -> event emitted", func(t *testing.T) {
		body := []byte(`{"payload_version": 1, "run_id": "run-111", "run_message": "⚙ Triggered by SuperPlane", "workspace_id": "ws-222", "notifications": [{"trigger": "run:created", "run_status": "pending"}]}`)
		signature := signPayload(secret, body)

		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"workspaceId":           "ws-222",
				"events":                []any{"run:created"},
				"includeSuperPlaneRuns": true,
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
			Events:      eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("multiple events configured -> only matching events emitted", func(t *testing.T) {
		body := makeBody("run-111", "ws-222", "run:errored", "errored")
		signature := signPayload(secret, body)

		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"workspaceId": "ws-222",
				"events":      []any{"run:created", "run:errored", "run:completed"},
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
			Events:      eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "terraform.runEvent", eventContext.Payloads[0].Type)
	})
}

func Test__TerraformNeedsAttention__HandleWebhook(t *testing.T) {
	trigger := &TerraformNeedsAttention{}
	secret := "test-secret"

	t.Run("action is not run:needs_attention -> event not emitted", func(t *testing.T) {
		body := []byte(`{"payload_version": 1, "run_id": "run-111", "workspace_id": "ws-222", "notifications": [{"trigger": "run:completed", "run_status": "applied"}]}`)
		signature := signPayload(secret, body)

		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"workspaceId": "ws-222",
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
			Events:      eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("workspace ID mismatch -> event not emitted", func(t *testing.T) {
		body := []byte(`{"payload_version": 1, "run_id": "run-111", "workspace_id": "ws-222", "notifications": [{"trigger": "run:needs_attention", "run_status": "policy_override"}]}`)
		signature := signPayload(secret, body)

		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"workspaceId": "ws-DIFFERENT",
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
			Events:      eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("workspace org/name format match -> event emitted", func(t *testing.T) {
		body := []byte(`{"payload_version": 1, "run_id": "run-111", "workspace_id": "ws-222", "workspace_name": "prod", "organization_name": "acme", "notifications": [{"trigger": "run:needs_attention", "run_status": "policy_override"}]}`)
		signature := signPayload(secret, body)

		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"workspaceId": "acme/prod",
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
			Events:      eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("needs_attention with workspace match -> event emitted with correct data", func(t *testing.T) {
		body := []byte(`{"payload_version": 1, "run_id": "run-111", "workspace_id": "ws-222", "workspace_name": "staging", "organization_name": "myorg", "run_url": "https://app.terraform.io/runs/run-111", "run_message": "Plan requires approval", "run_created_by": "user@example.com", "notifications": [{"trigger": "run:needs_attention", "run_status": "planned"}]}`)
		signature := signPayload(secret, body)

		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"workspaceId": "ws-222",
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"webhookSecret": secret}},
			Events:      eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())

		assert.Equal(t, "terraform.needsAttention", eventContext.Payloads[0].Type)
		emittedData := eventContext.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "run-111", emittedData["runId"])
		assert.Equal(t, "ws-222", emittedData["workspaceId"])
		assert.Equal(t, "run:needs_attention", emittedData["action"])
		assert.Equal(t, "planned", emittedData["runStatus"])
		assert.Equal(t, "staging", emittedData["workspaceName"])
		assert.Equal(t, "myorg", emittedData["organizationName"])
	})
}

func Test__TerraformRunEvent__Setup(t *testing.T) {
	trigger := &RunEvent{}

	t.Run("workspaceId is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			WebhookRequests: []any{},
		}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Configuration: map[string]any{"workspaceId": "", "events": []string{"run:created"}},
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "workspaceId is required")
	})

	t.Run("valid configuration -> webhook requested", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			WebhookRequests: []any{},
		}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Configuration: map[string]any{"workspaceId": "ws-123", "events": []string{"run:created"}},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)

		webhookConfig := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, "ws-123", webhookConfig.WorkspaceID)
	})
}
