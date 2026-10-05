package bitbucket

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const createdCommentResponse = `{
  "id": 481902337,
  "content": {"raw": "Preview environment is up", "markup": "markdown"},
  "links": {"html": {"href": "https://bitbucket.org/superplane/web/pull-requests/42/_/diff#comment-481902337"}}
}`

func Test__CreatePRComment__Setup(t *testing.T) {
	action := &CreatePRComment{}

	t.Run("pull request ID is required", func(t *testing.T) {
		err := action.Setup(newSetupContext(map[string]any{
			"repository": testRepositoryName,
			"body":       "hello",
		}))

		require.ErrorContains(t, err, "pull request ID is required")
	})

	t.Run("body is required", func(t *testing.T) {
		err := action.Setup(newSetupContext(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "42",
		}))

		require.ErrorContains(t, err, "body is required")
	})

	t.Run("valid configuration is accepted", func(t *testing.T) {
		err := action.Setup(newSetupContext(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "42",
			"body":          "Preview environment is up",
		}))

		require.NoError(t, err)
	})
}

func Test__CreatePRComment__Execute(t *testing.T) {
	action := &CreatePRComment{}

	t.Run("comment is posted and emitted", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "42",
			"body":          "Preview environment is up",
		}, jsonResponse(http.StatusCreated, createdCommentResponse))

		require.NoError(t, action.Execute(fixture.Context))

		require.Len(t, fixture.HTTP.Requests, 1)
		request := fixture.HTTP.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "https://api.bitbucket.org/2.0/repositories/superplane/web/pullrequests/42/comments", request.URL.String())

		body := requestBody(t, request)
		assert.Equal(t, "Preview environment is up", body["content"].(map[string]any)["raw"])

		comment := emittedPayload[*PullRequestComment](t, fixture.ExecutionState, 0)
		assert.Equal(t, 481902337, comment.ID)
		assert.Equal(t, "Preview environment is up", comment.Content.Raw)
		assert.Equal(t, "bitbucket.pullRequestComment", fixture.ExecutionState.Type)
	})

	t.Run("API error is reported", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "42",
			"body":          "Preview environment is up",
		}, jsonResponse(http.StatusNotFound, `{"error":{"message":"pull request not found"}}`))

		err := action.Execute(fixture.Context)

		require.ErrorContains(t, err, "failed to create pull request comment")
		require.ErrorContains(t, err, "pull request not found")
	})
}
