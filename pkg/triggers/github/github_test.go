package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__HandleWebhook(t *testing.T) {
	trigger := &GitHub{}

	t.Run("no X-Hub-Signature-256 -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("no X-GitHub-Event -> 400", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:        headers,
			EventContext:   &DummyEventContext{},
			WebhookContext: &DummyWebhookContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing X-GitHub-Event header")
	})

	t.Run("event not in configuration is ignored", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")
		headers.Set("X-GitHub-Event", "pull_request")

		eventContext := &DummyEventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:        headers,
			Configuration:  Configuration{EventType: "push"},
			EventContext:   eventContext,
			WebhookContext: &DummyWebhookContext{},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")
		headers.Set("X-GitHub-Event", "push")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:           []byte(`{"ref":"refs/heads/main"}`),
			Headers:        headers,
			Configuration:  Configuration{EventType: "push"},
			WebhookContext: &DummyWebhookContext{Secret: secret},
			EventContext:   &DummyEventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("branch deletion push is ignored", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/main","deleted":true}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "push")

		eventContext := &DummyEventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:           body,
			Headers:        headers,
			Configuration:  Configuration{EventType: "push"},
			WebhookContext: &DummyWebhookContext{Secret: secret},
			EventContext:   eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/main"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "push")

		eventContext := &DummyEventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:           body,
			Headers:        headers,
			Configuration:  Configuration{EventType: "push"},
			WebhookContext: &DummyWebhookContext{Secret: secret},
			EventContext:   eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})
}

type DummyWebhookContext struct {
	Secret string
}

func (w *DummyWebhookContext) GetSecret() ([]byte, error) {
	return []byte(w.Secret), nil
}

func (w *DummyWebhookContext) Setup(options *core.WebhookSetupOptions) error {
	return nil
}

type DummyEventContext struct {
	EmittedEvents []any
}

func (e *DummyEventContext) Emit(event any) error {
	e.EmittedEvents = append(e.EmittedEvents, event)
	return nil
}

func (e *DummyEventContext) Count() int {
	return len(e.EmittedEvents)
}

func Test__IsBranchDeletionEvent(t *testing.T) {
	assert.True(t, isBranchDeletionEvent("push", map[string]any{"deleted": true}))
	assert.False(t, isBranchDeletionEvent("push", map[string]any{"deleted": false}))
	assert.False(t, isBranchDeletionEvent("push", map[string]any{}))
	assert.False(t, isBranchDeletionEvent("pull_request", map[string]any{}))
}
