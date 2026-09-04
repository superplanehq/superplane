package bitbucket

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const publishedCommitStatusResponse = `{
  "key": "superplane-deploy",
  "name": "SuperPlane deploy",
  "state": "SUCCESSFUL",
  "url": "https://app.superplane.com/runs/8f2c1a44",
  "refname": "main"
}`

func Test__PublishCommitStatus__Setup(t *testing.T) {
	action := &PublishCommitStatus{}

	base := func() map[string]any {
		return map[string]any{
			"repository": testRepositoryName,
			"commit":     "9fec847784abb10b2fa567ee63b85bd238955d0e",
			"key":        "superplane-deploy",
			"state":      StateSuccessful,
		}
	}

	t.Run("commit is required", func(t *testing.T) {
		config := base()
		config["commit"] = ""

		require.ErrorContains(t, action.Setup(newSetupContext(config)), "commit is required")
	})

	t.Run("key is required", func(t *testing.T) {
		config := base()
		config["key"] = ""

		require.ErrorContains(t, action.Setup(newSetupContext(config)), "key is required")
	})

	t.Run("state is required", func(t *testing.T) {
		config := base()
		config["state"] = ""

		require.ErrorContains(t, action.Setup(newSetupContext(config)), "state is required")
	})

	t.Run("unsupported state is rejected", func(t *testing.T) {
		config := base()
		config["state"] = "GREEN"

		require.ErrorContains(t, action.Setup(newSetupContext(config)), `unsupported build state "GREEN"`)
	})

	t.Run("valid configuration is accepted", func(t *testing.T) {
		require.NoError(t, action.Setup(newSetupContext(base())))
	})
}

func Test__PublishCommitStatus__Execute(t *testing.T) {
	action := &PublishCommitStatus{}

	t.Run("status is published and emitted", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository":  testRepositoryName,
			"commit":      "9fec847784abb10b2fa567ee63b85bd238955d0e",
			"key":         "superplane-deploy",
			"name":        "SuperPlane deploy",
			"state":       StateSuccessful,
			"url":         "https://app.superplane.com/runs/8f2c1a44",
			"description": "Deployed to production",
		}, jsonResponse(http.StatusCreated, publishedCommitStatusResponse))

		require.NoError(t, action.Execute(fixture.Context))

		require.Len(t, fixture.HTTP.Requests, 1)
		request := fixture.HTTP.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(
			t,
			"https://api.bitbucket.org/2.0/repositories/superplane/web/commit/9fec847784abb10b2fa567ee63b85bd238955d0e/statuses/build",
			request.URL.String(),
		)

		body := requestBody(t, request)
		assert.Equal(t, "superplane-deploy", body["key"])
		assert.Equal(t, "SuperPlane deploy", body["name"])
		assert.Equal(t, StateSuccessful, body["state"])
		assert.Equal(t, "Deployed to production", body["description"])

		status := emittedPayload[*CommitStatus](t, fixture.ExecutionState, 0)
		assert.Equal(t, "superplane-deploy", status.Key)
		assert.Equal(t, StateSuccessful, status.State)
		assert.Equal(t, "bitbucket.commitStatus", fixture.ExecutionState.Type)
	})

	t.Run("name defaults to the key", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository": testRepositoryName,
			"commit":     "9fec847784abb10b2fa567ee63b85bd238955d0e",
			"key":        "superplane-deploy",
			"state":      StateInProgress,
		}, jsonResponse(http.StatusCreated, publishedCommitStatusResponse))

		require.NoError(t, action.Execute(fixture.Context))

		body := requestBody(t, fixture.HTTP.Requests[0])
		assert.Equal(t, "superplane-deploy", body["name"])
	})

	// The commit usually arrives from an expression, which can resolve to whitespace.
	t.Run("a blank commit is reported before any request is made", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository": testRepositoryName,
			"commit":     "   ",
			"key":        "superplane-deploy",
			"state":      StateSuccessful,
		})

		err := action.Execute(fixture.Context)

		require.ErrorContains(t, err, "commit is required")
		assert.Empty(t, fixture.HTTP.Requests)
	})

	t.Run("API error is reported", func(t *testing.T) {
		fixture := newExecutionFixture(map[string]any{
			"repository": testRepositoryName,
			"commit":     "9fec847784abb10b2fa567ee63b85bd238955d0e",
			"key":        "superplane-deploy",
			"state":      StateSuccessful,
		}, jsonResponse(http.StatusForbidden, `{"error":{"message":"write access required"}}`))

		err := action.Execute(fixture.Context)

		require.ErrorContains(t, err, "failed to publish commit status")
		require.ErrorContains(t, err, "write access required")
	})
}
