package bitbucket

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

const createdPullRequestResponse = `{
  "id": 42,
  "title": "Add login page",
  "state": "OPEN",
  "source": {"branch": {"name": "feature/login-page"}},
  "destination": {"branch": {"name": "main"}},
  "links": {"html": {"href": "https://bitbucket.org/superplane/web/pull-requests/42"}}
}`

func Test__CreatePullRequest__Setup(t *testing.T) {
	action := &CreatePullRequest{}

	t.Run("source branch is required", func(t *testing.T) {
		err := action.Setup(newSetupContext(map[string]any{
			"repository": testRepositoryName,
			"title":      "Add login page",
		}))

		require.ErrorContains(t, err, "source branch is required")
	})

	t.Run("title is required", func(t *testing.T) {
		err := action.Setup(newSetupContext(map[string]any{
			"repository":   testRepositoryName,
			"sourceBranch": "feature/login-page",
		}))

		require.ErrorContains(t, err, "title is required")
	})

	t.Run("valid configuration is accepted", func(t *testing.T) {
		err := action.Setup(newSetupContext(map[string]any{
			"repository":   testRepositoryName,
			"sourceBranch": "feature/login-page",
			"title":        "Add login page",
		}))

		require.NoError(t, err)
	})
}

func Test__CreatePullRequest__Execute(t *testing.T) {
	action := &CreatePullRequest{}

	t.Run("pull request is created and emitted", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository":        testRepositoryName,
			"sourceBranch":      "feature/login-page",
			"targetBranch":      "main",
			"title":             "Add login page",
			"description":       "Adds the login page.",
			"closeSourceBranch": true,
			"reviewers":         []string{"{reviewer-uuid}"},
		}, jsonResponse(http.StatusCreated, createdPullRequestResponse))

		require.NoError(t, action.Execute(fixture.Context))

		require.Len(t, fixture.HTTP.Requests, 1)
		request := fixture.HTTP.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "https://api.bitbucket.org/2.0/repositories/superplane/web/pullrequests", request.URL.String())

		body := requestBody(t, request)
		assert.Equal(t, "Add login page", body["title"])
		assert.Equal(t, "Adds the login page.", body["description"])
		assert.Equal(t, true, body["close_source_branch"])
		assert.Equal(t, "feature/login-page", body["source"].(map[string]any)["branch"].(map[string]any)["name"])
		assert.Equal(t, "main", body["destination"].(map[string]any)["branch"].(map[string]any)["name"])

		reviewers := body["reviewers"].([]any)
		require.Len(t, reviewers, 1)
		assert.Equal(t, "{reviewer-uuid}", reviewers[0].(map[string]any)["uuid"])

		assert.True(t, fixture.ExecutionState.Passed)
		assert.Equal(t, "bitbucket.pullRequest", fixture.ExecutionState.Type)
		require.Len(t, fixture.ExecutionState.Payloads, 1)
		pullRequest := emittedPayload[*PullRequest](t, fixture.ExecutionState, 0)
		assert.Equal(t, 42, pullRequest.ID)
		assert.Equal(t, "OPEN", pullRequest.State)
		assert.Equal(t, "https://bitbucket.org/superplane/web/pull-requests/42", pullRequest.Links.HTML.Href)
	})

	// Bitbucket falls back to the repository main branch when no destination is sent,
	// so an empty target branch must stay out of the payload entirely.
	t.Run("empty target branch is omitted from the payload", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository":   testRepositoryName,
			"sourceBranch": "feature/login-page",
			"title":        "Add login page",
		}, jsonResponse(http.StatusCreated, createdPullRequestResponse))

		require.NoError(t, action.Execute(fixture.Context))

		body := requestBody(t, fixture.HTTP.Requests[0])
		_, present := body["destination"]
		assert.False(t, present)
	})

	// Branch values commonly come from a push event, where the ref is a full refs/heads path.
	t.Run("a full ref is accepted as a branch name", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository":   testRepositoryName,
			"sourceBranch": "refs/heads/feature/login-page",
			"targetBranch": "refs/heads/main",
			"title":        "Add login page",
		}, jsonResponse(http.StatusCreated, createdPullRequestResponse))

		require.NoError(t, action.Execute(fixture.Context))

		body := requestBody(t, fixture.HTTP.Requests[0])
		assert.Equal(t, "feature/login-page", body["source"].(map[string]any)["branch"].(map[string]any)["name"])
		assert.Equal(t, "main", body["destination"].(map[string]any)["branch"].(map[string]any)["name"])
	})

	t.Run("API error is reported", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository":   testRepositoryName,
			"sourceBranch": "feature/login-page",
			"title":        "Add login page",
		}, jsonResponse(http.StatusBadRequest, `{"error":{"message":"source branch does not exist"}}`))

		err := action.Execute(fixture.Context)

		require.ErrorContains(t, err, "failed to create pull request")
		require.ErrorContains(t, err, "source branch does not exist")
	})

	t.Run("integration without workspace metadata is reported", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository":   testRepositoryName,
			"sourceBranch": "feature/login-page",
			"title":        "Add login page",
		})

		fixture.Context.Integration = &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "token"},
			Metadata:      Metadata{AuthType: AuthTypeWorkspaceAccessToken},
		}

		err := action.Execute(fixture.Context)

		require.ErrorContains(t, err, "integration is missing workspace metadata")
	})
}

func Test__NormalizeBranch(t *testing.T) {
	assert.Equal(t, "main", normalizeBranch("main"))
	assert.Equal(t, "main", normalizeBranch("refs/heads/main"))
	assert.Equal(t, "main", normalizeBranch("  main  "))
	assert.Equal(t, "release/2026-08", normalizeBranch("refs/heads/release/2026-08"))
	assert.Empty(t, normalizeBranch(""))
}

func Test__AccountRefs(t *testing.T) {
	t.Run("blank entries are dropped", func(t *testing.T) {
		refs := accountRefs([]string{"{a}", "", "  ", "{b}"})
		require.Len(t, refs, 2)
		assert.Equal(t, "{a}", refs[0].UUID)
		assert.Equal(t, "{b}", refs[1].UUID)
	})

	// A nil slice keeps the key out of the JSON payload, which is what "do not touch
	// the reviewers" means to the Bitbucket API.
	t.Run("no accounts yields nil", func(t *testing.T) {
		assert.Nil(t, accountRefs(nil))
		assert.Nil(t, accountRefs([]string{"", " "}))
	})
}
