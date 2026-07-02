package statuses

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func Test__OnCommitStatus__HandleWebhook(t *testing.T) {
	trigger := &OnCommitStatus{}

	t.Run("no X-Hub-Signature-256 -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-GitHub-Event", "status")

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
		})

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

	t.Run("wrong event type is ignored", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-GitHub-Event", "push")

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")
		headers.Set("X-GitHub-Event", "status")

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"state":"success","context":"ci/build"}`),
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: OnCommitStatusConfiguration{
				Repository: "test",
				States:     []string{"success"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("state matches filter -> event is emitted", func(t *testing.T) {
		eventContext := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(signedStatusRequest(
			[]byte(`{"state":"success","context":"ci/build","branches":[{"name":"main"}]}`),
			OnCommitStatusConfiguration{
				Repository: "test",
				States:     []string{"success"},
			},
			eventContext,
		))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "github.status", eventContext.Payloads[0].Type)
	})

	t.Run("state does not match filter -> event is not emitted", func(t *testing.T) {
		eventContext := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(signedStatusRequest(
			[]byte(`{"state":"failure","context":"ci/build","branches":[{"name":"main"}]}`),
			OnCommitStatusConfiguration{
				Repository: "test",
				States:     []string{"success"},
			},
			eventContext,
		))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("context matches filter -> event is emitted", func(t *testing.T) {
		eventContext := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(signedStatusRequest(
			[]byte(`{"state":"pending","context":"deploy/production","branches":[{"name":"main"}]}`),
			OnCommitStatusConfiguration{
				Repository: "test",
				States:     []string{"pending"},
				Contexts: []configuration.Predicate{
					{Type: configuration.PredicateTypeMatches, Value: "deploy/.*"},
				},
			},
			eventContext,
		))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("context does not match filter -> event is not emitted", func(t *testing.T) {
		eventContext := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(signedStatusRequest(
			[]byte(`{"state":"success","context":"ci/lint","branches":[{"name":"main"}]}`),
			OnCommitStatusConfiguration{
				Repository: "test",
				States:     []string{"success"},
				Contexts: []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "ci/build"},
				},
			},
			eventContext,
		))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("branch matches filter -> event is emitted", func(t *testing.T) {
		eventContext := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(signedStatusRequest(
			[]byte(`{"state":"success","context":"ci/build","branches":[{"name":"release/v1.2.3"},{"name":"main"}]}`),
			OnCommitStatusConfiguration{
				Repository: "test",
				States:     []string{"success"},
				Branches: []configuration.Predicate{
					{Type: configuration.PredicateTypeMatches, Value: "release/.*"},
				},
			},
			eventContext,
		))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("branch does not match filter -> event is not emitted", func(t *testing.T) {
		eventContext := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(signedStatusRequest(
			[]byte(`{"state":"success","context":"ci/build","branches":[{"name":"feature/example"}]}`),
			OnCommitStatusConfiguration{
				Repository: "test",
				States:     []string{"success"},
				Branches: []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "main"},
				},
			},
			eventContext,
		))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("missing state -> 400", func(t *testing.T) {
		eventContext := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(signedStatusRequest(
			[]byte(`{"context":"ci/build"}`),
			OnCommitStatusConfiguration{Repository: "test"},
			eventContext,
		))

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing or invalid status state")
		assert.Zero(t, eventContext.Count())
	})
}

func Test__OnCommitStatus__Setup(t *testing.T) {
	trigger := OnCommitStatus{}

	t.Run("webhook is requested", func(t *testing.T) {
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

		nodeMetadataCtx := contexts.MetadataContext{}
		require.NoError(t, trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			HTTP:          httpCtx,
			Metadata:      &nodeMetadataCtx,
			Configuration: map[string]any{"repository": "hello"},
		}))

		require.Len(t, integrationCtx.WebhookRequests, 1)
		require.Len(t, httpCtx.Requests, 1)
		webhookRequest := integrationCtx.WebhookRequests[0].(common.WebhookConfiguration)
		assert.Equal(t, "status", webhookRequest.EventType)
		assert.Equal(t, "hello", webhookRequest.Repository)
	})
}

func signedStatusRequest(body []byte, config OnCommitStatusConfiguration, eventContext *contexts.EventContext) core.WebhookRequestContext {
	secret := "test-secret"
	signature := signWebhookBody(secret, body)

	headers := http.Header{}
	headers.Set("X-Hub-Signature-256", "sha256="+signature)
	headers.Set("X-GitHub-Event", "status")

	return core.WebhookRequestContext{
		Body:          body,
		Headers:       headers,
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: config,
		Webhook:       &contexts.NodeWebhookContext{Secret: secret},
		Events:        eventContext,
	}
}

func signWebhookBody(secret string, body []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	return fmt.Sprintf("%x", h.Sum(nil))
}
