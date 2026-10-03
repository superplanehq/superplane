package bitbucket

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mergedPullRequestResponse = `{
  "id": 42,
  "title": "Add login page",
  "state": "MERGED",
  "merge_commit": {"hash": "7ab3c9d1e5f28046b1c3a7d9e2b4c6d8e0f1a2b3"},
  "source": {"branch": {"name": "feature/login-page"}},
  "destination": {"branch": {"name": "main"}}
}`

func Test__MergePullRequest__Setup(t *testing.T) {
	action := &MergePullRequest{}

	t.Run("pull request ID is required", func(t *testing.T) {
		err := action.Setup(newSetupContext(map[string]any{
			"repository":    testRepositoryName,
			"mergeStrategy": MergeStrategySquash,
		}))

		require.ErrorContains(t, err, "pull request ID is required")
	})

	t.Run("unsupported merge strategy is rejected", func(t *testing.T) {
		err := action.Setup(newSetupContext(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "42",
			"mergeStrategy": "rebase",
		}))

		require.ErrorContains(t, err, `unsupported merge strategy "rebase"`)
	})

	t.Run("valid configuration is accepted", func(t *testing.T) {
		err := action.Setup(newSetupContext(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "42",
			"mergeStrategy": MergeStrategySquash,
		}))

		require.NoError(t, err)
	})
}

func Test__MergePullRequest__Execute(t *testing.T) {
	action := &MergePullRequest{}

	t.Run("pull request is merged and emitted", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository":        testRepositoryName,
			"pullRequestId":     "42",
			"mergeStrategy":     MergeStrategySquash,
			"message":           "Release 2026.08",
			"closeSourceBranch": true,
		}, jsonResponse(http.StatusOK, mergedPullRequestResponse))

		require.NoError(t, action.Execute(fixture.Context))

		require.Len(t, fixture.HTTP.Requests, 1)
		request := fixture.HTTP.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "https://api.bitbucket.org/2.0/repositories/superplane/web/pullrequests/42/merge", request.URL.String())

		body := requestBody(t, request)
		assert.Equal(t, "pullrequest", body["type"])
		assert.Equal(t, MergeStrategySquash, body["merge_strategy"])
		assert.Equal(t, "Release 2026.08", body["message"])
		assert.Equal(t, true, body["close_source_branch"])

		pullRequest := emittedPayload[*PullRequest](t, fixture.ExecutionState, 0)
		assert.Equal(t, "MERGED", pullRequest.State)
		require.NotNil(t, pullRequest.MergeCommit)
		assert.Equal(t, "7ab3c9d1e5f28046b1c3a7d9e2b4c6d8e0f1a2b3", pullRequest.MergeCommit.Hash)
	})

	// Bitbucket answers 202 with a polling task on slow merges. Reporting that as a
	// success would tell the canvas a merge happened when it has not.
	t.Run("an asynchronously queued merge fails instead of reporting success", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "42",
			"mergeStrategy": MergeStrategyMergeCommit,
		}, jsonResponse(http.StatusAccepted, `{"type":"pullrequest_merge_task","status":{"state":"PENDING"}}`))

		err := action.Execute(fixture.Context)

		require.ErrorContains(t, err, "queued the merge asynchronously")
		assert.False(t, fixture.ExecutionState.Passed)
	})

	t.Run("a merge conflict is reported", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "42",
			"mergeStrategy": MergeStrategyFastForward,
		}, jsonResponse(http.StatusBadRequest, `{"error":{"message":"branch has diverged"}}`))

		err := action.Execute(fixture.Context)

		require.ErrorContains(t, err, "failed to merge pull request")
		require.ErrorContains(t, err, "branch has diverged")
	})
}
