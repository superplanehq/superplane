package pulls_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration/expressionvalidation"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/components/pulls"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func reviewPayloadWithThreads() map[string]any {
	return map[string]any{
		"data": map[string]any{
			"repository": map[string]any{"full_name": "testhq/hello"},
			"review_comments": []any{
				map[string]any{
					"id":                         float64(101),
					"body":                       "please rename this",
					"pull_request_review_thread": map[string]any{"id": "PRRT_a"},
				},
				map[string]any{
					"id":                         float64(102),
					"body":                       "and drop this line",
					"pull_request_review_thread": map[string]any{"id": "PRRT_b"},
				},
				map[string]any{
					"id":   float64(103),
					"body": "a conversation comment without a review thread",
				},
			},
		},
	}
}

func Test__ResolveReviewThread__ThreadIDsExpression__IsValid(t *testing.T) {
	t.Run("passes node expression validation", func(t *testing.T) {
		err := expressionvalidation.ValidateBareExpression(pulls.ThreadIDsExpression, map[string]struct{}{})

		require.NoError(t, err)
	})
}

func Test__ResolveReviewThread__ThreadIDsExpression__ReturnsEveryThread(t *testing.T) {
	expressions := workercontexts.NewExpressionContext(
		workercontexts.NewNodeConfigurationBuilder(nil, uuid.New()).
			WithNodeID("resolve-pr-review-thread").
			WithRootPayload(reviewPayloadWithThreads()),
	)

	result, err := expressions.Run(pulls.ThreadIDsExpression)
	require.NoError(t, err)
	require.Equal(t, []any{"PRRT_a", "PRRT_b", ""}, result)
}

func Test__ResolveReviewThread__Execute__ResolvesEveryThreadOfAReview(t *testing.T) {
	component := pulls.ResolveReviewThread{}

	expressions := workercontexts.NewExpressionContext(
		workercontexts.NewNodeConfigurationBuilder(nil, uuid.New()).
			WithNodeID("resolve-pr-review-thread").
			WithRootPayload(reviewPayloadWithThreads()),
	)

	state := &contexts.ExecutionStateContext{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			resolveReviewThreadResponse("PRRT_a"),
			resolveReviewThreadResponse("PRRT_b"),
		},
	}

	err := component.Execute(core.ExecutionContext{
		Integration:    mocks.IntegrationContextForNewSetupFlow(),
		HTTP:           httpCtx,
		Expressions:    expressions,
		ExecutionState: state,
		Configuration: map[string]any{
			"repository":          "hello",
			"threadIdsExpression": pulls.ThreadIDsExpression,
		},
	})

	require.NoError(t, err)
	require.True(t, state.Passed)
	require.Equal(t, core.DefaultOutputChannel.Name, state.Channel)
	require.Equal(t, "github.reviewThread", state.Type)
	require.Len(t, state.Payloads, 2)
	require.Len(t, httpCtx.Requests, 2)

	for i, threadID := range []string{"PRRT_a", "PRRT_b"} {
		request := httpCtx.Requests[i]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "/graphql", request.URL.Path)

		payload, ok := state.Payloads[i].(map[string]any)
		require.True(t, ok)
		thread, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, threadID, thread["id"])
		assert.Equal(t, true, thread["isResolved"])
	}
}

func resolveReviewThreadResponse(threadID string) *http.Response {
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
