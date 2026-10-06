package pulls

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

func Test__OnPullRequest__HandleWebhook(t *testing.T) {
	trigger := &OnPullRequest{}
	eventType := "pull_request"

	t.Run("no X-Hub-Signature-256 -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-GitHub-Event", eventType)
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

	t.Run("invalid signature -> 403", func(t *testing.T) {
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")
		headers.Set("X-GitHub-Event", eventType)

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"action":"opened"}`),
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"actions":    []string{"opened"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("action is in list -> event is emitted", func(t *testing.T) {
		body := []byte(`{"action":"opened"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", eventType)

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"actions":    []string{"opened"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("edited action is in list -> event is emitted", func(t *testing.T) {
		body := []byte(`{"action":"edited","changes":{"title":{"from":"Old title"}}}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", eventType)

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"actions":    []string{"edited"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("newly supported actions emit when configured", func(t *testing.T) {
		actions := []string{
			"converted_to_draft",
			"locked",
			"unlocked",
			"enqueued",
			"dequeued",
			"milestoned",
			"demilestoned",
			"ready_for_review",
			"review_requested",
			"review_request_removed",
			"auto_merge_enabled",
			"auto_merge_disabled",
		}

		secret := "test-secret"

		for _, action := range actions {
			t.Run(action, func(t *testing.T) {
				body := []byte(fmt.Sprintf(`{"action":%q}`, action))

				h := hmac.New(sha256.New, []byte(secret))
				h.Write(body)
				signature := fmt.Sprintf("%x", h.Sum(nil))

				headers := http.Header{}
				headers.Set("X-Hub-Signature-256", "sha256="+signature)
				headers.Set("X-GitHub-Event", eventType)

				eventContext := &contexts.EventContext{}
				code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
					Body:    body,
					Headers: headers,
					Logger:  logrus.NewEntry(logrus.New()),
					Configuration: map[string]any{
						"repository": "test",
						"actions":    []string{action},
					},
					Webhook: &contexts.NodeWebhookContext{Secret: secret},
					Events:  eventContext,
				})

				assert.Equal(t, http.StatusOK, code)
				assert.NoError(t, err)
				assert.Equal(t, 1, eventContext.Count())
			})
		}
	})

	t.Run("action is not in list -> event is not emitted", func(t *testing.T) {
		body := []byte(`{"action":"closed"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", eventType)

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"actions":    []string{"opened"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 0)
	})

	t.Run("ignoreDrafts filters draft pull requests", func(t *testing.T) {
		ignoreDrafts := true
		keepDrafts := false

		cases := []struct {
			name         string
			body         string
			actions      []string
			ignoreDrafts *bool
			wantEvents   int
		}{
			{
				name:         "toggle on and opened draft does not emit",
				body:         `{"action":"opened","pull_request":{"draft":true}}`,
				actions:      []string{"opened"},
				ignoreDrafts: &ignoreDrafts,
				wantEvents:   0,
			},
			{
				name:         "toggle on and opened non-draft emits",
				body:         `{"action":"opened","pull_request":{"draft":false}}`,
				actions:      []string{"opened"},
				ignoreDrafts: &ignoreDrafts,
				wantEvents:   1,
			},
			{
				name:         "toggle on and ready_for_review emits",
				body:         `{"action":"ready_for_review","pull_request":{"draft":false}}`,
				actions:      []string{"ready_for_review"},
				ignoreDrafts: &ignoreDrafts,
				wantEvents:   1,
			},
			{
				name:         "toggle on and converted_to_draft does not emit",
				body:         `{"action":"converted_to_draft","pull_request":{"draft":true}}`,
				actions:      []string{"converted_to_draft"},
				ignoreDrafts: &ignoreDrafts,
				wantEvents:   0,
			},
			{
				name:         "toggle off and opened draft emits",
				body:         `{"action":"opened","pull_request":{"draft":true}}`,
				actions:      []string{"opened"},
				ignoreDrafts: &keepDrafts,
				wantEvents:   1,
			},
			{
				name:       "omitted toggle and opened draft emits",
				body:       `{"action":"opened","pull_request":{"draft":true}}`,
				actions:    []string{"opened"},
				wantEvents: 1,
			},
			{
				name:         "toggle on and missing draft emits",
				body:         `{"action":"opened","pull_request":{"title":"Ready"}}`,
				actions:      []string{"opened"},
				ignoreDrafts: &ignoreDrafts,
				wantEvents:   1,
			},
			{
				name:         "toggle on and missing pull_request emits",
				body:         `{"action":"opened"}`,
				actions:      []string{"opened"},
				ignoreDrafts: &ignoreDrafts,
				wantEvents:   1,
			},
			{
				name:         "toggle on and non-boolean draft emits",
				body:         `{"action":"opened","pull_request":{"draft":"true"}}`,
				actions:      []string{"opened"},
				ignoreDrafts: &ignoreDrafts,
				wantEvents:   1,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				configuration := map[string]any{
					"repository": "test",
					"actions":    tc.actions,
				}
				if tc.ignoreDrafts != nil {
					configuration["ignoreDrafts"] = *tc.ignoreDrafts
				}

				code, events, err := handleSignedPullRequestWebhook([]byte(tc.body), configuration)

				assert.Equal(t, http.StatusOK, code)
				assert.NoError(t, err)
				assert.Equal(t, tc.wantEvents, events.Count())
			})
		}
	})
}

func Test__OnPullRequest__OnlyFactoryPullRequests(t *testing.T) {
	body := []byte(`{
		"action": "opened",
		"number": 42,
		"pull_request": {
			"number": 42,
			"html_url": "https://github.com/acme/app/pull/42",
			"draft": false
		},
		"repository": {"full_name": "acme/app"}
	}`)
	configuration := map[string]any{
		"repository":              "acme/app",
		"actions":                 []string{"opened"},
		"onlyFactoryPullRequests": true,
	}
	match := &core.PullRequestMatch{
		PullRequest: &core.PullRequest{ID: "pr-1", Number: 42, Repository: "acme/app"},
		WorkOrder:   &core.WorkOrder{ID: "wo-1", Number: 12, Title: "Show a clearer empty state"},
	}

	t.Run("includes the factory pull request and task", func(t *testing.T) {
		factoryCtx := &recordingFactory{match: match}

		code, events, err := handleSignedPullRequest(body, configuration, factoryCtx)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, factoryCtx.calls)
		assert.Equal(t, core.FindPullRequestParams{
			Provider:   "github",
			Repository: "acme/app",
			Number:     42,
			URL:        "https://github.com/acme/app/pull/42",
		}, factoryCtx.params)
		require.Equal(t, 1, events.Count())
		payload := events.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "github.pullRequest", events.Payloads[0].Type)
		assert.Equal(t, "opened", payload["action"])
		assert.Equal(t, match.PullRequest, payload["pullRequest"])
		assert.Equal(t, match.WorkOrder, payload["workOrder"])
	})

	t.Run("does not start a run when the pull request is not in this factory", func(t *testing.T) {
		code, events, err := handleSignedPullRequest(body, configuration, &recordingFactory{err: core.ErrPullRequestNotFound})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("leaves the event unchanged when the toggle is off", func(t *testing.T) {
		factoryCtx := &recordingFactory{match: match}
		off := map[string]any{
			"repository": "acme/app",
			"actions":    []string{"opened"},
		}

		code, events, err := handleSignedPullRequest(body, off, factoryCtx)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, factoryCtx.calls)
		require.Equal(t, 1, events.Count())
		payload := events.Payloads[0].Data.(map[string]any)
		assert.NotContains(t, payload, "workOrder")
	})

	t.Run("fails when the app is not owned by a factory", func(t *testing.T) {
		code, events, err := handleSignedPullRequest(body, configuration, nil)

		assert.Equal(t, http.StatusInternalServerError, code)
		assert.ErrorContains(t, err, "not owned by a factory")
		assert.Equal(t, 0, events.Count())
	})
}

func handleSignedPullRequestWebhook(body []byte, configuration map[string]any) (int, *contexts.EventContext, error) {
	return handleSignedPullRequest(body, configuration, nil)
}

func handleSignedPullRequest(body []byte, configuration map[string]any, factoryCtx core.FactoryContext) (int, *contexts.EventContext, error) {
	secret := "test-secret"
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	signature := fmt.Sprintf("%x", h.Sum(nil))

	headers := http.Header{}
	headers.Set("X-Hub-Signature-256", "sha256="+signature)
	headers.Set("X-GitHub-Event", "pull_request")

	eventContext := &contexts.EventContext{}
	code, _, err := (&OnPullRequest{}).HandleWebhook(core.WebhookRequestContext{
		Body:          body,
		Headers:       headers,
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: configuration,
		Webhook:       &contexts.NodeWebhookContext{Secret: secret},
		Events:        eventContext,
		Factory:       factoryCtx,
	})

	return code, eventContext, err
}

type recordingFactory struct {
	calls  int
	params core.FindPullRequestParams
	match  *core.PullRequestMatch
	err    error
}

func (f *recordingFactory) FindPullRequest(params core.FindPullRequestParams) (*core.PullRequestMatch, error) {
	f.calls++
	f.params = params
	return f.match, f.err
}

func (f *recordingFactory) CreateWorkOrder(core.WorkOrderParams) (*core.WorkOrder, bool, error) {
	return nil, false, nil
}

func (f *recordingFactory) FindWorkOrder(core.FindWorkOrderParams) (*core.WorkOrder, error) {
	return nil, nil
}

func (f *recordingFactory) UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams) (*core.WorkOrder, bool, error) {
	return nil, false, nil
}

func (f *recordingFactory) AddWorkOrderComment(core.AddWorkOrderCommentParams) error {
	return nil
}

func (f *recordingFactory) AddWorkOrderArtifact(core.AddWorkOrderArtifactParams) (*core.WorkOrderArtifact, error) {
	return nil, nil
}

func (f *recordingFactory) ReportWorkOrderCheck(core.ReportWorkOrderCheckParams) (*core.WorkOrderCheck, error) {
	return nil, nil
}

func (f *recordingFactory) SetWorkOrderStatusNote(core.SetWorkOrderStatusNoteParams) (*core.WorkOrderStatusNote, error) {
	return nil, nil
}

func (f *recordingFactory) AddPullRequest(core.AddPullRequestParams) (*core.PullRequest, error) {
	return nil, nil
}

func (f *recordingFactory) UpdatePullRequest(core.UpdatePullRequestParams) (*core.PullRequest, error) {
	return nil, nil
}

func (f *recordingFactory) AddPullRequestActivity(core.AddPullRequestActivityParams) (*core.PullRequestActivityResult, error) {
	return nil, nil
}

func (f *recordingFactory) UpdatePullRequestActivity(core.UpdatePullRequestActivityParams) (*core.PullRequestActivityResult, error) {
	return nil, nil
}

func Test__OnPullRequest__Setup(t *testing.T) {
	trigger := OnPullRequest{}

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
		webhookRequest := integrationCtx.WebhookRequests[0].(common.WebhookConfiguration)
		assert.Equal(t, webhookRequest.EventType, "pull_request")
		assert.Equal(t, webhookRequest.Repository, "hello")
	})
}
