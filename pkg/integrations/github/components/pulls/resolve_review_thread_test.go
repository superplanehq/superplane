package pulls

import (
	"io"
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

		require.ErrorContains(t, err, "thread ID or thread IDs expression is required")
	})

	t.Run("thread IDs expression is accepted at setup", func(t *testing.T) {
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
			Configuration: validConfig(map[string]any{"threadId": "", "threadIdsExpression": ThreadIDsExpression}),
		})

		require.NoError(t, err)
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

		require.ErrorContains(t, err, "failed to resolve any review threads")
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

	t.Run("resolves every thread the expression returns", func(t *testing.T) {
		state := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				resolveThreadResponse("PRRT_a"),
				resolveThreadResponse("PRRT_b"),
				resolveThreadResponse("PRRT_c"),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration: mocks.IntegrationContextForNewSetupFlow(),
			HTTP:        httpCtx,
			Expressions: &contexts.ExpressionContext{
				Output: []any{"PRRT_a", "PRRT_b", "PRRT_c"},
			},
			ExecutionState: state,
			Configuration: map[string]any{
				"repository":          "hello",
				"threadIdsExpression": ThreadIDsExpression,
			},
		})

		require.NoError(t, err)
		require.True(t, state.Passed)
		require.Equal(t, core.DefaultOutputChannel.Name, state.Channel)
		require.Equal(t, "github.reviewThread", state.Type)
		require.Len(t, state.Payloads, 3)
		require.Len(t, httpCtx.Requests, 3)

		for i, threadID := range []string{"PRRT_a", "PRRT_b", "PRRT_c"} {
			request := httpCtx.Requests[i]
			assert.Equal(t, http.MethodPost, request.Method)
			assert.Equal(t, "/graphql", request.URL.Path)
			assert.Contains(t, requestBody(t, request), threadID)

			payload := state.Payloads[i].(map[string]any)
			thread := payload["data"].(map[string]any)
			assert.Equal(t, threadID, thread["id"])
			assert.Equal(t, true, thread["isResolved"])
		}
	})

	t.Run("drops blank and duplicate thread IDs from the expression", func(t *testing.T) {
		state := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				resolveThreadResponse("PRRT_a"),
				resolveThreadResponse("PRRT_b"),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration: mocks.IntegrationContextForNewSetupFlow(),
			HTTP:        httpCtx,
			Expressions: &contexts.ExpressionContext{
				Output: []any{"PRRT_a", "", "  ", "PRRT_a", "PRRT_b", nil, 42},
			},
			ExecutionState: state,
			Configuration: map[string]any{
				"repository":          "hello",
				"threadIdsExpression": ThreadIDsExpression,
			},
		})

		require.NoError(t, err)
		require.True(t, state.Passed)
		require.Len(t, state.Payloads, 2)
		require.Len(t, httpCtx.Requests, 2)
	})

	t.Run("emits nothing when the expression returns no thread", func(t *testing.T) {
		state := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			Expressions:    &contexts.ExpressionContext{Output: []any{}},
			ExecutionState: state,
			Configuration: map[string]any{
				"repository":          "hello",
				"threadIdsExpression": ThreadIDsExpression,
			},
		})

		require.NoError(t, err)
		require.True(t, state.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, state.Channel)
		assert.Equal(t, "github.reviewThread", state.Type)
		assert.Empty(t, state.Payloads)
	})

	t.Run("thread IDs expression takes precedence over thread ID", func(t *testing.T) {
		state := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{resolveThreadResponse("PRRT_b")},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			Expressions:    &contexts.ExpressionContext{Output: []any{"PRRT_b"}},
			ExecutionState: state,
			Configuration: map[string]any{
				"repository":          "hello",
				"threadId":            "PRRT_a",
				"threadIdsExpression": ThreadIDsExpression,
			},
		})

		require.NoError(t, err)
		require.Len(t, state.Payloads, 1)
		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, requestBody(t, httpCtx.Requests[0]), "PRRT_b")
	})

	t.Run("resolves the remaining threads when one thread fails", func(t *testing.T) {
		state := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{"errors": [{"message": "Resource not accessible by integration"}]}`),
				resolveThreadResponse("PRRT_b"),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			Expressions:    &contexts.ExpressionContext{Output: []any{"PRRT_a", "PRRT_b"}},
			ExecutionState: state,
			Configuration: map[string]any{
				"repository":          "hello",
				"threadIdsExpression": ThreadIDsExpression,
			},
		})

		require.NoError(t, err)
		require.True(t, state.Passed)
		require.Len(t, state.Payloads, 1)

		payload := state.Payloads[0].(map[string]any)
		thread := payload["data"].(map[string]any)
		assert.Equal(t, "PRRT_b", thread["id"])
	})

	t.Run("fails when the thread IDs expression does not evaluate to a list", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			Expressions:    &contexts.ExpressionContext{Output: "PRRT_a"},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository":          "hello",
				"threadIdsExpression": ThreadIDsExpression,
			},
		})

		require.ErrorContains(t, err, "thread IDs expression must evaluate to a list, got string")
	})

	t.Run("fails when the thread IDs expression cannot be evaluated", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration: mocks.IntegrationContextForNewSetupFlow(),
			Expressions: &contexts.ExpressionContext{
				Error: assert.AnError,
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository":          "hello",
				"threadIdsExpression": ThreadIDsExpression,
			},
		})

		require.ErrorContains(t, err, "failed to evaluate thread IDs expression")
	})
}

func Test__ResolveReviewThread__HandleWebhook(t *testing.T) {
	component := ResolveReviewThread{}

	code, body, err := component.HandleWebhook(core.WebhookRequestContext{})
	require.NoError(t, err)
	assert.Equal(t, 200, code)
	assert.Nil(t, body)
}

func resolveThreadResponse(threadID string) *http.Response {
	return mocks.GitHubResponse(http.StatusOK, `{
		"data": {
			"resolveReviewThread": {
				"thread": {
					"id": "`+threadID+`",
					"isResolved": true
				}
			}
		}
	}`)
}

func requestBody(t *testing.T, request *http.Request) string {
	t.Helper()

	if request.Body == nil {
		return ""
	}

	body, err := io.ReadAll(request.Body)
	require.NoError(t, err)

	return string(body)
}
