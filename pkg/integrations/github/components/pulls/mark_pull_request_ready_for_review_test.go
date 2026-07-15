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

func draftPullRequestResponse() *http.Response {
	return mocks.GitHubResponse(http.StatusOK, `{
		"id": 1234567890,
		"node_id": "PR_kwDOABCD12MAAAABCDEFGH",
		"number": 42,
		"title": "Add new feature",
		"state": "open",
		"draft": true,
		"html_url": "https://github.com/testhq/hello/pull/42"
	}`)
}

func readyPullRequestResponse() *http.Response {
	return mocks.GitHubResponse(http.StatusOK, `{
		"id": 1234567890,
		"node_id": "PR_kwDOABCD12MAAAABCDEFGH",
		"number": 42,
		"title": "Add new feature",
		"state": "open",
		"draft": false,
		"html_url": "https://github.com/testhq/hello/pull/42"
	}`)
}

func Test__MarkPullRequestReadyForReview__Setup(t *testing.T) {
	component := MarkPullRequestReadyForReview{}

	validConfig := func(overrides map[string]any) map[string]any {
		config := map[string]any{
			"repository": "hello",
			"pullNumber": "42",
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

	t.Run("pull request number is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"pullNumber": ""}),
		})

		require.ErrorContains(t, err, "pull request number is required")
	})

	t.Run("literal pull request number must be positive", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"pullNumber": "0"}),
		})

		require.ErrorContains(t, err, "pull request number must be a positive integer")
	})

	t.Run("valid configuration is accepted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"id": 123456,
					"name": "hello",
					"html_url": "https://github.com/testhq/hello"
				}`),
			},
		}

		err := component.Setup(core.SetupContext{
			Integration:   mocks.IntegrationContextForNewSetupFlow(),
			HTTP:          httpCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(nil),
		})

		require.NoError(t, err)
	})

	t.Run("expression pull request number is accepted at setup", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"id": 123456,
					"name": "hello",
					"html_url": "https://github.com/testhq/hello"
				}`),
			},
		}

		err := component.Setup(core.SetupContext{
			Integration: mocks.IntegrationContextForNewSetupFlow(),
			HTTP:        httpCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{
				"pullNumber": `{{$["github.onPullRequest"].data.pull_request.number}}`,
			}),
		})

		require.NoError(t, err)
	})
}

func Test__MarkPullRequestReadyForReview__Execute(t *testing.T) {
	component := MarkPullRequestReadyForReview{}

	t.Run("fails when configuration decode fails", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  "not a map",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("repository is required", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "",
				"pullNumber": "42",
			},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("pull request number must be positive", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"pullNumber": "-1",
			},
		})

		require.ErrorContains(t, err, "pull request number must be a positive integer")
	})

	t.Run("marks a draft pull request ready for review", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				draftPullRequestResponse(),
				mocks.GitHubResponse(http.StatusOK, `{
					"data": {
						"markPullRequestReadyForReview": {
							"pullRequest": {
								"number": 42,
								"isDraft": false
							}
						}
					}
				}`),
				readyPullRequestResponse(),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository": "hello",
				"pullNumber": "42",
			},
		})

		require.NoError(t, err)
		require.True(t, executionState.Passed)
		require.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		require.Equal(t, "github.pullRequest", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		require.Len(t, httpCtx.Requests, 3)

		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, "/repos/testhq/hello/pulls/42", httpCtx.Requests[0].URL.Path)

		mutation := httpCtx.Requests[1]
		assert.Equal(t, http.MethodPost, mutation.Method)
		assert.Equal(t, "/graphql", mutation.URL.Path)

		mutationBody := readJSONBody(t, mutation)
		assert.Contains(t, mutationBody["query"], "markPullRequestReadyForReview")
		assert.Equal(t, map[string]any{
			"pullRequestId": "PR_kwDOABCD12MAAAABCDEFGH",
		}, mutationBody["variables"])

		assert.Equal(t, http.MethodGet, httpCtx.Requests[2].Method)
		assert.Equal(t, "/repos/testhq/hello/pulls/42", httpCtx.Requests[2].URL.Path)

		payload := executionState.Payloads[0].(map[string]any)
		pullRequest := payload["data"].(*github.PullRequest)
		assert.Equal(t, 42, pullRequest.GetNumber())
		assert.Equal(t, "Add new feature", pullRequest.GetTitle())
		assert.False(t, pullRequest.GetDraft())
	})

	t.Run("pull request that is already ready is emitted without the mutation", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				readyPullRequestResponse(),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository": "hello",
				"pullNumber": "42",
			},
		})

		require.NoError(t, err)
		require.True(t, executionState.Passed)
		require.Equal(t, "github.pullRequest", executionState.Type)
		require.Len(t, httpCtx.Requests, 1)

		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, "/repos/testhq/hello/pulls/42", httpCtx.Requests[0].URL.Path)

		payload := executionState.Payloads[0].(map[string]any)
		pullRequest := payload["data"].(*github.PullRequest)
		assert.False(t, pullRequest.GetDraft())
	})

	t.Run("fails when the pull request cannot be fetched", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusNotFound, `{"message": "Not Found"}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"pullNumber": "42",
			},
		})

		require.ErrorContains(t, err, "failed to get pull request")
		require.ErrorContains(t, err, "Not Found")
	})

	t.Run("surfaces GraphQL errors returned on a 200 response", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				draftPullRequestResponse(),
				mocks.GitHubResponse(http.StatusOK, `{
					"errors": [
						{"message": "Resource not accessible by integration"}
					]
				}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"pullNumber": "42",
			},
		})

		require.ErrorContains(t, err, "failed to mark pull request ready for review")
		require.ErrorContains(t, err, "Resource not accessible by integration")
	})
}
