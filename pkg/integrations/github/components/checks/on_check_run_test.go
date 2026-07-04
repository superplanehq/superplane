package checks

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

func Test__OnCheckRun__HandleWebhook(t *testing.T) {
	trigger := &OnCheckRun{}

	t.Run("wrong event type is ignored", func(t *testing.T) {
		eventContext := &contexts.EventContext{}
		headers := http.Header{}
		headers.Set("X-GitHub-Event", "status")

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("matching check run is emitted", func(t *testing.T) {
		eventContext := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(signedCheckRunRequest(
			[]byte(`{
				"action": "completed",
				"check_run": {
					"name": "DCO",
					"status": "completed",
					"conclusion": "success",
					"check_suite": {"head_branch": "feature/pr-checks"},
					"pull_requests": [{"number": 42}]
				}
			}`),
			OnCheckRunConfiguration{
				Repository:       "hello",
				Statuses:         []string{"completed"},
				Conclusions:      []string{"success"},
				Names:            []configuration.Predicate{{Type: configuration.PredicateTypeEquals, Value: "DCO"}},
				Branches:         []configuration.Predicate{{Type: configuration.PredicateTypeMatches, Value: "feature/.*"}},
				PullRequestsOnly: true,
			},
			eventContext,
		))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "github.checkRun", eventContext.Payloads[0].Type)
	})

	t.Run("status filter prevents emitting", func(t *testing.T) {
		eventContext := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(signedCheckRunRequest(
			[]byte(`{"check_run":{"name":"DCO","status":"queued"}}`),
			OnCheckRunConfiguration{Repository: "hello", Statuses: []string{"completed"}},
			eventContext,
		))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("conclusion filter does not apply to in-progress checks", func(t *testing.T) {
		eventContext := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(signedCheckRunRequest(
			[]byte(`{"check_run":{"name":"DCO","status":"in_progress"}}`),
			OnCheckRunConfiguration{Repository: "hello", Conclusions: []string{"success"}},
			eventContext,
		))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "github.checkRun", eventContext.Payloads[0].Type)
	})

	t.Run("conclusion filter prevents emitting completed checks with another conclusion", func(t *testing.T) {
		eventContext := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(signedCheckRunRequest(
			[]byte(`{"check_run":{"name":"DCO","status":"completed","conclusion":"failure"}}`),
			OnCheckRunConfiguration{Repository: "hello", Conclusions: []string{"success"}},
			eventContext,
		))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("pull request filter requires pull request metadata", func(t *testing.T) {
		eventContext := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(signedCheckRunRequest(
			[]byte(`{"check_run":{"name":"DCO","status":"completed","conclusion":"success"}}`),
			OnCheckRunConfiguration{Repository: "hello", PullRequestsOnly: true},
			eventContext,
		))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})
}

func Test__OnCheckRun__Setup(t *testing.T) {
	trigger := OnCheckRun{}
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
	webhookRequest := integrationCtx.WebhookRequests[0].(common.WebhookConfiguration)
	assert.Equal(t, "check_run", webhookRequest.EventType)
	assert.Equal(t, "hello", webhookRequest.Repository)
}

func signedCheckRunRequest(body []byte, config OnCheckRunConfiguration, eventContext *contexts.EventContext) core.WebhookRequestContext {
	secret := "test-secret"
	signature := signWebhookBody(secret, body)

	headers := http.Header{}
	headers.Set("X-Hub-Signature-256", "sha256="+signature)
	headers.Set("X-GitHub-Event", "check_run")

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
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return fmt.Sprintf("%x", mac.Sum(nil))
}
