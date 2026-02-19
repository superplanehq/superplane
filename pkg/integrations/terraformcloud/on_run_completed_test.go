package terraformcloud

import (
	"crypto/hmac"
	"crypto/sha512"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func signBody(secret string, body []byte) string {
	h := hmac.New(sha512.New, []byte(secret))
	h.Write(body)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func Test__OnRunCompleted__HandleWebhook(t *testing.T) {
	trigger := &OnRunCompleted{}

	t.Run("no signature -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "missing signature")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Tfe-Notification-Signature", "invalidsignature")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"notifications":[{"trigger":"run:completed","run_status":"applied"}]}`),
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("valid signature with run:completed event -> event is emitted", func(t *testing.T) {
		body := []byte(`{"run_id":"run-123","workspace_id":"ws-456","workspace_name":"my-workspace","organization_name":"my-org","notifications":[{"trigger":"run:completed","run_status":"applied","message":"Run applied"}]}`)

		secret := "test-secret"
		signature := signBody(secret, body)

		headers := http.Header{}
		headers.Set("X-Tfe-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		require.Len(t, eventContext.Payloads, 1)
		assert.Equal(t, "terraformcloud.run.completed", eventContext.Payloads[0].Type)
	})

	t.Run("valid signature with run:errored event -> event is emitted", func(t *testing.T) {
		body := []byte(`{"run_id":"run-789","workspace_id":"ws-456","workspace_name":"my-workspace","organization_name":"my-org","notifications":[{"trigger":"run:errored","run_status":"errored","message":"Run errored"}]}`)

		secret := "test-secret"
		signature := signBody(secret, body)

		headers := http.Header{}
		headers.Set("X-Tfe-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
		require.Len(t, eventContext.Payloads, 1)
		assert.Equal(t, "terraformcloud.run.completed", eventContext.Payloads[0].Type)
	})

	t.Run("valid signature with non-matching trigger -> no event emitted", func(t *testing.T) {
		body := []byte(`{"run_id":"run-123","notifications":[{"trigger":"run:needs_attention","run_status":"policy_checked","message":"Needs attention"}]}`)

		secret := "test-secret"
		signature := signBody(secret, body)

		headers := http.Header{}
		headers.Set("X-Tfe-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("invalid JSON body -> 400", func(t *testing.T) {
		body := []byte(`invalid json`)

		secret := "test-secret"
		signature := signBody(secret, body)

		headers := http.Header{}
		headers.Set("X-Tfe-Notification-Signature", signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})
}

func Test__OnRunCompleted__Setup(t *testing.T) {
	t.Run("missing organization -> error", func(t *testing.T) {
		trigger := &OnRunCompleted{}
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"workspaceId": "ws-123",
			},
			Metadata: &contexts.MetadataContext{Metadata: map[string]any{}},
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "organization is required")
	})

	t.Run("missing workspace -> error", func(t *testing.T) {
		trigger := &OnRunCompleted{}
		err := trigger.Setup(core.TriggerContext{
			Configuration: map[string]any{
				"organization": "my-org",
			},
			Metadata: &contexts.MetadataContext{Metadata: map[string]any{}},
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "workspace is required")
	})
}
