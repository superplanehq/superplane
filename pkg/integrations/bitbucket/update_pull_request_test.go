package bitbucket

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const updatedPullRequestResponse = `{
  "id": 42,
  "title": "Add login page and session handling",
  "state": "OPEN",
  "source": {"branch": {"name": "feature/login-page"}},
  "destination": {"branch": {"name": "main"}}
}`

func Test__UpdatePullRequest__Setup(t *testing.T) {
	action := &UpdatePullRequest{}

	t.Run("pull request ID is required", func(t *testing.T) {
		err := action.Setup(newSetupContext(map[string]any{
			"repository": testRepositoryName,
			"title":      "New title",
		}))

		require.ErrorContains(t, err, "pull request ID is required")
	})

	t.Run("at least one field to update is required", func(t *testing.T) {
		err := action.Setup(newSetupContext(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "42",
		}))

		require.ErrorContains(t, err, "at least one field to update is required")
	})

	t.Run("an enabled but blank title is rejected", func(t *testing.T) {
		err := action.Setup(newSetupContext(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "42",
			"title":         "",
		}))

		require.ErrorContains(t, err, "title cannot be empty when it is enabled")
	})

	t.Run("an enabled but blank target branch is rejected", func(t *testing.T) {
		err := action.Setup(newSetupContext(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "42",
			"targetBranch":  "",
		}))

		require.ErrorContains(t, err, "target branch cannot be empty when it is enabled")
	})

	// Clearing the description is a legitimate update, unlike clearing the title.
	t.Run("an enabled but blank description is accepted", func(t *testing.T) {
		err := action.Setup(newSetupContext(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "42",
			"description":   "",
		}))

		require.NoError(t, err)
	})
}

func Test__UpdatePullRequest__Execute(t *testing.T) {
	action := &UpdatePullRequest{}

	t.Run("only the enabled fields are sent", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "42",
			"title":         "Add login page and session handling",
		}, jsonResponse(http.StatusOK, updatedPullRequestResponse))

		require.NoError(t, action.Execute(fixture.Context))

		require.Len(t, fixture.HTTP.Requests, 1)
		request := fixture.HTTP.Requests[0]
		assert.Equal(t, http.MethodPut, request.Method)
		assert.Equal(t, "https://api.bitbucket.org/2.0/repositories/superplane/web/pullrequests/42", request.URL.String())

		body := requestBody(t, request)
		assert.Equal(t, "Add login page and session handling", body["title"])

		// Fields that were never toggled on must stay out of the payload, otherwise
		// Bitbucket would overwrite them with a zero value.
		for _, field := range []string{"description", "destination", "reviewers", "close_source_branch"} {
			_, present := body[field]
			assert.False(t, present, "%s should not be sent", field)
		}

		pullRequest := emittedPayload[*PullRequest](t, fixture.ExecutionState, 0)
		assert.Equal(t, "Add login page and session handling", pullRequest.Title)
	})

	t.Run("clearing the description sends an empty string", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "42",
			"description":   "",
		}, jsonResponse(http.StatusOK, updatedPullRequestResponse))

		require.NoError(t, action.Execute(fixture.Context))

		body := requestBody(t, fixture.HTTP.Requests[0])
		description, present := body["description"]
		require.True(t, present)
		assert.Equal(t, "", description)
	})

	t.Run("clearing the reviewers sends an empty list", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "42",
			"reviewers":     []string{},
		}, jsonResponse(http.StatusOK, updatedPullRequestResponse))

		require.NoError(t, action.Execute(fixture.Context))

		body := requestBody(t, fixture.HTTP.Requests[0])
		reviewers, present := body["reviewers"]
		require.True(t, present)
		assert.Empty(t, reviewers)
	})

	t.Run("a non-numeric pull request ID is reported", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "not-a-number",
			"title":         "New title",
		})

		err := action.Execute(fixture.Context)

		require.ErrorContains(t, err, `pull request ID "not-a-number" is not a number`)
		assert.Empty(t, fixture.HTTP.Requests)
	})

	t.Run("a zero pull request ID is reported", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository":    testRepositoryName,
			"pullRequestId": "0",
			"title":         "New title",
		})

		err := action.Execute(fixture.Context)

		require.ErrorContains(t, err, "pull request ID must be greater than zero")
	})
}
