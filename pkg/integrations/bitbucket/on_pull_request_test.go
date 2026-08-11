package bitbucket

import (
	"io"
	"net/http"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnPullRequest__Setup(t *testing.T) {
	trigger := &OnPullRequest{}
	metadata := Metadata{
		AuthType: AuthTypeWorkspaceAccessToken,
		Workspace: &WorkspaceMetadata{
			Slug: "superplane",
		},
	}

	t.Run("at least one action is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"token": "token"}, Metadata: metadata},
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
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"token": "token"}, Metadata: metadata},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"actions":    []string{"exploded"},
			},
		})

		require.ErrorContains(t, err, `unsupported pull request action "exploded"`)
	})

	t.Run("repository is required", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"token": "token"}, Metadata: metadata},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "",
				"actions":    []string{"created"},
			},
		})

		require.ErrorContains(t, err, "repository is required")
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
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "token"},
			Metadata:      metadata,
		}

		require.NoError(t, trigger.Setup(core.TriggerContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"actions":    []string{"created", "fulfilled"},
			},
		}))

		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookRequest, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"pullrequest:created", "pullrequest:fulfilled"}, webhookRequest.EventTypes)
		assert.Equal(t, "hello", webhookRequest.RepositorySlug)
	})
}

func Test__OnPullRequest__HandleWebhook(t *testing.T) {
	trigger := &OnPullRequest{}

	configurationFor := func(actions []string, targetBranches []configuration.Predicate) map[string]any {
		config := map[string]any{
			"repository": "hello",
			"actions":    actions,
		}

		if targetBranches != nil {
			config["targetBranches"] = targetBranches
		}

		return config
	}

	t.Run("no X-Event-Key -> 400", func(t *testing.T) {
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Logger:        log.NewEntry(log.New()),
			Headers:       http.Header{},
			Configuration: configurationFor([]string{"created"}, nil),
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing X-Event-Key header")
	})

	// Webhooks are shared across triggers on the same repository, so a push delivery
	// reaching this trigger must be dropped without touching the signature or payload.
	t.Run("event belongs to another trigger -> 200", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Event-Key", "repo:push")

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Logger:        log.NewEntry(log.New()),
			Headers:       headers,
			Events:        eventContext,
			Configuration: configurationFor([]string{"created"}, nil),
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("action is not selected -> 200", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Event-Key", "pullrequest:rejected")

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Logger:        log.NewEntry(log.New()),
			Headers:       headers,
			Events:        eventContext,
			Configuration: configurationFor([]string{"created", "fulfilled"}, nil),
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Event-Key", "pullrequest:created")
		headers.Set("X-Hub-Signature", "sha256=invalid")

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Logger:        log.NewEntry(log.New()),
			Body:          []byte(`{}`),
			Headers:       headers,
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: configurationFor([]string{"created"}, nil),
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("invalid body -> 400", func(t *testing.T) {
		body := []byte("{")
		headers := http.Header{}
		headers.Set("X-Event-Key", "pullrequest:created")
		headers.Set("X-Hub-Signature", "sha256="+signBitbucketPayload("test-secret", body))

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Logger:        log.NewEntry(log.New()),
			Body:          body,
			Headers:       headers,
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: configurationFor([]string{"created"}, nil),
			Events:        &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("target branch does not match -> event is not emitted", func(t *testing.T) {
		body := []byte(`{"pullrequest":{"id":42,"destination":{"branch":{"name":"develop"}}}}`)
		headers := http.Header{}
		headers.Set("X-Event-Key", "pullrequest:created")
		headers.Set("X-Hub-Signature", "sha256="+signBitbucketPayload("test-secret", body))

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Logger:  log.NewEntry(log.New()),
			Body:    body,
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: configurationFor([]string{"created"}, []configuration.Predicate{
				{Type: configuration.PredicateTypeEquals, Value: "main"},
			}),
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("target branch matches -> event is emitted", func(t *testing.T) {
		body := []byte(`{"pullrequest":{"id":42,"destination":{"branch":{"name":"main"}}}}`)
		headers := http.Header{}
		headers.Set("X-Event-Key", "pullrequest:created")
		headers.Set("X-Hub-Signature", "sha256="+signBitbucketPayload("test-secret", body))

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Logger:  log.NewEntry(log.New()),
			Body:    body,
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: configurationFor([]string{"created"}, []configuration.Predicate{
				{Type: configuration.PredicateTypeEquals, Value: "main"},
			}),
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "bitbucket.pullRequest", eventContext.Payloads[0].Type)
	})

	t.Run("no target branch filter -> every target branch is accepted", func(t *testing.T) {
		body := []byte(`{"pullrequest":{"id":7,"destination":{"branch":{"name":"release/2026-08"}}}}`)
		headers := http.Header{}
		headers.Set("X-Event-Key", "pullrequest:fulfilled")
		headers.Set("X-Hub-Signature", "sha256="+signBitbucketPayload("test-secret", body))

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Logger:        log.NewEntry(log.New()),
			Body:          body,
			Headers:       headers,
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: configurationFor([]string{"fulfilled"}, nil),
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})
}

func Test__PullRequestTargetBranch(t *testing.T) {
	t.Run("reads the destination branch", func(t *testing.T) {
		branch := pullRequestTargetBranch(map[string]any{
			"pullrequest": map[string]any{
				"destination": map[string]any{
					"branch": map[string]any{"name": "main"},
				},
			},
		})

		assert.Equal(t, "main", branch)
	})

	t.Run("missing destination returns empty string", func(t *testing.T) {
		assert.Empty(t, pullRequestTargetBranch(map[string]any{"pullrequest": map[string]any{}}))
	})

	t.Run("missing pull request returns empty string", func(t *testing.T) {
		assert.Empty(t, pullRequestTargetBranch(map[string]any{}))
	})
}
