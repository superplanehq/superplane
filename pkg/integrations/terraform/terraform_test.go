package terraform

import (
	"crypto/hmac"
	"crypto/sha512"
	"fmt"
	"net/http"
	"net/http/httptest"
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
			Integration: &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{"webhookSecret": {Name: "webhookSecret", Value: []byte(secret)}}},
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
			Integration: &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{"webhookSecret": {Name: "webhookSecret", Value: []byte(secret)}}},
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
			Integration: &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{"webhookSecret": {Name: "webhookSecret", Value: []byte(secret)}}},
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
		assert.ErrorContains(t, err, "failed to get webhook secret or none configured")
	})

	t.Run("verification handshake -> 200 with nil payload", func(t *testing.T) {
		body := []byte(`{"payload_version": 1, "notifications": [{"trigger": "verification"}]}`)
		signature := signPayload(secret, body)

		headers := http.Header{}
		headers.Set("X-TFE-Notification-Signature", signature)

		payload, code, err := ParseAndValidateWebhook(core.WebhookRequestContext{
			Headers:     headers,
			Body:        body,
			Integration: &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{"webhookSecret": {Name: "webhookSecret", Value: []byte(secret)}}},
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
			Integration: &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{"webhookSecret": {Name: "webhookSecret", Value: []byte(secret)}}},
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
			Integration: &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{"webhookSecret": {Name: "webhookSecret", Value: []byte(secret)}}},
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
			Integration: &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{"webhookSecret": {Name: "webhookSecret", Value: []byte(secret)}}},
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
			Integration: &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{"webhookSecret": {Name: "webhookSecret", Value: []byte(secret)}}},
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
			Integration: &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{"webhookSecret": {Name: "webhookSecret", Value: []byte(secret)}}},
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
			Integration: &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{"webhookSecret": {Name: "webhookSecret", Value: []byte(secret)}}},
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
			Integration: &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{"webhookSecret": {Name: "webhookSecret", Value: []byte(secret)}}},
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

	t.Run("SuperPlane-initiated run (gear emoji) -> event emitted by default", func(t *testing.T) {
		body := []byte(`{"payload_version": 1, "run_id": "run-111", "run_message": "⚙ Triggered by SuperPlane", "workspace_id": "ws-222", "workspace_name": "my-ws", "organization_name": "my-org", "notifications": [{"trigger": "run:created", "run_status": "pending"}]}`)
		signature := signPayload(secret, body)

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
			Integration: &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{"webhookSecret": {Name: "webhookSecret", Value: []byte(secret)}}},
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
			Integration: &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{"webhookSecret": {Name: "webhookSecret", Value: []byte(secret)}}},
			Events:      eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "terraform.runEvent", eventContext.Payloads[0].Type)
	})
}

func Test__TerraformRunEvent__Setup(t *testing.T) {
	trigger := &RunEvent{}

	t.Run("workspaceId is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			WebhookRequests: []any{},
			Configuration:   map[string]any{"apiToken": "test-token"},
		}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"workspaceId": "", "events": []string{"run:created"}},
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "workspaceId is required")
	})

	t.Run("valid configuration -> webhook requested", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v2/workspaces/ws-123", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data": {"id": "ws-123", "attributes": {"name": "test-workspace"}}}`))
		}))
		defer ts.Close()

		integrationCtx := &contexts.IntegrationContext{
			WebhookRequests: []any{},
			Configuration: map[string]any{
				"apiToken": "test-token",
				"address":  ts.URL,
			},
		}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"workspaceId": "ws-123", "events": []string{"run:created"}},
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)

		webhookConfig := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, "ws-123", webhookConfig.WorkspaceID)
	})
}

func Test__TerraformPlan__Cancel(t *testing.T) {
	plan := &Plan{}

	t.Run("successfully cancels the running plan via API", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v2/runs/run-123/actions/cancel", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		metadata := ExecutionMetadata{RunID: "run-123"}

		metaCtx := &contexts.MetadataContext{}
		_ = metaCtx.Set(metadata)

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
				"address":  ts.URL,
			},
		}

		err := plan.Cancel(core.ExecutionContext{
			Integration: integrationCtx,
			Metadata:    metaCtx,
		})

		require.NoError(t, err)
	})

	t.Run("returns error when API fails to cancel", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
		}))
		defer ts.Close()

		metadata := ExecutionMetadata{RunID: "run-invalid"}

		metaCtx := &contexts.MetadataContext{}
		_ = metaCtx.Set(metadata)

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
				"address":  ts.URL,
			},
		}

		err := plan.Cancel(core.ExecutionContext{
			Integration: integrationCtx,
			Metadata:    metaCtx,
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to cancel terraform run")
	})
}
