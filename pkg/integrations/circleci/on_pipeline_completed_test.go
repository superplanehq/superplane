package circleci

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnPipelineCompleted__HandleWebhook(t *testing.T) {
	trigger := &OnPipelineCompleted{}

	t.Run("no circleci-signature -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "missing signature")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("circleci-signature", "invalidsignature")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"type":"workflow-completed","workflow":{"status":"success"}}`),
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("valid signature with workflow-completed event -> event is emitted", func(t *testing.T) {
		body := []byte(`{"type":"workflow-completed","workflow":{"id":"wf-123","status":"success"},"pipeline":{"id":"pipe-123"}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("circleci-signature", signature)

		eventContext := &contexts.EventContext{}
		requestContext := &contexts.RequestContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
			Requests: requestContext,
		})

		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		// The webhook schedules a poll action instead of emitting immediately
		assert.Equal(t, "poll", requestContext.Action)
		assert.Equal(t, "pipe-123", requestContext.Params["pipelineId"])
		assert.Equal(t, 3*time.Second, requestContext.Duration)
	})

	t.Run("valid signature with non-workflow-completed event -> no event emitted", func(t *testing.T) {
		body := []byte(`{"type":"job-completed","job":{"id":"job-123","status":"success"}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("circleci-signature", signature)

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
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("circleci-signature", signature)

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
