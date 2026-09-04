package factory

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestListPullRequests_Execute(t *testing.T) {
	component := &ListPullRequests{}

	t.Run("lists pull requests for the configured repository", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{
			listResult: []*core.PullRequest{
				{ID: "pr-1", Repository: "acme/app", Number: 1, State: "open"},
				{ID: "pr-2", Repository: "acme/app", Number: 2, State: "draft"},
			},
		}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "acme/app",
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, factoryCtx.listCalls)
		assert.Equal(t, "acme/app", factoryCtx.listParams.Repository)
		assert.Empty(t, factoryCtx.listParams.States)
		assert.Equal(t, core.DefaultOutputChannel.Name, stateCtx.Channel)
		assert.Equal(t, listPullRequestsEventType, stateCtx.Type)
		require.Len(t, stateCtx.Payloads, 1)

		pullRequests, ok := payloadData(t, stateCtx, 0)["pullRequests"].([]*core.PullRequest)
		require.True(t, ok)
		assert.Len(t, pullRequests, 2)
	})

	t.Run("emits an empty list without error", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{listResult: nil}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"repository": "acme/app"},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Empty(t, payloadData(t, stateCtx, 0)["pullRequests"])
	})

	t.Run("passes explicit states through", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "acme/app",
				"states":     []any{"open"},
			},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"open"}, factoryCtx.listParams.States)
	})

	t.Run("propagates real errors", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{listErr: errors.New("boom")}
		stateCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"repository": "acme/app"},
			ExecutionState: stateCtx,
			Factory:        factoryCtx,
		})
		require.Error(t, err)
		assert.EqualError(t, err, "boom")
	})
}

func payloadData(t *testing.T, stateCtx *contexts.ExecutionStateContext, index int) map[string]any {
	t.Helper()

	payload, ok := stateCtx.Payloads[index].(map[string]any)
	require.True(t, ok)
	data, ok := payload["data"].(map[string]any)
	require.True(t, ok)
	return data
}
