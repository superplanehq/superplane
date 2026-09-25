package dependabot

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
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

func Test__OnAlert__HandleWebhook(t *testing.T) {
	trigger := &OnAlert{}

	t.Run("created action emits an event", func(t *testing.T) {
		body := []byte(`{"action":"created","alert":{"number":7}}`)
		secret := "test-secret"
		eventContext := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(signedDependabotRequest(t, body, secret, map[string]any{
			"repository": "hello",
			"actions":    []string{"created", "reopened", "reintroduced"},
		}, eventContext))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, PayloadType, eventContext.Payloads[0].Type)
	})

	t.Run("fixed action is ignored", func(t *testing.T) {
		body := []byte(`{"action":"fixed"}`)
		secret := "test-secret"
		eventContext := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(signedDependabotRequest(t, body, secret, map[string]any{
			"repository": "hello",
			"actions":    []string{"created"},
		}, eventContext))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("other GitHub events are ignored", func(t *testing.T) {
		body := []byte(`{"action":"opened"}`)
		secret := "test-secret"
		headers := signedHeaders(body, secret)
		headers.Set("X-GitHub-Event", "issues")

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "hello",
				"actions":    []string{"created"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
	})
}

func Test__OnAlert__Setup(t *testing.T) {
	trigger := OnAlert{}

	t.Run("webhook is requested for dependabot alerts", func(t *testing.T) {
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
		assert.Equal(t, "dependabot_alert", webhookRequest.EventType)
		assert.Equal(t, "hello", webhookRequest.Repository)
	})
}

func signedDependabotRequest(
	t *testing.T,
	body []byte,
	secret string,
	configuration map[string]any,
	events *contexts.EventContext,
) core.WebhookRequestContext {
	t.Helper()
	return core.WebhookRequestContext{
		Body:          body,
		Headers:       signedHeaders(body, secret),
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: configuration,
		Webhook:       &contexts.NodeWebhookContext{Secret: secret},
		Events:        events,
	}
}

func signedHeaders(body []byte, secret string) http.Header {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	headers := http.Header{}
	headers.Set("X-Hub-Signature-256", "sha256="+fmt.Sprintf("%x", mac.Sum(nil)))
	headers.Set("X-GitHub-Event", "dependabot_alert")
	return headers
}
