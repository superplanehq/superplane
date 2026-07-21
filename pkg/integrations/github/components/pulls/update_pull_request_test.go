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

func updatedPullRequestResponse() *http.Response {
	return mocks.GitHubResponse(http.StatusOK, `{
		"id": 1234567890,
		"number": 42,
		"title": "Updated title",
		"state": "closed",
		"draft": false,
		"html_url": "https://github.com/testhq/hello/pull/42"
	}`)
}

func Test__UpdatePullRequest__Setup(t *testing.T) {
	component := UpdatePullRequest{}

	validConfig := func(overrides map[string]any) map[string]any {
		config := map[string]any{
			"repository": "hello",
			"pullNumber": "42",
			"title":      "Updated title",
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

	t.Run("state must be open or closed", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"state": "merged"}),
		})

		require.ErrorContains(t, err, "state must be one of: open, closed")
	})

	t.Run("at least one field to update is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"pullNumber": "42",
			},
		})

		require.ErrorContains(t, err, "at least one of title, body, state, base, assignees, or labels is required")
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

	t.Run("labels alone are a valid configuration", func(t *testing.T) {
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
			Configuration: map[string]any{
				"repository": "hello",
				"pullNumber": "42",
				"labels":     []string{"bug"},
			},
		})

		require.NoError(t, err)
	})
}

func Test__UpdatePullRequest__Execute(t *testing.T) {
	component := UpdatePullRequest{}

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
				"title":      "Updated title",
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
				"title":      "Updated title",
			},
		})

		require.ErrorContains(t, err, "pull request number must be a positive integer")
	})

	t.Run("state must be open or closed", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"pullNumber": "42",
				"state":      "merged",
			},
		})

		require.ErrorContains(t, err, "state must be one of: open, closed")
	})

	t.Run("at least one field to update is required", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"pullNumber": "42",
			},
		})

		require.ErrorContains(t, err, "at least one of title, body, state, base, assignees, or labels is required")
	})

	t.Run("updates title, body, and state through the Pulls API", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				updatedPullRequestResponse(),
				updatedPullRequestResponse(),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository": "hello",
				"pullNumber": "42",
				"title":      "Updated title",
				"body":       "Updated body",
				"state":      "closed",
			},
		})

		require.NoError(t, err)
		require.True(t, executionState.Passed)
		require.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		require.Equal(t, "github.pullRequest", executionState.Type)
		require.Len(t, httpCtx.Requests, 2)

		editRequest := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPatch, editRequest.Method)
		assert.Equal(t, "/repos/testhq/hello/pulls/42", editRequest.URL.Path)

		requestBody := readJSONBody(t, editRequest)
		assert.Equal(t, "Updated title", requestBody["title"])
		assert.Equal(t, "Updated body", requestBody["body"])
		assert.Equal(t, "closed", requestBody["state"])

		getRequest := httpCtx.Requests[1]
		assert.Equal(t, http.MethodGet, getRequest.Method)
		assert.Equal(t, "/repos/testhq/hello/pulls/42", getRequest.URL.Path)

		payload := executionState.Payloads[0].(map[string]any)
		pullRequest := payload["data"].(*github.PullRequest)
		assert.Equal(t, "Updated title", pullRequest.GetTitle())
		assert.Equal(t, "closed", pullRequest.GetState())
	})

	t.Run("retargets the base branch through the Pulls API", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				updatedPullRequestResponse(),
				updatedPullRequestResponse(),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository": "hello",
				"pullNumber": "42",
				"base":       "release",
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 2)

		editRequest := httpCtx.Requests[0]
		requestBody := readJSONBody(t, editRequest)
		assert.Equal(t, "release", requestBody["base"])
	})

	t.Run("updates labels and assignees through the Issues API", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"id": 101,
					"number": 42,
					"title": "Add new feature",
					"state": "open",
					"labels": [{"name": "bug"}],
					"assignees": [{"login": "octocat"}]
				}`),
				updatedPullRequestResponse(),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository": "hello",
				"pullNumber": "42",
				"labels":     []string{"bug"},
				"assignees":  []string{"octocat"},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 2)

		issueRequest := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPatch, issueRequest.Method)
		assert.Equal(t, "/repos/testhq/hello/issues/42", issueRequest.URL.Path)

		requestBody := readJSONBody(t, issueRequest)
		assert.Equal(t, []any{"bug"}, requestBody["labels"])
		assert.Equal(t, []any{"octocat"}, requestBody["assignees"])

		getRequest := httpCtx.Requests[1]
		assert.Equal(t, http.MethodGet, getRequest.Method)
		assert.Equal(t, "/repos/testhq/hello/pulls/42", getRequest.URL.Path)
	})

	t.Run("only calls the Pulls API when no labels or assignees are set", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				updatedPullRequestResponse(),
				updatedPullRequestResponse(),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"pullNumber": "42",
				"title":      "Updated title",
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, "/repos/testhq/hello/pulls/42", httpCtx.Requests[0].URL.Path)
		assert.Equal(t, "/repos/testhq/hello/pulls/42", httpCtx.Requests[1].URL.Path)
	})

	t.Run("only calls the Issues API when no title, body, state, or base are set", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{"id": 101, "number": 42}`),
				updatedPullRequestResponse(),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"pullNumber": "42",
				"labels":     []string{"bug"},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, "/repos/testhq/hello/issues/42", httpCtx.Requests[0].URL.Path)
		assert.Equal(t, "/repos/testhq/hello/pulls/42", httpCtx.Requests[1].URL.Path)
	})

	t.Run("fails when the Pulls API returns an error", func(t *testing.T) {
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
				"pullNumber": "42",
				"title":      "Updated title",
			},
		})

		require.ErrorContains(t, err, "failed to update pull request")
		require.ErrorContains(t, err, "Validation Failed")
	})

	t.Run("fails when the Issues API returns an error", func(t *testing.T) {
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
				"pullNumber": "42",
				"labels":     []string{"bug"},
			},
		})

		require.ErrorContains(t, err, "failed to update pull request labels/assignees")
		require.ErrorContains(t, err, "Validation Failed")
	})

	t.Run("fails when the pull request cannot be re-fetched", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				updatedPullRequestResponse(),
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
				"title":      "Updated title",
			},
		})

		require.ErrorContains(t, err, "failed to get pull request")
		require.ErrorContains(t, err, "Not Found")
	})
}
