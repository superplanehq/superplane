package pulls

import (
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func Test__OnPRReview__HandleWebhook(t *testing.T) {
	trigger := &OnPRReview{}
	eventType := "pull_request_review"
	reviewBody := `{
		"action":"submitted",
		"review":{"id":987,"body":"Please fix this @superplaneagent","user":{"login":"jules","type":"User"}},
		"pull_request":{"number":42,"title":"Add widget","html_url":"https://github.com/testhq/hello/pull/42"},
		"repository":{"full_name":"testhq/hello"}
	}`

	t.Run("no X-Hub-Signature-256 -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-GitHub-Event", eventType)
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{Headers: headers, Logger: logrus.NewEntry(logrus.New())})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("no X-GitHub-Event -> 400", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Events:  &contexts.EventContext{},
			Webhook: &contexts.NodeWebhookContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing X-GitHub-Event header")
	})

	t.Run("submitted review with mention in summary -> one event", func(t *testing.T) {
		body := []byte(reviewBody)
		headers := signedHeaders(body, "test-secret", eventType)
		events := &contexts.EventContext{}
		httpCtx := reviewCommentsHTTPContext(`[{"id":1,"body":"nit"}]`)

		code, _, err := trigger.HandleWebhook(reviewWebhookContext(body, headers, events, httpCtx, map[string]any{
			"repository":    "hello",
			"contentFilter": "@superplaneagent",
		}))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, httpCtx.Requests[0].URL.Path, "/pulls/42/reviews/987/comments")
	})

	t.Run("submitted review with mention in inline comment -> one event", func(t *testing.T) {
		body := []byte(`{
			"action":"submitted",
			"review":{"id":987,"body":"Looks good","user":{"login":"jules","type":"User"}},
			"pull_request":{"number":42,"title":"Add widget"},
			"repository":{"full_name":"testhq/hello"}
		}`)
		headers := signedHeaders(body, "test-secret", eventType)
		events := &contexts.EventContext{}
		httpCtx := reviewCommentsHTTPContext(
			`[{"id":1,"body":"Please @superplaneagent rename this"},{"id":2,"body":"optional nit"}]`,
		)

		code, _, err := trigger.HandleWebhook(reviewWebhookContext(body, headers, events, httpCtx, map[string]any{
			"repository":    "hello",
			"contentFilter": "@superplaneagent",
		}))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("five inline comments still emit one event", func(t *testing.T) {
		body := []byte(reviewBody)
		headers := signedHeaders(body, "test-secret", eventType)
		events := &contexts.EventContext{}
		httpCtx := reviewCommentsHTTPContext(`[
			{"id":1,"body":"one"},
			{"id":2,"body":"two"},
			{"id":3,"body":"three"},
			{"id":4,"body":"four"},
			{"id":5,"body":"five"}
		]`)

		code, _, err := trigger.HandleWebhook(reviewWebhookContext(body, headers, events, httpCtx, map[string]any{
			"repository": "hello",
		}))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("paginated review comments are all fetched", func(t *testing.T) {
		body := []byte(`{
			"action":"submitted",
			"review":{"id":987,"body":"","user":{"login":"jules","type":"User"}},
			"pull_request":{"number":42,"title":"Add widget"},
			"repository":{"full_name":"testhq/hello"}
		}`)
		headers := signedHeaders(body, "test-secret", eventType)
		events := &contexts.EventContext{}

		page1 := mocks.GitHubResponse(http.StatusOK, `[{"id":1,"body":"page one"}]`)
		page1.Header.Set("Link", `<https://api.github.com/repos/testhq/hello/pulls/42/reviews/987/comments?page=2>; rel="next"`)
		page2 := mocks.GitHubResponse(http.StatusOK, `[{"id":2,"body":"@superplaneagent page two"}]`)
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{page1, page2}}

		code, _, err := trigger.HandleWebhook(reviewWebhookContext(body, headers, events, httpCtx, map[string]any{
			"repository":    "hello",
			"contentFilter": "@superplaneagent",
		}))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
		require.Len(t, httpCtx.Requests, 2)
	})

	t.Run("near mention does not match", func(t *testing.T) {
		body := []byte(`{
			"action":"submitted",
			"review":{"id":987,"body":"Hey @superplaneagent-old","user":{"login":"jules","type":"User"}},
			"pull_request":{"number":42,"title":"Add widget"},
			"repository":{"full_name":"testhq/hello"}
		}`)
		headers := signedHeaders(body, "test-secret", eventType)
		events := &contexts.EventContext{}
		httpCtx := reviewCommentsHTTPContext(`[{"id":1,"body":"no mention"}]`)

		code, _, err := trigger.HandleWebhook(reviewWebhookContext(body, headers, events, httpCtx, map[string]any{
			"repository":    "hello",
			"contentFilter": "@superplaneagent",
		}))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("bot review is ignored", func(t *testing.T) {
		body := []byte(`{
			"action":"submitted",
			"review":{"id":987,"body":"@superplaneagent please","user":{"login":"codecov","type":"Bot"}},
			"pull_request":{"number":42,"title":"Add widget"},
			"repository":{"full_name":"testhq/hello"}
		}`)
		headers := signedHeaders(body, "test-secret", eventType)
		events := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(reviewWebhookContext(body, headers, events, &contexts.HTTPContext{}, map[string]any{
			"repository":    "hello",
			"contentFilter": "@superplaneagent",
			"ignoreBots":    true,
		}))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("comment loading failure is retriable", func(t *testing.T) {
		body := []byte(reviewBody)
		headers := signedHeaders(body, "test-secret", eventType)
		events := &contexts.EventContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{mocks.GitHubResponse(http.StatusInternalServerError, `{"message":"boom"}`)},
		}

		code, _, err := trigger.HandleWebhook(reviewWebhookContext(body, headers, events, httpCtx, map[string]any{
			"repository": "hello",
		}))

		assert.Equal(t, http.StatusInternalServerError, code)
		assert.Error(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("dismissed review is ignored", func(t *testing.T) {
		body := []byte(`{"action":"dismissed","review":{"id":987,"body":"nope"},"pull_request":{"number":42}}`)
		headers := signedHeaders(body, "test-secret", eventType)
		events := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Logger:        logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{"repository": "hello"},
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:        events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("pull_request_review_comment event type -> ignored", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"@superplaneagent"}}`)
		headers := signedHeaders(body, "test-secret", "pull_request_review_comment")
		events := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Logger:        logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{"repository": "hello"},
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:        events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})
}

func Test__OnPRReview__Setup(t *testing.T) {
	trigger := OnPRReview{}

	t.Run("webhook is requested for pull_request_review", func(t *testing.T) {
		integrationCtx := mocks.IntegrationContextForNewSetupFlow()
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"id": 123456,
					"name": "hello",
					"html_url": "https://github.com/testhq/hello"
				}`),
			},
		}

		require.NoError(t, trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			HTTP:          httpCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "hello"},
		}))

		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookRequest := integrationCtx.WebhookRequests[0].(common.WebhookConfiguration)
		assert.Equal(t, "hello", webhookRequest.Repository)
		assert.Equal(t, "pull_request_review", webhookRequest.EventType)
	})
}

func Test__ContentFilterMatchesMention(t *testing.T) {
	t.Run("exact mention matches", func(t *testing.T) {
		matched, err := contentFilterMatches("@superplaneagent", "please @superplaneagent fix this")
		require.NoError(t, err)
		assert.True(t, matched)
	})

	t.Run("near mention does not match", func(t *testing.T) {
		matched, err := contentFilterMatches("@superplaneagent", "please @superplaneagent-old fix this")
		require.NoError(t, err)
		assert.False(t, matched)
	})

	t.Run("mention match is case insensitive", func(t *testing.T) {
		matched, err := contentFilterMatches("@SuperPlaneAgent", "Ping @superplaneagent")
		require.NoError(t, err)
		assert.True(t, matched)
	})

	t.Run("regex filter still matches substrings", func(t *testing.T) {
		matched, err := contentFilterMatches("/deploy", "please /deploy now")
		require.NoError(t, err)
		assert.True(t, matched)
	})
}

func reviewWebhookContext(
	body []byte,
	headers http.Header,
	events *contexts.EventContext,
	httpCtx *contexts.HTTPContext,
	configuration map[string]any,
) core.WebhookRequestContext {
	return core.WebhookRequestContext{
		Body:          body,
		Headers:       headers,
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: configuration,
		Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
		Events:        events,
		HTTP:          httpCtx,
		Integration:   mocks.IntegrationContextForNewSetupFlow(),
	}
}

func reviewCommentsHTTPContext(body string) *contexts.HTTPContext {
	return &contexts.HTTPContext{
		Responses: []*http.Response{mocks.GitHubResponse(http.StatusOK, body)},
	}
}
