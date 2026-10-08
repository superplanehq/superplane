package bitbucket

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__EvaluateBuildStatuses(t *testing.T) {
	builds := func(statuses ...CommitStatus) []BuildStatus {
		return normalizeBuildStatuses(statuses)
	}
	passing := CommitStatus{Key: "build-a", Name: "Build A", State: "SUCCESSFUL"}

	t.Run("passes when selected builds succeed", func(t *testing.T) {
		evaluation := evaluateBuildStatuses(builds(passing), []string{"build-a"}, false)
		assert.Equal(t, waitBuildsOutcomePassed, evaluation.Outcome)
		assert.Empty(t, evaluation.FailedBuilds)
	})

	t.Run("missing builds stay pending until the timeout", func(t *testing.T) {
		pending := evaluateBuildStatuses(builds(), []string{"build-a"}, false)
		assert.Equal(t, waitBuildsOutcomePending, pending.Outcome)

		timedOut := evaluateBuildStatuses(builds(), []string{"build-a"}, true)
		assert.Equal(t, waitBuildsOutcomeTimedOut, timedOut.Outcome)
	})

	t.Run("failed builds fail", func(t *testing.T) {
		evaluation := evaluateBuildStatuses(builds(
			passing,
			CommitStatus{Key: "build-b", State: "FAILED"},
		), []string{"build-a", "build-b"}, false)
		assert.Equal(t, waitBuildsOutcomeFailed, evaluation.Outcome)
		require.Len(t, evaluation.FailedBuilds, 1)
		assert.Equal(t, "build-b", evaluation.FailedBuilds[0].Key)
	})

	t.Run("stopped and unknown states", func(t *testing.T) {
		stopped := evaluateBuildStatuses(builds(CommitStatus{Key: "build-a", State: "STOPPED"}), []string{"build-a"}, false)
		assert.Equal(t, waitBuildsOutcomeFailed, stopped.Outcome)

		unknown := evaluateBuildStatuses(builds(CommitStatus{Key: "build-a", State: "weird"}), []string{"build-a"}, false)
		assert.Equal(t, waitBuildsOutcomePending, unknown.Outcome)
	})

	t.Run("in-progress builds stay pending", func(t *testing.T) {
		evaluation := evaluateBuildStatuses(builds(
			passing,
			CommitStatus{Key: "build-b", State: "INPROGRESS"},
		), []string{"build-a", "build-b"}, false)
		assert.Equal(t, waitBuildsOutcomePending, evaluation.Outcome)
	})

	t.Run("a display name does not satisfy a required key", func(t *testing.T) {
		evaluation := evaluateBuildStatuses(builds(
			CommitStatus{Key: "other", Name: "build-a", State: "SUCCESSFUL"},
		), []string{"build-a"}, false)
		assert.Equal(t, waitBuildsOutcomePending, evaluation.Outcome)
		assert.Equal(t, []string{"build-a"}, evaluation.MissingSelected)
	})
}

func Test__WaitForBuilds__Execute(t *testing.T) {
	component := &WaitForBuilds{}
	sha := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	integration := func() *contexts.IntegrationContext {
		return &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "token"},
			Metadata: Metadata{
				AuthType:  AuthTypeWorkspaceAccessToken,
				Workspace: &WorkspaceMetadata{Slug: "acme"},
			},
		}
	}

	t.Run("emits passed when selected builds succeed", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"values": [{"key": "build-a", "name": "Build A", "state": "SUCCESSFUL", "url": "https://ci.example/a"}]}`,
					)),
				},
			},
		}
		executionState := &contexts.ExecutionStateContext{}
		requests := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: WaitForBuildsConfiguration{
				Repository: "acme/widgets",
				Ref:        sha,
				BuildKeys:  []string{"build-a"},
			},
			HTTP:           httpCtx,
			Integration:    integration(),
			ExecutionState: executionState,
			Requests:       requests,
			Metadata:       &contexts.MetadataContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, waitBuildsPassedChannel, executionState.Channel)
		assert.Equal(t, waitBuildsPayloadType, executionState.Type)
		assert.Equal(t, "acme/widgets@"+sha, executionState.KVs[waitBuildsRefKV])
		assert.Empty(t, requests.Action)
	})

	t.Run("rejects empty build keys", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: WaitForBuildsConfiguration{
				Repository: "acme/widgets",
				Ref:        sha,
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    integration(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       &contexts.RequestContext{},
			Metadata:       &contexts.MetadataContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "buildKeys is required")
	})

	t.Run("stays pending and polls when builds are missing", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"values": []}`))},
			},
		}
		executionState := &contexts.ExecutionStateContext{}
		requests := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: WaitForBuildsConfiguration{
				Repository: "acme/widgets",
				Ref:        sha,
				BuildKeys:  []string{"build-a"},
			},
			HTTP:           httpCtx,
			Integration:    integration(),
			ExecutionState: executionState,
			Requests:       requests,
			Metadata:       &contexts.MetadataContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Empty(t, executionState.Channel)
		assert.Equal(t, waitBuildsEvaluateHook, requests.Action)
	})
}

func Test__WaitForBuilds__HandleWebhook(t *testing.T) {
	component := &WaitForBuilds{}
	configuration := map[string]any{
		"repository": "acme/widgets",
		"ref":        "abc123",
		"buildKeys":  []string{"build-a"},
	}

	t.Run("non-build events are ignored", func(t *testing.T) {
		body := []byte(`{}`)
		headers := http.Header{}
		headers.Set("X-Event-Key", "repo:push")
		headers.Set("X-Hub-Signature", "sha256="+signBitbucketPayload("test-secret", body))

		eventContext := &contexts.EventContext{}
		code, _, err := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
	})

	t.Run("repository mismatch is ignored", func(t *testing.T) {
		body := []byte(`{"repository": {"full_name": "acme/other"}, "commit_status": {"key": "build-a", "state": "SUCCESSFUL", "commit": {"hash": "abc123"}}}`)
		headers := http.Header{}
		headers.Set("X-Event-Key", "repo:commit_status_created")
		headers.Set("X-Hub-Signature", "sha256="+signBitbucketPayload("test-secret", body))

		code, _, err := component.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
			Configuration: configuration,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
	})
}

func Test__WaitBuildsTimeout(t *testing.T) {
	assert.Equal(t, 3600*time.Second, WaitForBuildsConfiguration{}.timeout())
	seconds := 60
	assert.Equal(t, 60*time.Second, WaitForBuildsConfiguration{TimeoutSeconds: &seconds}.timeout())
}
