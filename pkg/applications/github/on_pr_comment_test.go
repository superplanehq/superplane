package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnPRComment__HandleWebhook(t *testing.T) {
	trigger := &OnPRComment{}
	eventType := "pull_request_review_comment"

	t.Run("no X-Hub-Signature-256 -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-GitHub-Event", eventType)
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("no X-GitHub-Event -> 400", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Events:  &contexts.EventContext{},
			Webhook: &contexts.WebhookContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing X-GitHub-Event header")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")
		headers.Set("X-GitHub-Event", eventType)

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"action":"created"}`),
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("created action -> event is emitted", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"some comment"}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", eventType)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("non-created action -> event is not emitted", func(t *testing.T) {
		body := []byte(`{"action":"deleted"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", eventType)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 0)
	})

	t.Run("content filter matches -> event is emitted", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"/solve this issue"}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", eventType)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository":    "test",
				"contentFilter": "/solve",
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("content filter does not match -> event is not emitted", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"regular comment"}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", eventType)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository":    "test",
				"contentFilter": "/solve",
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 0)
	})

	t.Run("issue_comment event with pull_request -> event IS emitted", func(t *testing.T) {
		// PR conversation comments come as issue_comment events with a pull_request field
		body := []byte(`{"action":"created","issue":{"pull_request":{"url":"https://api.github.com/repos/test/test/pulls/1"}},"comment":{"body":"comment on PR conversation"}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "issue_comment")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("issue_comment event without pull_request -> event is NOT emitted", func(t *testing.T) {
		// Regular issue comments should not trigger this handler
		body := []byte(`{"action":"created","issue":{"id":123},"comment":{"body":"comment on issue"}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "issue_comment")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("issue_comment event with content filter matches -> event IS emitted", func(t *testing.T) {
		body := []byte(`{"action":"created","issue":{"pull_request":{"url":"https://api.github.com/repos/test/test/pulls/1"}},"comment":{"body":"/solve this PR"}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "issue_comment")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository":    "test",
				"contentFilter": "/solve",
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("pull_request_review submitted -> event IS emitted", func(t *testing.T) {
		// Review submission comments (when clicking "Submit review")
		body := []byte(`{"action":"submitted","review":{"body":"LGTM!","state":"approved"},"pull_request":{"number":1}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "pull_request_review")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("pull_request_review dismissed -> event is NOT emitted", func(t *testing.T) {
		body := []byte(`{"action":"dismissed","review":{"body":"dismissed review"},"pull_request":{"number":1}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "pull_request_review")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("pull_request_review with content filter matches -> event IS emitted", func(t *testing.T) {
		body := []byte(`{"action":"submitted","review":{"body":"/deploy to staging","state":"approved"},"pull_request":{"number":1}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "pull_request_review")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository":    "test",
				"contentFilter": "/deploy",
			},
			Webhook: &contexts.WebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})
}

func Test__OnPRComment__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	trigger := OnPRComment{}

	t.Run("repository is required", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := trigger.Setup(core.TriggerContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration:   map[string]any{"repository": ""},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("repository is not accessible", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}
		err := trigger.Setup(core.TriggerContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration:   map[string]any{"repository": "world"},
		})

		require.ErrorContains(t, err, "repository world is not accessible to app installation")
	})

	t.Run("metadata is set and webhook is requested with all event types", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}

		nodeMetadataCtx := contexts.MetadataContext{}
		require.NoError(t, trigger.Setup(core.TriggerContext{
			AppInstallation: appCtx,
			Metadata:        &nodeMetadataCtx,
			Configuration:   map[string]any{"repository": "hello"},
		}))

		require.Equal(t, nodeMetadataCtx.Get(), NodeMetadata{Repository: &helloRepo})
		require.Len(t, appCtx.WebhookRequests, 1)

		// Single webhook listening to all PR comment event types
		webhookRequest := appCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, "hello", webhookRequest.Repository)
		assert.ElementsMatch(t, []string{"pull_request_review_comment", "issue_comment", "pull_request_review"}, webhookRequest.EventTypes)
	})
}
