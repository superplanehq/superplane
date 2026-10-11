package bitbucket

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func pullRequestCommentFixture(t *testing.T) []byte {
	t.Helper()
	body, err := os.ReadFile("example_data_on_pull_request_comment.json")
	require.NoError(t, err)
	return body
}

func Test__OnPullRequestComment__Setup(t *testing.T) {
	trigger := OnPullRequestComment{}
	metadata := Metadata{
		AuthType: AuthTypeWorkspaceAccessToken,
		Workspace: &WorkspaceMetadata{
			Slug: "acme",
		},
	}

	t.Run("repository is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "token"},
			Metadata:      metadata,
		}

		err := trigger.Setup(core.TriggerContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": ""},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("webhook requests comment events", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"values":[{"uuid":"{repo-uuid}","name":"widgets","full_name":"acme/widgets","slug":"widgets"}]}`,
					)),
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "token"},
			Metadata:      metadata,
		}

		require.NoError(t, trigger.Setup(core.TriggerContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "acme/widgets",
			},
		}))

		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookRequest, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.ElementsMatch(t, []string{"pullrequest:comment_created"}, webhookRequest.EventTypes)
		assert.Equal(t, "widgets", webhookRequest.RepositorySlug)
	})
}

func Test__OnPullRequestComment__HandleWebhook(t *testing.T) {
	trigger := &OnPullRequestComment{}

	configuration := map[string]any{
		"repository": "acme/widgets",
	}

	signed := func(t *testing.T, eventKey string, body []byte) (http.Header, *contexts.EventContext) {
		t.Helper()
		headers := http.Header{}
		headers.Set("X-Event-Key", eventKey)
		headers.Set("X-Hub-Signature", "sha256="+signBitbucketPayload("test-secret", body))
		return headers, &contexts.EventContext{}
	}

	t.Run("no X-Event-Key -> 400", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{Headers: http.Header{}})
		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing X-Event-Key header")
	})

	t.Run("comment updates do not start runs", func(t *testing.T) {
		body := pullRequestCommentFixture(t)
		headers, eventContext := signed(t, "pullrequest:comment_updated", body)

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: configuration,
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Event-Key", "pullrequest:comment_created")
		headers.Set("X-Hub-Signature", "sha256=invalid")

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    pullRequestCommentFixture(t),
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("repository mismatch -> event is not emitted", func(t *testing.T) {
		body := pullRequestCommentFixture(t)
		headers, eventContext := signed(t, "pullrequest:comment_created", body)

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Metadata: &contexts.MetadataContext{Metadata: NodeMetadata{Repository: &RepositoryMetadata{
				UUID:     "{other-uuid}",
				FullName: "acme/other",
				Slug:     "other",
				Name:     "other",
			}}},
			Configuration: configuration,
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("mention filter matches the comment body", func(t *testing.T) {
		body := pullRequestCommentFixture(t)
		headers, eventContext := signed(t, "pullrequest:comment_created", body)

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: map[string]any{
				"repository":    "acme/widgets",
				"contentFilter": "@ada",
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())

		missHeaders, missEvents := signed(t, "pullrequest:comment_created", body)
		code, _, err = trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: missHeaders,
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: map[string]any{
				"repository":    "acme/widgets",
				"contentFilter": "@grace",
			},
			Events: missEvents,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, missEvents.Count())
	})

	t.Run("created comment emits inline context", func(t *testing.T) {
		body := pullRequestCommentFixture(t)
		headers, eventContext := signed(t, "pullrequest:comment_created", body)

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: configuration,
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "bitbucket.pullRequestComment", eventContext.Payloads[0].Type)
		event, ok := eventContext.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		comment, ok := event["comment"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "@ada please check the retry backoff on line 42", comment["body"])
		assert.Equal(t, "grace", comment["nickname"])
		inline, ok := comment["inline"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "pkg/workers/retry.go", inline["path"])
		pullRequest, ok := event["pullrequest"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(42), pullRequest["id"])
	})
}

func Test__MatchBitbucketContentFilter(t *testing.T) {
	event := map[string]any{"comment": map[string]any{
		"body":     "looks good, /solve it @Ada",
		"nickname": "grace",
	}}

	assert.True(t, matchBitbucketContentFilter("", event))
	assert.True(t, matchBitbucketContentFilter("@ada", event))
	assert.True(t, matchBitbucketContentFilter("@Ada", event))
	assert.False(t, matchBitbucketContentFilter("@grace", event))
	assert.False(t, matchBitbucketContentFilter("@ad", event))
	assert.False(t, matchBitbucketContentFilter("@adalovelace", event))
	assert.True(t, matchBitbucketContentFilter("/solve", event))
	assert.False(t, matchBitbucketContentFilter("/deploy", event))
}
