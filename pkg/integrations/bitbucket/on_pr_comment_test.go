package bitbucket

import (
	"io"
	"net/http"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnPRComment__Setup(t *testing.T) {
	trigger := &OnPRComment{}
	metadata := Metadata{
		AuthType:  AuthTypeWorkspaceAccessToken,
		Workspace: &WorkspaceMetadata{Slug: "superplane"},
	}

	integrationContext := func() *contexts.IntegrationContext {
		return &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "token"},
			Metadata:      metadata,
		}
	}

	t.Run("at least one action is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: integrationContext(),
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"actions":    []string{},
			},
		})

		require.ErrorContains(t, err, "at least one action is required")
	})

	t.Run("unsupported action is rejected", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: integrationContext(),
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"actions":    []string{"comment_exploded"},
			},
		})

		require.ErrorContains(t, err, `unsupported comment action "comment_exploded"`)
	})

	t.Run("invalid content filter is rejected", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: integrationContext(),
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository":    "hello",
				"actions":       []string{"comment_created"},
				"contentFilter": "[unclosed",
			},
		})

		require.ErrorContains(t, err, "invalid content filter pattern")
	})

	t.Run("webhook is requested for every selected action", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"values":[{"uuid":"{hello}","name":"hello","full_name":"superplane/hello","slug":"hello"}]}`,
					)),
				},
			},
		}
		integrationCtx := integrationContext()

		require.NoError(t, trigger.Setup(core.TriggerContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"actions":    []string{"comment_created", "comment_updated"},
			},
		}))

		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookRequest, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"pullrequest:comment_created", "pullrequest:comment_updated"}, webhookRequest.EventTypes)
	})
}

func Test__OnPRComment__HandleWebhook(t *testing.T) {
	trigger := &OnPRComment{}

	handle := func(body []byte, config map[string]any) (int, *contexts.EventContext, error) {
		headers := http.Header{}
		headers.Set("X-Event-Key", "pullrequest:comment_created")
		headers.Set("X-Hub-Signature", "sha256="+signBitbucketPayload("test-secret", body))

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Logger:        log.NewEntry(log.New()),
			Body:          body,
			Headers:       headers,
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: config,
			Events:        eventContext,
		})

		return code, eventContext, err
	}

	baseConfig := map[string]any{
		"repository": "hello",
		"actions":    []string{"comment_created"},
	}

	t.Run("comment matching the filter -> event is emitted", func(t *testing.T) {
		config := map[string]any{
			"repository":    "hello",
			"actions":       []string{"comment_created"},
			"contentFilter": "^/deploy",
		}

		code, eventContext, err := handle([]byte(`{"comment":{"content":{"raw":"/deploy staging"}}}`), config)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "bitbucket.pullRequestComment", eventContext.Payloads[0].Type)
	})

	t.Run("comment not matching the filter -> event is not emitted", func(t *testing.T) {
		config := map[string]any{
			"repository":    "hello",
			"actions":       []string{"comment_created"},
			"contentFilter": "^/deploy",
		}

		code, eventContext, err := handle([]byte(`{"comment":{"content":{"raw":"looks good to me"}}}`), config)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("no filter -> every comment is emitted", func(t *testing.T) {
		code, eventContext, err := handle([]byte(`{"comment":{"content":{"raw":"looks good to me"}}}`), baseConfig)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("comment without a body never matches a filter", func(t *testing.T) {
		config := map[string]any{
			"repository":    "hello",
			"actions":       []string{"comment_created"},
			"contentFilter": "^/deploy",
		}

		code, eventContext, err := handle([]byte(`{"comment":{}}`), config)

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("deleted comment event is dropped when not selected", func(t *testing.T) {
		body := []byte(`{"comment":{"content":{"raw":"gone"}}}`)
		headers := http.Header{}
		headers.Set("X-Event-Key", "pullrequest:comment_deleted")
		headers.Set("X-Hub-Signature", "sha256="+signBitbucketPayload("test-secret", body))

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Logger:        log.NewEntry(log.New()),
			Body:          body,
			Headers:       headers,
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: baseConfig,
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})
}
