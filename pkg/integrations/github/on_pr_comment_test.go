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
	eventType := "issue_comment"

	t.Run("no X-Hub-Signature-256 -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-GitHub-Event", eventType)
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{Headers: headers})

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
		headers := signedHeaders([]byte(`{"action":"created"}`), "wrong", eventType)
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"action":"created"}`),
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Events:  &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("issue_comment for PR with created action -> event is emitted", func(t *testing.T) {
		body := []byte(`{"action":"created","issue":{"pull_request":{"url":"https://api.github.com/repos/test/test/pulls/1"},"number":1},"comment":{"body":"comment on PR conversation"}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("issue_comment without pull_request -> event is NOT emitted", func(t *testing.T) {
		body := []byte(`{"action":"created","issue":{"id":123},"comment":{"body":"comment on issue"}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("non-created action -> event is not emitted", func(t *testing.T) {
		body := []byte(`{"action":"edited","issue":{"pull_request":{"url":"https://api.github.com/repos/test/test/pulls/1"}},"comment":{"body":"edited"}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("content filter matches -> event is emitted", func(t *testing.T) {
		body := []byte(`{"action":"created","issue":{"pull_request":{"url":"https://api.github.com/repos/test/test/pulls/1"}},"comment":{"body":"/solve this PR"}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository":    "test",
				"contentFilter": "/solve",
			},
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("content filter does not match -> event is not emitted", func(t *testing.T) {
		body := []byte(`{"action":"created","issue":{"pull_request":{"url":"https://api.github.com/repos/test/test/pulls/1"}},"comment":{"body":"regular comment"}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository":    "test",
				"contentFilter": "/solve",
			},
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("pull_request_review_comment event type -> ignored", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"some comment"}}`)
		headers := signedHeaders(body, "test-secret", "pull_request_review_comment")

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})
}

func Test__OnPRComment__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	trigger := OnPRComment{}

	t.Run("repository is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": ""},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("repository is not accessible", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{Repositories: []Repository{helloRepo}},
		}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "world"},
		})

		require.ErrorContains(t, err, "repository world is not accessible to app installation")
	})

	t.Run("metadata is set and webhook is requested for issue_comment", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{Repositories: []Repository{helloRepo}},
		}

		nodeMetadataCtx := contexts.MetadataContext{}
		require.NoError(t, trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &nodeMetadataCtx,
			Configuration: map[string]any{"repository": "hello"},
		}))

		require.Equal(t, nodeMetadataCtx.Get(), NodeMetadata{Repository: &helloRepo})
		require.Len(t, integrationCtx.WebhookRequests, 1)

		webhookRequest := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, "hello", webhookRequest.Repository)
		assert.Equal(t, "issue_comment", webhookRequest.EventType)
		assert.Empty(t, webhookRequest.EventTypes)
	})
}

func signedHeaders(body []byte, secret, eventType string) http.Header {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	signature := fmt.Sprintf("%x", h.Sum(nil))

	headers := http.Header{}
	headers.Set("X-Hub-Signature-256", "sha256="+signature)
	headers.Set("X-GitHub-Event", eventType)
	return headers
}
