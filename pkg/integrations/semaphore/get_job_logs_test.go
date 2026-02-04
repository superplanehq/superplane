package semaphore

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetJobLogs__Execute__Validation(t *testing.T) {
	component := GetJobLogs{}

	t.Run("returns error when jobId is empty", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://test.semaphoreci.com",
				"apiToken":        "test-token",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"jobId": "",
			},
		})

		require.ErrorContains(t, err, "job ID is required")
	})

	t.Run("returns error when limit exceeds maximum", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://test.semaphoreci.com",
				"apiToken":        "test-token",
			},
		}

		limit := 2000
		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"jobId": "test-job-id",
				"limit": limit,
			},
		})

		require.ErrorContains(t, err, "limit cannot exceed 1000 lines")
	})
}

func Test__GetJobLogs__buildLogsOutput(t *testing.T) {
	component := GetJobLogs{}

	t.Run("extracts output lines from events", func(t *testing.T) {
		logs := &JobLogsResponse{
			Events: []JobLogEvent{
				{Event: "job_started", Timestamp: 1000},
				{Event: "cmd_output", Output: "line 1\n"},
				{Event: "cmd_output", Output: "line 2\n"},
				{Event: "job_finished", Result: "passed"},
			},
		}

		output := component.buildLogsOutput(logs, nil)

		require.Equal(t, "line 1\nline 2\n", output["output"])
		require.Equal(t, 2, output["lineCount"])
		require.Equal(t, "passed", output["result"])
	})

	t.Run("applies line limit", func(t *testing.T) {
		logs := &JobLogsResponse{
			Events: []JobLogEvent{
				{Event: "cmd_output", Output: "line 1\n"},
				{Event: "cmd_output", Output: "line 2\n"},
				{Event: "cmd_output", Output: "line 3\n"},
				{Event: "cmd_output", Output: "line 4\n"},
			},
		}

		limit := 2
		output := component.buildLogsOutput(logs, &limit)

		// Should return last 2 lines
		require.Equal(t, "line 3\nline 4\n", output["output"])
		require.Equal(t, 2, output["lineCount"])
	})
}
