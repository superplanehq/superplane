package pulls

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func Test__MergePullRequest__Setup(t *testing.T) {
	component := MergePullRequest{}

	validConfig := func(overrides map[string]any) map[string]any {
		config := map[string]any{
			"repository":  "hello",
			"pullNumber":  "42",
			"mergeMethod": "merge",
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

	t.Run("merge method must be supported", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"mergeMethod": "fast-forward"}),
		})

		require.ErrorContains(t, err, "merge method must be one of")
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

func Test__MergePullRequest__Execute(t *testing.T) {
	component := MergePullRequest{}

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

	t.Run("accepts numeric pull request number", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"sha": "0e98bc41ab56cee9ff17883607b56f96e7814c98",
					"merged": true,
					"message": "Pull Request successfully merged"
				}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"pullNumber": 42,
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "/repos/testhq/hello/pulls/42/merge", httpCtx.Requests[0].URL.Path)
	})

	t.Run("emits the merge result", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"sha": "0e98bc41ab56cee9ff17883607b56f96e7814c98",
					"merged": true,
					"message": "Pull Request successfully merged"
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
				"mergeMethod":   "squash",
				"sha":           "abc123",
				"commitTitle":   "Merge PR #42",
				"commitMessage": "Ship it",
			},
		})

		require.NoError(t, err)
		require.True(t, executionState.Passed)
		require.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		require.Equal(t, "github.pullRequestMerge", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		require.Len(t, httpCtx.Requests, 1)

		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPut, request.Method)
		assert.Equal(t, "/repos/testhq/hello/pulls/42/merge", request.URL.Path)

		requestBody := readJSONBody(t, request)
		assert.Equal(t, "Ship it", requestBody["commit_message"])
		assert.Equal(t, "Merge PR #42", requestBody["commit_title"])
		assert.Equal(t, "squash", requestBody["merge_method"])
		assert.Equal(t, "abc123", requestBody["sha"])

		payload := executionState.Payloads[0].(map[string]any)
		result := payload["data"].(*github.PullRequestMergeResult)
		assert.True(t, result.GetMerged())
		assert.Equal(t, "0e98bc41ab56cee9ff17883607b56f96e7814c98", result.GetSHA())
		assert.Equal(t, "Pull Request successfully merged", result.GetMessage())
	})

	t.Run("defaults to merge commit", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"sha": "0e98bc41ab56cee9ff17883607b56f96e7814c98",
					"merged": true,
					"message": "Pull Request successfully merged"
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

		require.NoError(t, err)
		requestBody := readJSONBody(t, httpCtx.Requests[0])
		assert.Equal(t, "merge", requestBody["merge_method"])
	})
}

func readJSONBody(t *testing.T, request *http.Request) map[string]any {
	t.Helper()

	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)

	var requestBody map[string]any
	require.NoError(t, json.Unmarshal(body, &requestBody))
	return requestBody
}
