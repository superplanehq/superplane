package statuses

import (
	"net/http"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func Test__GetCombinedCommitStatus__Execute(t *testing.T) {
	component := GetCombinedCommitStatus{}

	t.Run("fails when configuration decode fails", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  "not a map",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("requires repository", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"ref": "main"},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("requires ref", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"repository": "hello"},
		})

		require.ErrorContains(t, err, "ref is required")
	})

	t.Run("emits the combined commit status", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"state": "failure",
					"sha": "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44",
					"total_count": 3,
					"commit_url": "https://api.github.com/repos/testhq/hello/commits/d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44",
					"repository_url": "https://api.github.com/repos/testhq/hello",
					"statuses": [
						{"state": "success", "context": "ci/build", "description": "Build passed"},
						{"state": "failure", "context": "ci/lint", "description": "Lint failed"},
						{"state": "pending", "context": "deploy/preview", "description": "Deployment pending"}
					]
				}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository": "hello",
				"ref":        "main",
			},
		})

		require.NoError(t, err)
		require.True(t, executionState.Passed)
		require.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		require.Equal(t, "github.combinedCommitStatus", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "/repos/testhq/hello/commits/main/status", httpCtx.Requests[0].URL.Path)

		payload := executionState.Payloads[0].(map[string]any)
		status := payload["data"].(*github.CombinedStatus)
		assert.Equal(t, "failure", status.GetState())
		assert.Equal(t, "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44", status.GetSHA())
		assert.Equal(t, 3, status.GetTotalCount())
		assert.Len(t, status.Statuses, 3)
	})
}
