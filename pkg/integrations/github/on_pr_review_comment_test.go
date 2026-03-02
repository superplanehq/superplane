package github

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnPRReviewComment__HandleWebhook(t *testing.T) {
	trigger := &OnPRReviewComment{}
	eventType := "pull_request_review_comment"

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
			Webhook: &contexts.NodeWebhookContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing X-GitHub-Event header")
	})

	t.Run("pull_request_review_comment created action -> event is emitted", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"some review comment"},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("pull_request_review_comment non-created action -> event is not emitted", func(t *testing.T) {
		body := []byte(`{"action":"edited","comment":{"body":"some review comment"},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("pull_request_review submitted -> event is emitted", func(t *testing.T) {
		body := []byte(`{"action":"submitted","review":{"body":"LGTM"},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", "pull_request_review")

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("pull_request_review dismissed -> event is not emitted", func(t *testing.T) {
		body := []byte(`{"action":"dismissed","review":{"body":"dismissed"},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", "pull_request_review")

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("content filter matches review comment -> event is emitted", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"/deploy please"},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository":    "test",
				"contentFilter": "/deploy",
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("content filter matches review submission -> event is emitted", func(t *testing.T) {
		body := []byte(`{"action":"submitted","review":{"body":"/deploy now"},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", "pull_request_review")

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository":    "test",
				"contentFilter": "/deploy",
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("issue_comment event type -> ignored", func(t *testing.T) {
		body := []byte(`{"action":"created","issue":{"pull_request":{"url":"https://api.github.com/repos/test/test/pulls/1"}},"comment":{"body":"comment on PR conversation"}}`)
		headers := signedHeaders(body, "test-secret", "issue_comment")

		events := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Configuration: map[string]any{
				"repository": "test",
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})
}

func Test__OnPRReviewComment__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	trigger := OnPRReviewComment{}

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
		integrationCtx := &contexts.IntegrationContext{Metadata: Metadata{Repositories: []Repository{helloRepo}}}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "world"},
		})

		require.ErrorContains(t, err, "repository world is not accessible to app installation")
	})

	t.Run("metadata is set and webhook is requested with review event types", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{Metadata: Metadata{Repositories: []Repository{helloRepo}}}
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
		assert.ElementsMatch(t, []string{"pull_request_review_comment", "pull_request_review"}, webhookRequest.EventTypes)
		assert.Empty(t, webhookRequest.EventType)
	})
}
