package circleci

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__RunPipeline__buildParameters(t *testing.T) {
	t.Run("builds parameters map", func(t *testing.T) {
		tp := &RunPipeline{}
		params := []Parameter{
			{Name: "env", Value: "production"},
			{Name: "version", Value: "1.0.0"},
		}

		result := tp.buildParameters(params)

		assert.Equal(t, "production", result["env"])
		assert.Equal(t, "1.0.0", result["version"])
		assert.Len(t, result, 2)
	})
}

func pollContext(t *testing.T, workflows string) (core.ActionHookContext, *contexts.ExecutionStateContext, *contexts.RequestContext) {
	t.Helper()

	// The pipeline lookup and the workflow listing are both served the same
	// body, so the fake does not depend on the order the two are called in.
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(workflows)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(workflows)),
			},
		},
	}

	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requests := &contexts.RequestContext{}

	return core.ActionHookContext{
		Name: "poll",
		HTTP: httpContext,
		Metadata: &contexts.MetadataContext{
			Metadata: map[string]any{
				"pipeline": map[string]any{"id": "pipe-123", "number": 7},
			},
		},
		ExecutionState: executionState,
		Requests:       requests,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		},
	}, executionState, requests
}

func Test__RunPipeline__poll(t *testing.T) {
	component := &RunPipeline{}

	t.Run("all workflows successful -> success channel", func(t *testing.T) {
		ctx, executionState, _ := pollContext(t, `{"items":[
			{"id":"wf-1","name":"build","status":"success"},
			{"id":"wf-2","name":"deploy","status":"success"}
		]}`)

		require.NoError(t, component.poll(ctx))

		assert.True(t, executionState.Finished)
		assert.Equal(t, SuccessOutputChannel, executionState.Channel)
	})

	t.Run("a later workflow is still running -> keeps polling", func(t *testing.T) {
		ctx, executionState, requests := pollContext(t, `{"items":[
			{"id":"wf-1","name":"build","status":"success"},
			{"id":"wf-2","name":"deploy","status":"running"}
		]}`)

		require.NoError(t, component.poll(ctx))

		assert.False(t, executionState.Finished, "must not emit while a workflow is still running")
		assert.Equal(t, "poll", requests.Action)
	})

	t.Run("a later workflow failed -> failed channel", func(t *testing.T) {
		ctx, executionState, _ := pollContext(t, `{"items":[
			{"id":"wf-1","name":"build","status":"success"},
			{"id":"wf-2","name":"deploy","status":"failed"}
		]}`)

		require.NoError(t, component.poll(ctx))

		assert.True(t, executionState.Finished)
		assert.Equal(t, FailedOutputChannel, executionState.Channel)
	})

	t.Run("no workflows and the pipeline errored -> failed channel", func(t *testing.T) {
		ctx, executionState, _ := pollContext(t, `{"items":[],"state":"errored"}`)

		require.NoError(t, component.poll(ctx))

		assert.True(t, executionState.Finished)
		assert.Equal(t, FailedOutputChannel, executionState.Channel)
	})

	t.Run("no workflows started yet -> keeps polling", func(t *testing.T) {
		ctx, executionState, requests := pollContext(t, `{"items":[],"state":"created"}`)

		require.NoError(t, component.poll(ctx))

		assert.False(t, executionState.Finished)
		assert.Equal(t, "poll", requests.Action)
	})
}
