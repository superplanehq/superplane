package bitbucket

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestForgePullRequestEventsReachFactoryTrigger(t *testing.T) {
	for _, test := range []struct{ event, action string }{
		{"created", "created"}, {"updated", "updated"}, {"fulfilled", "merged"}, {"rejected", "declined"},
	} {
		t.Run(test.event, func(t *testing.T) {
			eventType := "avi:bitbucket:" + test.event + ":pullrequest"
			timestamp := time.Now().Format(time.RFC3339)
			body, err := json.Marshal(ForgePullRequestPayload(eventType, timestamp, "{repo}", "acme/widgets",
				map[string]any{"id": 2, "state": "OPEN"}, nil))
			require.NoError(t, err)
			headers := http.Header{}
			headers.Set("X-Event-Key", ForgeEventKey(eventType))
			headers.Set("X-Hub-Signature", "sha256="+signBitbucketPayload("secret", body))
			factory := &recordingBitbucketFactory{match: &core.PullRequestMatch{
				PullRequest: &core.PullRequest{ID: "pr"}, WorkOrder: &core.WorkOrder{ID: "task"},
			}}
			events := &contexts.EventContext{}
			code, _, err := (&OnPullRequest{}).HandleWebhook(core.WebhookRequestContext{
				Body: body, Headers: headers, Webhook: &contexts.NodeWebhookContext{Secret: "secret"},
				Configuration: map[string]any{"repository": "acme/widgets", "actions": []string{test.action}, "onlyFactoryPullRequests": true},
				Factory:       factory, Events: events,
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, code)
			require.Equal(t, 1, events.Count())
			assert.Equal(t, core.FindPullRequestParams{Provider: "bitbucket", Repository: "acme/widgets", Number: 2,
				URL: "https://bitbucket.org/acme/widgets/pull-requests/2"}, factory.params)
			payload := events.Payloads[0].Data.(map[string]any)
			assert.Equal(t, test.action, payload["action"])
			assert.Equal(t, factory.match.WorkOrder, payload["workOrder"])
			view := payload["pull_request"].(map[string]any)
			if test.action == "declined" {
				assert.Equal(t, timestamp, view["closed_at"])
				assert.Equal(t, "closed", view["state"])
			}
		})
	}
}

func TestForgeEventKeyMapsBuildStatus(t *testing.T) {
	assert.Equal(t, "repo:commit_status_created", ForgeEventKey("avi:bitbucket:created:build-status"))
	assert.Equal(t, "repo:commit_status_updated", ForgeEventKey("avi:bitbucket:updated:build-status"))
	assert.True(t, IsForgeBuildEvent("avi:bitbucket:created:build-status"))
	assert.True(t, IsForgeBuildEvent("avi:bitbucket:updated:build-status"))
	assert.False(t, IsForgeBuildEvent("avi:bitbucket:created:pullrequest"))
	assert.True(t, IsForgePullRequestEvent("avi:bitbucket:created:pullrequest"))
	assert.False(t, IsForgePullRequestEvent("avi:bitbucket:created:build-status"))
	assert.Equal(t, "", ForgeEventKey("avi:bitbucket:created:repository"))
}

func TestForgeBuildStatusPayloadNormalizesCommitStatus(t *testing.T) {
	sha := "b88c4b490b648bf960eba6f59123456797960e55"
	timestamp := "2025-01-31T01:03:17.137933Z"
	payload := ForgeBuildStatusPayload(timestamp, "{repo-uuid}", "acme/widgets", map[string]any{
		"key": "my-build1", "state": "SUCCESSFUL", "url": "https://ci.example/1",
		"commit":    map[string]any{"hash": sha},
		"createdOn": "2025-01-31T00:42:46.866793Z",
		"updatedOn": "2025-01-31T01:03:17.105700Z",
	}, map[string]any{"nickname": "ada"})

	repo, _ := payload["repository"].(map[string]any)
	assert.Equal(t, "acme/widgets", repo["full_name"])
	status, _ := payload["commit_status"].(map[string]any)
	assert.Equal(t, "my-build1", status["key"])
	assert.Equal(t, "SUCCESSFUL", status["state"])
	assert.Equal(t, "https://ci.example/1", status["url"])
	commit, _ := status["commit"].(map[string]any)
	assert.Equal(t, sha, commit["hash"])
	assert.Equal(t, "2025-01-31T00:42:46.866793Z", status["created_on"])
	assert.Equal(t, "2025-01-31T01:03:17.105700Z", status["updated_on"])
	assert.Equal(t, sha, ForgeBuildCommitSHA(map[string]any{"commit": map[string]any{"hash": sha}}))
	assert.Equal(t, "my-build1", ForgeBuildKey(map[string]any{"key": "my-build1"}))
	assert.True(t, IsValidFullCommitSHA(sha))
	assert.False(t, IsValidFullCommitSHA("abc123"))
	assert.False(t, IsValidFullCommitSHA(""))
}

func TestForgeBuildStatusEventsReachWaiter(t *testing.T) {
	for _, eventType := range []string{"avi:bitbucket:created:build-status", "avi:bitbucket:updated:build-status"} {
		t.Run(eventType, func(t *testing.T) {
			sha := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
			payload, err := json.Marshal(ForgeBuildStatusPayload(
				time.Now().Format(time.RFC3339), "{repo}", "acme/widgets",
				map[string]any{"key": "build-a", "state": "SUCCESSFUL", "commit": map[string]any{"hash": sha}}, nil))
			require.NoError(t, err)
			headers := http.Header{}
			headers.Set("X-Event-Key", ForgeEventKey(eventType))
			headers.Set("X-Hub-Signature", "sha256="+signBitbucketPayload("secret", payload))

			requests := &contexts.RequestContext{}
			executionCtx := &core.ExecutionContext{
				ExecutionState: &contexts.ExecutionStateContext{},
				Requests:       requests,
			}
			code, _, err := (&WaitForBuilds{}).HandleWebhook(core.WebhookRequestContext{
				Body:    payload,
				Headers: headers,
				Webhook: &contexts.NodeWebhookContext{Secret: "secret"},
				Configuration: map[string]any{
					"repository": "acme/widgets", "ref": sha, "buildKeys": []string{"build-a"},
				},
				FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
					assert.Equal(t, waitBuildsRefKV, key)
					return executionCtx, nil
				},
			})
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, code)
			assert.Equal(t, waitBuildsEvaluateHook, requests.Action)
		})
	}
}
