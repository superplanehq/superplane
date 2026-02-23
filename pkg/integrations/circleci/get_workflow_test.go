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

func Test__GetWorkflow__Setup(t *testing.T) {
	c := &GetWorkflow{}

	t.Run("missing workflow ID -> error", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"workflowId": "",
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "workflow ID is required")
	})

	t.Run("valid workflow ID -> success", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"workflowId": "wf-123",
			},
		})

		require.NoError(t, err)
	})
}

func Test__GetWorkflow__Execute(t *testing.T) {
	c := &GetWorkflow{}

	t.Run("fetches workflow and jobs -> emits to default channel", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"wf-123","name":"build","status":"success","created_at":"2024-01-01T00:00:00Z","stopped_at":"2024-01-01T00:05:00Z"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"items":[{"id":"job-1","name":"compile","type":"build","status":"success","job_number":10}]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token-123",
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"workflowId": "wf-123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Finished)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "circleci.workflow", executionState.Type)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/workflow/wf-123")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/workflow/wf-123/job")
	})
}
