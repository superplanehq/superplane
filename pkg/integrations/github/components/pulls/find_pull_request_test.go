package pulls

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

func matchingPullRequestResponse() *http.Response {
	return mocks.GitHubResponse(http.StatusOK, `[
		{
			"id": 1234567890,
			"number": 42,
			"title": "Add new feature",
			"state": "open",
			"draft": false,
			"html_url": "https://github.com/testhq/hello/pull/42"
		}
	]`)
}

func noPullRequestsResponse() *http.Response {
	return mocks.GitHubResponse(http.StatusOK, `[]`)
}

func repositoryResponse() *http.Response {
	return mocks.GitHubResponse(http.StatusOK, `{
		"id": 123456,
		"name": "hello",
		"html_url": "https://github.com/testhq/hello"
	}`)
}

func Test__FindPullRequest__Setup(t *testing.T) {
	component := FindPullRequest{}

	validConfig := func(overrides map[string]any) map[string]any {
		config := map[string]any{
			"repository": "hello",
			"head":       "feature",
		}
		for key, value := range overrides {
			config[key] = value
		}
		return config
	}

	t.Run("repository is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"repository": ""}),
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("head branch is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"head": ""}),
		})

		require.ErrorContains(t, err, "head branch is required")
	})

	t.Run("invalid state is rejected", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"state": "merged"}),
		})

		require.ErrorContains(t, err, "state must be one of: open, closed, all")
	})

	t.Run("valid configuration is accepted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{repositoryResponse()},
		}

		err := component.Setup(core.SetupContext{
			Integration:   mocks.IntegrationContextForNewSetupFlow(),
			HTTP:          httpCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(nil),
		})

		require.NoError(t, err)
	})

	t.Run("expression state is accepted at setup", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{repositoryResponse()},
		}

		err := component.Setup(core.SetupContext{
			Integration: mocks.IntegrationContextForNewSetupFlow(),
			HTTP:        httpCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{
				"state": `{{ task().pr_state }}`,
			}),
		})

		require.NoError(t, err)
	})
}

func Test__FindPullRequest__Execute(t *testing.T) {
	component := FindPullRequest{}

	t.Run("repository is required", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "",
				"head":       "feature",
			},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("head branch is required", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"head":       "",
			},
		})

		require.ErrorContains(t, err, "head branch is required")
	})

	t.Run("invalid state is rejected", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"head":       "feature",
				"state":      "merged",
			},
		})

		require.ErrorContains(t, err, "state must be one of: open, closed, all")
	})

	t.Run("found: emits the first matching pull request", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{matchingPullRequestResponse()},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository": "hello",
				"head":       "feature",
			},
		})

		require.NoError(t, err)
		require.True(t, executionState.Passed)
		require.Len(t, httpCtx.Requests, 1)

		request := httpCtx.Requests[0]
		assert.Equal(t, "/repos/testhq/hello/pulls", request.URL.Path)
		assert.Equal(t, "testhq:feature", request.URL.Query().Get("head"))
		assert.Equal(t, "open", request.URL.Query().Get("state"))
		assert.Empty(t, request.URL.Query().Get("base"))

		assert.Equal(t, FindPullRequestFoundChannel, executionState.Channel)
		assert.Equal(t, "github.pullRequest", executionState.Type)

		payload := executionState.Payloads[0].(map[string]any)
		pullRequest := payload["data"].(*github.PullRequest)
		assert.Equal(t, 42, pullRequest.GetNumber())
	})

	t.Run("notFound: emits when no pull request matches", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{noPullRequestsResponse()},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository": "hello",
				"head":       "feature",
			},
		})

		require.NoError(t, err)
		require.True(t, executionState.Passed)
		assert.Equal(t, FindPullRequestNotFoundChannel, executionState.Channel)
		assert.Equal(t, "github.findPullRequest.notFound", executionState.Type)
	})

	t.Run("base filtering: base is included in the request query when set", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{matchingPullRequestResponse()},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository": "hello",
				"head":       "feature",
				"base":       "main",
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "main", httpCtx.Requests[0].URL.Query().Get("base"))
	})

	t.Run("state options: state is passed through to the query", func(t *testing.T) {
		for _, state := range []string{"closed", "all"} {
			httpCtx := &contexts.HTTPContext{
				Responses: []*http.Response{matchingPullRequestResponse()},
			}

			err := component.Execute(core.ExecutionContext{
				Integration:    mocks.IntegrationContextForNewSetupFlow(),
				HTTP:           httpCtx,
				ExecutionState: &contexts.ExecutionStateContext{},
				Configuration: map[string]any{
					"repository": "hello",
					"head":       "feature",
					"state":      state,
				},
			})

			require.NoError(t, err)
			require.Len(t, httpCtx.Requests, 1)
			assert.Equal(t, state, httpCtx.Requests[0].URL.Query().Get("state"))
		}
	})

	t.Run("a head that already carries an owner prefix is passed through unchanged", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{matchingPullRequestResponse()},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"head":       "someoneelse:feature",
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "someoneelse:feature", httpCtx.Requests[0].URL.Query().Get("head"))
	})

	t.Run("fails when the GitHub API returns an error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusUnprocessableEntity, `{"message": "Validation Failed"}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"head":       "feature",
			},
		})

		require.ErrorContains(t, err, "failed to list pull requests")
		require.ErrorContains(t, err, "Validation Failed")
	})
}
