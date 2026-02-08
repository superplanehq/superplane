package circleci

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__extractV1Signature(t *testing.T) {
	assert.Equal(t, "abc", extractV1Signature("v1=abc"))
	assert.Equal(t, "bar", extractV1Signature("v2=foo,v1=bar"))
	assert.Equal(t, "bar", extractV1Signature(" v1=bar , v2=foo "))
	assert.Equal(t, "", extractV1Signature("v2=foo"))
	assert.Equal(t, "", extractV1Signature(""))
}

func Test__OnWorkflowCompleted__HandleWebhook(t *testing.T) {
	trigger := &OnWorkflowCompleted{}

	t.Run("missing signature -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
		})
		assert.Equal(t, http.StatusForbidden, code)
		assert.Error(t, err)
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("circleci-signature", "v1=deadbeef")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"type":"workflow-completed"}`),
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Events:  &contexts.EventContext{},
		})
		assert.Equal(t, http.StatusForbidden, code)
		assert.Error(t, err)
	})

	t.Run("valid signature -> emits event", func(t *testing.T) {
		body := []byte(`{"type":"workflow-completed","workflow":{"id":"w1"}}`)
		secret := []byte("test-secret")

		mac := hmac.New(sha256.New, secret)
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))

		headers := http.Header{}
		headers.Set("circleci-signature", "v1="+sig)

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: string(secret)},
			Events:  events,
		})
		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
		assert.Equal(t, "circleci.workflow.completed", events.Payloads[0].Type)
	})
}
