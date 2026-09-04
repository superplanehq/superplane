package pulls

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func mergeablePullRequestResponse(mergeable, mergeableState string) *http.Response {
	body := `{
		"number": 42,
		"head": {"sha": "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44"},
		"base": {"ref": "main"}` +
		mergeableField(mergeable) +
		`, "mergeable_state": "` + mergeableState + `"
	}`
	return mocks.GitHubResponse(http.StatusOK, body)
}

func mergeableField(mergeable string) string {
	if mergeable == "null" {
		return `, "mergeable": null`
	}
	return `, "mergeable": ` + mergeable
}

func Test__WaitForPullRequestMergeable__Setup(t *testing.T) {
	component := &WaitForPullRequestMergeable{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			mocks.GitHubResponse(http.StatusOK, `{
				"id": 123456,
				"name": "hello",
				"html_url": "https://github.com/testhq/hello"
			}`),
		},
	}
	metadata := &contexts.MetadataContext{}

	err := component.Setup(core.SetupContext{
		Integration:   mocks.IntegrationContextForNewSetupFlow(),
		HTTP:          httpCtx,
		Metadata:      metadata,
		Configuration: map[string]any{"repository": "hello", "number": "42"},
	})

	require.NoError(t, err)
}

func Test__WaitForPullRequestMergeable__Setup_RejectsMissingNumber(t *testing.T) {
	component := &WaitForPullRequestMergeable{}

	err := component.Setup(core.SetupContext{
		Integration:   mocks.IntegrationContextForNewSetupFlow(),
		HTTP:          &contexts.HTTPContext{},
		Metadata:      &contexts.MetadataContext{},
		Configuration: map[string]any{"repository": "hello"},
	})

	assert.Error(t, err)
}

func Test__WaitForPullRequestMergeable__Execute(t *testing.T) {
	component := &WaitForPullRequestMergeable{}

	t.Run("emits clean when GitHub reports mergeable true", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{mergeablePullRequestResponse("true", "clean")}}
		executionState := &contexts.ExecutionStateContext{}
		requests := &contexts.RequestContext{}
		metadata := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "hello",
				"number":     "42",
			},
			HTTP:           httpCtx,
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: executionState,
			Requests:       requests,
			Metadata:       metadata,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, waitMergeableCleanChannel, executionState.Channel)
		assert.Empty(t, requests.Action)

		payload := executionState.Payloads[0].(map[string]any)
		data := payload["data"].(WaitForPullRequestMergeableOutput)
		require.NotNil(t, data.Mergeable)
		assert.True(t, *data.Mergeable)
		assert.Equal(t, "clean", data.MergeableState)
		assert.Equal(t, "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44", data.SHA)
		assert.Equal(t, "main", data.BaseRef)
	})

	t.Run("emits conflicted when GitHub reports mergeable false", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{mergeablePullRequestResponse("false", "dirty")}}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "hello",
				"number":     "42",
			},
			HTTP:           httpCtx,
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
			Metadata:       &contexts.MetadataContext{},
		})

		require.NoError(t, err)
		assert.Equal(t, waitMergeableConflictedChannel, executionState.Channel)

		payload := executionState.Payloads[0].(map[string]any)
		data := payload["data"].(WaitForPullRequestMergeableOutput)
		require.NotNil(t, data.Mergeable)
		assert.False(t, *data.Mergeable)
	})

	t.Run("treats mergeable_state dirty as a conflict even when the flag lags", func(t *testing.T) {
		// GitHub occasionally reports mergeable: true for one more poll while
		// mergeable_state still reads dirty from a stale cache entry.
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{mergeablePullRequestResponse("true", "dirty")}}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "hello",
				"number":     "42",
			},
			HTTP:           httpCtx,
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
			Metadata:       &contexts.MetadataContext{},
		})

		require.NoError(t, err)
		assert.Equal(t, waitMergeableConflictedChannel, executionState.Channel)
	})

	t.Run("does not treat blocked, behind, or unstable as a conflict", func(t *testing.T) {
		for _, state := range []string{"blocked", "behind", "unstable"} {
			httpCtx := &contexts.HTTPContext{Responses: []*http.Response{mergeablePullRequestResponse("true", state)}}
			executionState := &contexts.ExecutionStateContext{}

			err := component.Execute(core.ExecutionContext{
				Configuration: map[string]any{
					"repository": "hello",
					"number":     "42",
				},
				HTTP:           httpCtx,
				Integration:    mocks.IntegrationContextForNewSetupFlow(),
				ExecutionState: executionState,
				Requests:       &contexts.RequestContext{},
				Metadata:       &contexts.MetadataContext{},
			})

			require.NoError(t, err)
			assert.Equal(t, waitMergeableCleanChannel, executionState.Channel, "state %s should not be a conflict", state)
		}
	})

	t.Run("schedules another poll while mergeable is null", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{mergeablePullRequestResponse("null", "unknown")}}
		executionState := &contexts.ExecutionStateContext{}
		requests := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository":     "hello",
				"number":         "42",
				"timeoutSeconds": 300,
			},
			HTTP:           httpCtx,
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: executionState,
			Requests:       requests,
			Metadata:       &contexts.MetadataContext{},
		})

		require.NoError(t, err)
		assert.False(t, executionState.Finished)
		assert.Equal(t, waitMergeableEvaluateHook, requests.Action)
		assert.Equal(t, waitMergeablePollInterval, requests.Duration)
	})

	t.Run("emits timedOut when mergeable stays null past the deadline", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{mergeablePullRequestResponse("null", "unknown")}}
		executionState := &contexts.ExecutionStateContext{}
		metadata := &contexts.MetadataContext{
			Metadata: WaitForPullRequestMergeableMetadata{TimeoutAtUnix: time.Now().Add(-time.Second).Unix()},
		}

		err := evaluateWaitForPullRequestMergeable(waitMergeableRuntime{
			Configuration:  WaitForPullRequestMergeableConfiguration{Repository: "hello", Number: "42"},
			HTTP:           httpCtx,
			Metadata:       metadata,
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
		})

		require.NoError(t, err)
		assert.Equal(t, waitMergeableTimedOutChannel, executionState.Channel)

		payload := executionState.Payloads[0].(map[string]any)
		data := payload["data"].(WaitForPullRequestMergeableOutput)
		assert.Nil(t, data.Mergeable)
	})
}

func Test__WaitForPullRequestMergeable__HandleHook(t *testing.T) {
	component := &WaitForPullRequestMergeable{}

	t.Run("does nothing when the execution already finished", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{Finished: true}

		err := component.HandleHook(core.ActionHookContext{
			Name:           waitMergeableEvaluateHook,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.Empty(t, executionState.Channel)
	})

	t.Run("rejects an unknown hook name", func(t *testing.T) {
		err := component.HandleHook(core.ActionHookContext{
			Name:           "bogus",
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		assert.Error(t, err)
	})
}
