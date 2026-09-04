package bitbucket

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__GetCommitStatus__Setup(t *testing.T) {
	action := &GetCommitStatus{}

	t.Run("commit is required", func(t *testing.T) {
		err := action.Setup(newSetupContext(map[string]any{
			"repository": testRepositoryName,
		}))

		require.ErrorContains(t, err, "commit is required")
	})

	t.Run("valid configuration is accepted", func(t *testing.T) {
		err := action.Setup(newSetupContext(map[string]any{
			"repository": testRepositoryName,
			"commit":     "9fec847784abb10b2fa567ee63b85bd238955d0e",
		}))

		require.NoError(t, err)
	})
}

func Test__GetCommitStatus__Execute(t *testing.T) {
	action := &GetCommitStatus{}

	config := map[string]any{
		"repository": testRepositoryName,
		"commit":     "9fec847784abb10b2fa567ee63b85bd238955d0e",
	}

	t.Run("statuses are aggregated and emitted", func(t *testing.T) {
		fixture := newExecutionFixture(config, jsonResponse(http.StatusOK, `{
			"values": [
				{"key": "BITBUCKETPIPELINE", "state": "SUCCESSFUL", "refname": "main"},
				{"key": "security-scan", "state": "FAILED", "refname": "main"}
			]
		}`))

		require.NoError(t, action.Execute(fixture.Context))

		require.Len(t, fixture.HTTP.Requests, 1)
		request := fixture.HTTP.Requests[0]
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(
			t,
			"https://api.bitbucket.org/2.0/repositories/superplane/web/commit/9fec847784abb10b2fa567ee63b85bd238955d0e/statuses?pagelen=100",
			request.URL.String(),
		)

		combined := emittedPayload[CombinedCommitStatus](t, fixture.ExecutionState, 0)
		assert.Equal(t, "9fec847784abb10b2fa567ee63b85bd238955d0e", combined.Commit)
		assert.Equal(t, StateFailed, combined.State)
		assert.Equal(t, 2, combined.TotalCount)
		require.Len(t, combined.Statuses, 2)
		assert.Equal(t, "bitbucket.commitStatuses", fixture.ExecutionState.Type)
	})

	// A commit with no build status must not read as a pass, or a deploy gate would
	// let through code that was never built.
	t.Run("a commit with no statuses reports NO_STATUS", func(t *testing.T) {
		fixture := newExecutionFixture(config, jsonResponse(http.StatusOK, `{"values": []}`))

		require.NoError(t, action.Execute(fixture.Context))

		combined := emittedPayload[CombinedCommitStatus](t, fixture.ExecutionState, 0)
		assert.Equal(t, StateNoStatus, combined.State)
		assert.Equal(t, 0, combined.TotalCount)
		assert.Empty(t, combined.Statuses)
	})

	t.Run("every page is followed", func(t *testing.T) {
		fixture := newExecutionFixture(
			config,
			jsonResponse(http.StatusOK, `{
				"values": [{"key": "BITBUCKETPIPELINE", "state": "SUCCESSFUL"}],
				"next": "https://api.bitbucket.org/2.0/repositories/superplane/web/commit/9fec847784abb10b2fa567ee63b85bd238955d0e/statuses?page=2"
			}`),
			jsonResponse(http.StatusOK, `{"values": [{"key": "security-scan", "state": "SUCCESSFUL"}]}`),
		)

		require.NoError(t, action.Execute(fixture.Context))

		assert.Len(t, fixture.HTTP.Requests, 2)

		combined := emittedPayload[CombinedCommitStatus](t, fixture.ExecutionState, 0)
		assert.Equal(t, StateSuccessful, combined.State)
		assert.Equal(t, 2, combined.TotalCount)
	})

	t.Run("API error is reported", func(t *testing.T) {
		fixture := newExecutionFixture(config, jsonResponse(http.StatusNotFound, `{"error":{"message":"commit not found"}}`))

		err := action.Execute(fixture.Context)

		require.ErrorContains(t, err, "failed to get commit statuses")
		require.ErrorContains(t, err, "commit not found")
	})
}
