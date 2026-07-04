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

func Test__AddPullRequestReviewers__Setup(t *testing.T) {
	component := AddPullRequestReviewers{}

	validConfig := func(overrides map[string]any) map[string]any {
		config := map[string]any{
			"repository": "hello",
			"pullNumber": "42",
			"reviewers":  []string{"octocat"},
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

	t.Run("at least one reviewer or team reviewer is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{
				"reviewers":     []string{},
				"teamReviewers": []string{},
			}),
		})

		require.ErrorContains(t, err, "at least one reviewer or team reviewer is required")
	})

	t.Run("team reviewers alone are accepted", func(t *testing.T) {
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
				"reviewers":     []string{},
				"teamReviewers": []string{"justice-league"},
			}),
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

func Test__AddPullRequestReviewers__Execute(t *testing.T) {
	component := AddPullRequestReviewers{}

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
				"reviewers":  []string{"octocat"},
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
				"reviewers":  []string{"octocat"},
			},
		})

		require.ErrorContains(t, err, "pull request number must be a positive integer")
	})

	t.Run("at least one reviewer or team reviewer is required", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository":    "hello",
				"pullNumber":    "42",
				"reviewers":     []string{},
				"teamReviewers": []string{},
			},
		})

		require.ErrorContains(t, err, "at least one reviewer or team reviewer is required")
	})

	t.Run("emits the updated pull request", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusCreated, `{
					"id": 1234567890,
					"number": 42,
					"title": "Add new feature",
					"state": "open",
					"html_url": "https://github.com/testhq/hello/pull/42",
					"requested_reviewers": [
						{
							"login": "octocat",
							"id": 1,
							"html_url": "https://github.com/octocat"
						}
					],
					"requested_teams": [
						{
							"slug": "justice-league",
							"id": 1,
							"html_url": "https://github.com/orgs/testhq/teams/justice-league"
						}
					]
				}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository":    "hello",
				"pullNumber":    "42",
				"reviewers":     []string{"@octocat"},
				"teamReviewers": []string{"justice-league"},
			},
		})

		require.NoError(t, err)
		require.True(t, executionState.Passed)
		require.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		require.Equal(t, "github.pullRequest", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		require.Len(t, httpCtx.Requests, 1)

		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "/repos/testhq/hello/pulls/42/requested_reviewers", request.URL.Path)

		requestBody := readJSONBody(t, request)
		assert.Equal(t, []any{"octocat"}, requestBody["reviewers"])
		assert.Equal(t, []any{"justice-league"}, requestBody["team_reviewers"])

		payload := executionState.Payloads[0].(map[string]any)
		pullRequest := payload["data"].(*github.PullRequest)
		assert.Equal(t, 42, pullRequest.GetNumber())
		assert.Equal(t, "Add new feature", pullRequest.GetTitle())
		assert.Equal(t, "open", pullRequest.GetState())
	})
}
