package pulls

import (
	"encoding/json"
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

func Test__OnPRReviewComment__HandleWebhook(t *testing.T) {
	trigger := &OnPRReviewComment{}
	eventType := "pull_request_review_comment"

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

	t.Run("pull_request_review_comment created action -> event is emitted", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"some review comment"},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
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

	t.Run("pull_request_review_comment deleted action -> event is not emitted", func(t *testing.T) {
		body := []byte(`{"action":"deleted","comment":{"body":"some review comment"},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
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
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
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

	for _, action := range []string{"dismissed", "edited"} {
		t.Run("pull_request_review "+action+" -> event is not emitted", func(t *testing.T) {
			body := []byte(fmt.Sprintf(`{"action":%q,"review":{"body":"@superplaneagent please"},"pull_request":{"number":1}}`, action))
			headers := signedHeaders(body, "test-secret", "pull_request_review")

			events := &contexts.EventContext{}
			code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
				Body:    body,
				Headers: headers,
				Logger:  logrus.NewEntry(logrus.New()),
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

	t.Run("content filter matches review comment -> event is emitted", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"/deploy please"},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
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
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
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

	t.Run("includeReviewSubmissions false ignores submitted reviews", func(t *testing.T) {
		body := []byte(`{"action":"submitted","review":{"body":"LGTM"},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", "pull_request_review")

		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository":               "test",
				"includeReviewSubmissions": false,
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("replies scope ignores top-level review comments", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"@superplaneagent please","user":{"type":"User"}},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository":   "test",
				"commentScope": "replies",
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("replies scope accepts mentioned replies", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"@superplaneagent please","in_reply_to_id":99,"user":{"type":"User"}},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository":    "test",
				"commentScope":  "replies",
				"contentFilter": "@superplaneagent",
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("replies scope ignores replies without the mention", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"just a follow-up","in_reply_to_id":99,"user":{"type":"User"}},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository":    "test",
				"commentScope":  "replies",
				"contentFilter": "@superplaneagent",
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("ignoreBots skips bot comments", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"@superplaneagent","user":{"type":"Bot"}},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository":    "test",
				"contentFilter": "@superplaneagent",
				"ignoreBots":    true,
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("allowed bot comment with no mention -> event is emitted", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"Consider extracting this into a helper.","user":{"login":"coderabbitai[bot]","type":"Bot"}},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository":    "test",
				"contentFilter": "@superplaneagent",
				"allowedBots":   []string{"coderabbitai"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("bot not on allowlist with no mention -> ignored", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"Consider extracting this into a helper.","user":{"login":"codecov","type":"Bot"}},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository":    "test",
				"contentFilter": "@superplaneagent",
				"allowedBots":   []string{"coderabbitai"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	for _, action := range []string{"created", "edited"} {
		t.Run(action+" allowed bot comment while ignoreBots is true -> still emitted", func(t *testing.T) {
			body := []byte(fmt.Sprintf(`{"action":%q,"comment":{"body":"Consider extracting this into a helper.","user":{"login":"coderabbitai[bot]","type":"Bot"}},"pull_request":{"number":1}}`, action))
			headers := signedHeaders(body, "test-secret", eventType)

			events := &contexts.EventContext{}
			code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
				Body:    body,
				Headers: headers,
				Logger:  logrus.NewEntry(logrus.New()),
				Configuration: map[string]any{
					"repository":    "test",
					"contentFilter": "@superplaneagent",
					"ignoreBots":    true,
					"allowedBots":   []string{"@CodeRabbitAI"},
				},
				Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
				Events:  events,
			})

			assert.Equal(t, http.StatusOK, code)
			assert.NoError(t, err)
			assert.Equal(t, 1, events.Count())
		})
	}

	t.Run("non-bot author matching allowlist name -> normal content filter behavior", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"Consider extracting this into a helper.","user":{"login":"coderabbitai","type":"User"}},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository":    "test",
				"contentFilter": "@superplaneagent",
				"allowedBots":   []string{"coderabbitai"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("near mention does not match", func(t *testing.T) {
		body := []byte(`{"action":"created","comment":{"body":"@superplaneagent-old please","user":{"type":"User"}},"pull_request":{"number":1}}`)
		headers := signedHeaders(body, "test-secret", eventType)

		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository":    "test",
				"contentFilter": "@superplaneagent",
			},
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  events,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("issue_comment event type -> ignored", func(t *testing.T) {
		body := []byte(`{"action":"created","issue":{"pull_request":{"url":"https://api.github.com/repos/test/test/pulls/1"}},"comment":{"body":"comment on PR conversation"}}`)
		headers := signedHeaders(body, "test-secret", "issue_comment")

		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
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

func TestOnPRReviewCommentEdits(t *testing.T) {
	trigger := &OnPRReviewComment{}
	cases := []struct {
		name    string
		filter  string
		replyTo int
		bodies  []string
		counts  []int
	}{
		{
			name:   "mention added, retained, and removed from reply",
			filter: "@superplaneagent", replyTo: 99,
			bodies: []string{"please fix this", "@superplaneagent please fix this", "@superplaneagent fix the race", "fix the race"},
			counts: []int{0, 1, 2, 2},
		},
		{name: "nonmatching reply edit", filter: "@superplaneagent", replyTo: 99, bodies: []string{"please fix this", "please fix the race"}, counts: []int{0, 0}},
		{name: "top-level edit remains excluded", filter: "@superplaneagent", bodies: []string{"please fix this", "@superplaneagent please fix this"}, counts: []int{0, 0}},
		{name: "empty filter accepts reply edits", replyTo: 99, bodies: []string{"please fix this", "please fix the race"}, counts: []int{1, 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events := &contexts.EventContext{}
			previous := ""
			for i, text := range tc.bodies {
				action := "edited"
				if i == 0 {
					action = "created"
				}
				body := []byte(fmt.Sprintf(
					`{"action":%q,"comment":{"id":42,"body":%q,"in_reply_to_id":%d,"user":{"login":"octocat","type":"User"}},"pull_request":{"number":1},"changes":{"body":{"from":%q}}}`,
					action, text, tc.replyTo, previous,
				))
				before := events.Count()
				code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
					Body:    body,
					Headers: signedHeaders(body, "test-secret", "pull_request_review_comment"),
					Logger:  logrus.NewEntry(logrus.New()),
					Configuration: map[string]any{
						"repository": "test", "contentFilter": tc.filter,
						"commentScope": "replies", "includeReviewSubmissions": false,
					},
					Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
					Events:  events,
				})
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, code)
				require.Equal(t, tc.counts[i], events.Count())
				if events.Count() > before {
					payload := map[string]any{}
					require.NoError(t, json.Unmarshal(body, &payload))
					assert.Equal(t, "github.prReviewComment", events.Payloads[before].Type)
					assert.Equal(t, payload, events.Payloads[before].Data)
				}
				previous = text
			}
		})
	}
}

func Test__OnPRReviewComment__Setup(t *testing.T) {
	trigger := OnPRReviewComment{}

	t.Run("webhook is requested with review event types", func(t *testing.T) {
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
		assert.Equal(t, "hello", webhookRequest.Repository)
		assert.ElementsMatch(t, []string{"pull_request_review_comment", "pull_request_review"}, webhookRequest.EventTypes)
		assert.Empty(t, webhookRequest.EventType)
	})

	t.Run("webhook is requested without review submissions when disabled", func(t *testing.T) {
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
			Integration: integrationCtx,
			HTTP:        httpCtx,
			Metadata:    &nodeMetadataCtx,
			Configuration: map[string]any{
				"repository":               "hello",
				"includeReviewSubmissions": false,
			},
		}))

		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookRequest := integrationCtx.WebhookRequests[0].(common.WebhookConfiguration)
		assert.Equal(t, "hello", webhookRequest.Repository)
		assert.Equal(t, "pull_request_review_comment", webhookRequest.EventType)
		assert.Empty(t, webhookRequest.EventTypes)
	})
}

func TestOnPRReviewCommentAddressAllBots(t *testing.T) {
	trigger := &OnPRReviewComment{}
	configuration := map[string]any{
		"repository":     "test",
		"contentFilter":  "@superplaneagent",
		"ignoreBots":     true,
		"addressAllBots": true,
		"commentScope":   "replies",
	}
	cases := []struct {
		name     string
		action   string
		userType string
		login    string
		body     string
		want     int
	}{
		{name: "other bot without mention starts a run", userType: "Bot", login: "coderabbitai[bot]", body: "please fix the race", want: 1},
		{name: "superplane comment does not start a run", userType: "Bot", login: "superplaneagent[bot]", body: "@superplaneagent please", want: 0},
		{name: "human without mention does not start a run", userType: "User", login: "octocat", body: "please fix the race", want: 0},
		{name: "human mention starts a run", userType: "User", login: "octocat", body: "@superplaneagent please", want: 1},
		{name: "edited bot reply bypasses the mention filter", action: "edited", userType: "Bot", login: "coderabbitai[bot]", body: "please fix the race", want: 1},
		{name: "edited superplane reply does not start a run", action: "edited", userType: "Bot", login: "superplaneagent[bot]", body: "@superplaneagent please", want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			action := tc.action
			if action == "" {
				action = "created"
			}
			body := []byte(fmt.Sprintf(
				`{"action":%q,"comment":{"body":%q,"in_reply_to_id":99,"user":{"login":%q,"type":%q}},"pull_request":{"number":1}}`,
				action, tc.body, tc.login, tc.userType,
			))
			events := &contexts.EventContext{}
			code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
				Body:          body,
				Headers:       signedHeaders(body, "test-secret", "pull_request_review_comment"),
				Logger:        logrus.NewEntry(logrus.New()),
				Configuration: configuration,
				Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
				Events:        events,
			})
			assert.Equal(t, http.StatusOK, code)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, events.Count())
		})
	}
}
