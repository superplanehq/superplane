package pulls

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func Test__ResolveReviewThread__Setup(t *testing.T) {
	component := ResolveReviewThread{}

	validConfig := func(overrides map[string]any) map[string]any {
		config := map[string]any{
			"repository": "hello",
			"threadId":   "PRRT_kwDOABCD12MAAAABCDEFGH",
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

	t.Run("thread ID is required when not an expression", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"threadId": ""}),
		})

		require.ErrorContains(t, err, "thread ID is required")
	})

	t.Run("expression thread ID is accepted at setup", func(t *testing.T) {
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
			Configuration: validConfig(map[string]any{"threadId": `{{ root().data.comment.pull_request_review_thread.id }}`}),
		})

		require.NoError(t, err)
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
}

func Test__ResolveReviewThread__Execute(t *testing.T) {
	component := ResolveReviewThread{}

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
				"threadId":   "PRRT_kwDOABCD12MAAAABCDEFGH",
			},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("emits nothing when thread ID is blank", func(t *testing.T) {
		state := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"id": 123456,
					"name": "hello",
					"html_url": "https://github.com/testhq/hello"
				}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: state,
			Configuration: map[string]any{
				"repository": "hello",
				"threadId":   "  ",
			},
		})

		require.NoError(t, err)
		require.True(t, state.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, state.Channel)
		assert.Equal(t, "github.reviewThread", state.Type)
		assert.Empty(t, state.Payloads)
	})

	t.Run("resolves a review thread via GraphQL", func(t *testing.T) {
		state := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"data": {
						"resolveReviewThread": {
							"thread": {
								"id": "PRRT_kwDOABCD12MAAAABCDEFGH",
								"isResolved": true
							}
						}
					}
				}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: state,
			Configuration: map[string]any{
				"repository": "hello",
				"threadId":   "PRRT_kwDOABCD12MAAAABCDEFGH",
			},
		})

		require.NoError(t, err)
		require.True(t, state.Passed)
		require.Equal(t, core.DefaultOutputChannel.Name, state.Channel)
		require.Equal(t, "github.reviewThread", state.Type)
		require.Len(t, state.Payloads, 1)
		require.Len(t, httpCtx.Requests, 1)

		mutation := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, mutation.Method)
		assert.Equal(t, "/graphql", mutation.URL.Path)

		payload := state.Payloads[0].(map[string]any)
		thread := payload["data"].(map[string]any)
		assert.Equal(t, "PRRT_kwDOABCD12MAAAABCDEFGH", thread["id"])
		assert.Equal(t, true, thread["isResolved"])
	})

	t.Run("surfaces GraphQL errors returned on a 200 response", func(t *testing.T) {
		state := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
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
			ExecutionState: state,
			Configuration: map[string]any{
				"repository": "hello",
				"threadId":   "PRRT_kwDOABCD12MAAAABCDEFGH",
			},
		})

		require.ErrorContains(t, err, "failed to resolve review thread")
		require.ErrorContains(t, err, "Resource not accessible by integration")
	})

	t.Run("fails when client initialization fails", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"threadId":   "PRRT_kwDOABCD12MAAAABCDEFGH",
			},
		})

		require.ErrorContains(t, err, "failed to initialize GitHub client")
	})
}

func Test__ResolveReviewThread__HandleWebhook(t *testing.T) {
	component := ResolveReviewThread{}

	code, body, err := component.HandleWebhook(core.WebhookRequestContext{})
	require.NoError(t, err)
	assert.Equal(t, 200, code)
	assert.Nil(t, body)
}
