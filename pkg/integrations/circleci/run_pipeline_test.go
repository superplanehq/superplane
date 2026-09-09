package circleci

import (
	"fmt"
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

func workflowsResponse(statuses ...string) *http.Response {
	items := make([]string, 0, len(statuses))
	for i, status := range statuses {
		items = append(items, fmt.Sprintf(`{
			"id": "workflow-%d",
			"name": "workflow-%d",
			"status": "%s"
		}`, i, i, status))
	}

	body := `{"items": [` + strings.Join(items, ",") + `]}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func pipelineResponse(state string) *http.Response {
	body := `{"id": "pipeline-1", "number": 1, "state": "` + state + `"}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func Test__RunPipeline__poll(t *testing.T) {
	metadata := &contexts.MetadataContext{
		Metadata: map[string]any{
			"pipeline": map[string]any{
				"id": "pipeline-1",
			},
		},
	}

	newCtx := func(responses ...*http.Response) (*contexts.ExecutionStateContext, *contexts.RequestContext, core.ActionHookContext) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		requests := &contexts.RequestContext{}

		ctx := core.ActionHookContext{
			Name:           "poll",
			HTTP:           &contexts.HTTPContext{Responses: responses},
			Metadata:       metadata,
			ExecutionState: executionState,
			Requests:       requests,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
		}

		return executionState, requests, ctx
	}

	component := &RunPipeline{}

	t.Run("a later workflow failed -> failed channel", func(t *testing.T) {
		executionState, requests, ctx := newCtx(workflowsResponse("success", "failed"))

		err := component.poll(ctx)

		require.NoError(t, err)
		assert.True(t, executionState.Finished)
		assert.Equal(t, FailedOutputChannel, executionState.Channel)
		assert.Empty(t, requests.Action)
	})

	t.Run("a later workflow still running -> reschedules poll", func(t *testing.T) {
		executionState, requests, ctx := newCtx(workflowsResponse("success", "running"))

		err := component.poll(ctx)

		require.NoError(t, err)
		assert.False(t, executionState.Finished)
		assert.Equal(t, "poll", requests.Action)
		assert.Equal(t, PollInterval, requests.Duration)
	})

	t.Run("all workflows succeeded -> success channel", func(t *testing.T) {
		executionState, requests, ctx := newCtx(workflowsResponse("success", "success"))

		err := component.poll(ctx)

		require.NoError(t, err)
		assert.True(t, executionState.Finished)
		assert.Equal(t, SuccessOutputChannel, executionState.Channel)
		assert.Empty(t, requests.Action)
	})

	t.Run("empty workflow list, pipeline not errored -> reschedules poll", func(t *testing.T) {
		executionState, requests, ctx := newCtx(workflowsResponse(), pipelineResponse("created"))

		err := component.poll(ctx)

		require.NoError(t, err)
		assert.False(t, executionState.Finished)
		assert.Equal(t, "poll", requests.Action)
		assert.Equal(t, PollInterval, requests.Duration)
	})

	t.Run("empty workflow list, pipeline errored -> failed channel", func(t *testing.T) {
		executionState, requests, ctx := newCtx(workflowsResponse(), pipelineResponse("errored"))

		err := component.poll(ctx)

		require.NoError(t, err)
		assert.True(t, executionState.Finished)
		assert.Equal(t, FailedOutputChannel, executionState.Channel)
		assert.Empty(t, requests.Action)
	})

	t.Run("workflow fetch error -> reschedules poll without failing", func(t *testing.T) {
		executionState, requests, ctx := newCtx(&http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		})

		err := component.poll(ctx)

		require.NoError(t, err)
		assert.False(t, executionState.Finished)
		assert.Equal(t, "poll", requests.Action)
		assert.Equal(t, PollInterval, requests.Duration)
	})

	t.Run("a later workflow canceled -> failed channel", func(t *testing.T) {
		executionState, requests, ctx := newCtx(workflowsResponse("success", "canceled"))

		err := component.poll(ctx)

		require.NoError(t, err)
		assert.True(t, executionState.Finished)
		assert.Equal(t, FailedOutputChannel, executionState.Channel)
		assert.Empty(t, requests.Action)
	})
}
