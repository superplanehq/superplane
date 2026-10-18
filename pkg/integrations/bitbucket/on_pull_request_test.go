package bitbucket

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func pullRequestFixture(t *testing.T) []byte {
	t.Helper()
	body, err := os.ReadFile("example_data_on_pull_request.json")
	require.NoError(t, err)
	return body
}

func Test__OnPullRequest__Setup(t *testing.T) {
	trigger := OnPullRequest{}
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

	t.Run("webhook requests pull-request events", func(t *testing.T) {
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
				"actions":    []string{"created", "merged"},
			},
		}))

		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookRequest, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.ElementsMatch(t, []string{"pullrequest:created", "pullrequest:fulfilled"}, webhookRequest.EventTypes)
		assert.Equal(t, "widgets", webhookRequest.RepositorySlug)
	})
}

func Test__OnPullRequest__HandleWebhook(t *testing.T) {
	trigger := &OnPullRequest{}

	configuration := map[string]any{
		"repository": "acme/widgets",
		"actions":    []string{"created", "merged", "declined"},
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

	t.Run("event is not a pull request -> 200", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Event-Key", "repo:push")

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Events:  eventContext,
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
			Body:    pullRequestFixture(t),
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("repository mismatch -> event is not emitted", func(t *testing.T) {
		body := pullRequestFixture(t)
		headers, eventContext := signed(t, "pullrequest:created", body)

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

	t.Run("unsubscribed action -> event is not emitted", func(t *testing.T) {
		body := pullRequestFixture(t)
		headers, eventContext := signed(t, "pullrequest:approved", body)

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: map[string]any{
				"repository": "acme/widgets",
				"actions":    []string{"created"},
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("created -> normalized event is emitted", func(t *testing.T) {
		body := pullRequestFixture(t)
		headers, eventContext := signed(t, "pullrequest:created", body)

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
		assert.Equal(t, "bitbucket.pullRequest", eventContext.Payloads[0].Type)
		event, ok := eventContext.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "created", event["action"])
		repository, ok := event["repository"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "acme/widgets", repository["full_name"])
		pullRequest, ok := event["pullrequest"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "OPEN", pullRequest["state"])
		source, ok := pullRequest["source"].(map[string]any)
		require.True(t, ok)
		branch, ok := source["branch"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "feat/retry-handling", branch["name"])
		actor, ok := event["actor"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "{d301aafa-d676-4ee0-a3f1-8b94c681feaa}", actor["uuid"])
	})

	t.Run("fulfilled maps to merged", func(t *testing.T) {
		body := pullRequestFixture(t)
		headers, eventContext := signed(t, "pullrequest:fulfilled", body)

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
		event, ok := eventContext.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "merged", event["action"])
	})
}

func Test__BitbucketWebhook__Merge(t *testing.T) {
	handler := &BitbucketWebhookHandler{}

	t.Run("unions event types for the same repository", func(t *testing.T) {
		merged, changed, err := handler.Merge(
			map[string]any{"eventTypes": []string{"repo:push"}, "repositorySlug": "widgets"},
			map[string]any{"eventTypes": []string{"pullrequest:created"}, "repositorySlug": "widgets"},
		)
		require.NoError(t, err)
		assert.True(t, changed)
		config, ok := merged.(WebhookConfiguration)
		require.True(t, ok)
		assert.ElementsMatch(t, []string{"repo:push", "pullrequest:created"}, config.EventTypes)
	})

	t.Run("duplicate events do not change the webhook", func(t *testing.T) {
		_, changed, err := handler.Merge(
			map[string]any{"eventTypes": []string{"repo:push"}, "repositorySlug": "widgets"},
			map[string]any{"eventTypes": []string{"repo:push"}, "repositorySlug": "widgets"},
		)
		require.NoError(t, err)
		assert.False(t, changed)
	})

	t.Run("different repositories never merge", func(t *testing.T) {
		_, changed, err := handler.Merge(
			map[string]any{"eventTypes": []string{"repo:push"}, "repositorySlug": "widgets"},
			map[string]any{"eventTypes": []string{"pullrequest:created"}, "repositorySlug": "other"},
		)
		require.NoError(t, err)
		assert.False(t, changed)
	})
}

func Test__OnPullRequest__OnlyFactoryPullRequests(t *testing.T) {
	configuration := map[string]any{
		"repository":              "acme/widgets",
		"actions":                 []string{"merged", "declined"},
		"onlyFactoryPullRequests": true,
	}
	match := &core.PullRequestMatch{
		PullRequest: &core.PullRequest{ID: "pr-1", Number: 42, Repository: "acme/widgets"},
		WorkOrder:   &core.WorkOrder{ID: "wo-1", Number: 12, Title: "Add retry handling"},
	}

	handle := func(t *testing.T, eventKey string, factoryCtx core.FactoryContext) (int, *contexts.EventContext, error) {
		t.Helper()
		body := pullRequestFixture(t)
		headers := http.Header{}
		headers.Set("X-Event-Key", eventKey)
		headers.Set("X-Hub-Signature", "sha256="+signBitbucketPayload("test-secret", body))
		eventContext := &contexts.EventContext{}
		code, _, err := triggerHandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: configuration,
			Events:        eventContext,
			Factory:       factoryCtx,
		})
		return code, eventContext, err
	}

	t.Run("includes the factory pull request and task", func(t *testing.T) {
		factoryCtx := &recordingBitbucketFactory{match: match}

		code, events, err := handle(t, "pullrequest:fulfilled", factoryCtx)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		require.Equal(t, 1, factoryCtx.calls)
		assert.Equal(t, core.FindPullRequestParams{
			Provider:   "bitbucket",
			Repository: "acme/widgets",
			Number:     42,
			URL:        "https://bitbucket.org/acme/widgets/pull-requests/42",
		}, factoryCtx.params)
		require.Equal(t, 1, events.Count())
		payload := events.Payloads[0].Data.(map[string]any)
		assert.Equal(t, "bitbucket.pullRequest", events.Payloads[0].Type)
		assert.Equal(t, "merged", payload["action"])
		assert.Equal(t, match.PullRequest, payload["pullRequest"])
		assert.Equal(t, match.WorkOrder, payload["workOrder"])
	})

	t.Run("does not start a run when the pull request is not in this factory", func(t *testing.T) {
		code, events, err := handle(t, "pullrequest:fulfilled", &recordingBitbucketFactory{err: core.ErrPullRequestNotFound})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("leaves the event unchanged when the toggle is off", func(t *testing.T) {
		factoryCtx := &recordingBitbucketFactory{match: match}
		body := pullRequestFixture(t)
		headers := http.Header{}
		headers.Set("X-Event-Key", "pullrequest:fulfilled")
		headers.Set("X-Hub-Signature", "sha256="+signBitbucketPayload("test-secret", body))
		eventContext := &contexts.EventContext{}
		off := map[string]any{
			"repository": "acme/widgets",
			"actions":    []string{"merged", "declined"},
		}

		code, _, err := triggerHandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: off,
			Events:        eventContext,
			Factory:       factoryCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, 0, factoryCtx.calls)
		require.Equal(t, 1, eventContext.Count())
		payload := eventContext.Payloads[0].Data.(map[string]any)
		assert.NotContains(t, payload, "workOrder")
	})

	t.Run("fails when the app is not owned by a factory", func(t *testing.T) {
		code, events, err := handle(t, "pullrequest:fulfilled", nil)

		assert.Equal(t, http.StatusInternalServerError, code)
		assert.ErrorContains(t, err, "not owned by a factory")
		assert.Equal(t, 0, events.Count())
	})
}

func triggerHandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return (&OnPullRequest{}).HandleWebhook(ctx)
}

type recordingBitbucketFactory struct {
	calls  int
	params core.FindPullRequestParams
	match  *core.PullRequestMatch
	err    error
}

func (f *recordingBitbucketFactory) FindPullRequest(params core.FindPullRequestParams) (*core.PullRequestMatch, error) {
	f.calls++
	f.params = params
	return f.match, f.err
}

func (f *recordingBitbucketFactory) CreateWorkOrder(core.WorkOrderParams) (*core.WorkOrder, bool, error) {
	return nil, false, nil
}

func (f *recordingBitbucketFactory) FindWorkOrder(core.FindWorkOrderParams) (*core.WorkOrder, error) {
	return nil, nil
}

func (f *recordingBitbucketFactory) UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams) (*core.WorkOrder, bool, error) {
	return nil, false, nil
}

func (f *recordingBitbucketFactory) AddWorkOrderComment(core.AddWorkOrderCommentParams) error {
	return nil
}

func (f *recordingBitbucketFactory) AddWorkOrderArtifact(core.AddWorkOrderArtifactParams) (*core.WorkOrderArtifact, error) {
	return nil, nil
}

func (f *recordingBitbucketFactory) ReportWorkOrderCheck(core.ReportWorkOrderCheckParams) (*core.WorkOrderCheck, error) {
	return nil, nil
}

func (f *recordingBitbucketFactory) SetWorkOrderStatusNote(core.SetWorkOrderStatusNoteParams) (*core.WorkOrderStatusNote, error) {
	return nil, nil
}

func (f *recordingBitbucketFactory) AddPullRequest(core.AddPullRequestParams) (*core.PullRequest, error) {
	return nil, nil
}

func (f *recordingBitbucketFactory) UpdatePullRequest(core.UpdatePullRequestParams) (*core.PullRequest, error) {
	return nil, nil
}

func (f *recordingBitbucketFactory) AddPullRequestActivity(core.AddPullRequestActivityParams) (*core.PullRequestActivityResult, error) {
	return nil, nil
}

func (f *recordingBitbucketFactory) UpdatePullRequestActivity(core.UpdatePullRequestActivityParams) (*core.PullRequestActivityResult, error) {
	return nil, nil
}

func (f *recordingBitbucketFactory) VCSProvider() (string, error) {
	return "", nil
}

func Test__OnPush__RepositoryRouting(t *testing.T) {

	trigger := &OnPush{}
	configuration := map[string]any{
		"repository": "acme/widgets",
		"refs": []configuration.Predicate{
			{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"},
		},
	}

	t.Run("payload for another repository is ignored", func(t *testing.T) {
		body := []byte(`{"repository":{"uuid":"{other}","full_name":"acme/other"},"push":{"changes":[{"new":{"type":"branch","name":"main"}}]}}`)
		headers := http.Header{}
		headers.Set("X-Event-Key", "repo:push")
		headers.Set("X-Hub-Signature", "sha256="+signBitbucketPayload("test-secret", body))

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Metadata: &contexts.MetadataContext{Metadata: NodeMetadata{Repository: &RepositoryMetadata{
				UUID:     "{widgets}",
				FullName: "acme/widgets",
				Slug:     "widgets",
				Name:     "widgets",
			}}},
			Configuration: configuration,
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("payload without repository identity still emits", func(t *testing.T) {
		body := []byte(`{"push":{"changes":[{"new":{"type":"branch","name":"main"}}]}}`)
		headers := http.Header{}
		headers.Set("X-Event-Key", "repo:push")
		headers.Set("X-Hub-Signature", "sha256="+signBitbucketPayload("test-secret", body))

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: configuration,
			Events:        eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})
}
