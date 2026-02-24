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

func Test__GetWorkflow__SetupAndExecute(t *testing.T) {
	component := &GetWorkflow{}

	t.Run("missing workflow ID returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "workflowId is required")
	})

	t.Run("success returns workflow, jobs, pipeline info and duration", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": "wf-1",
						"name": "build-test-deploy",
						"status": "success",
						"created_at": "2026-02-24T16:00:00Z",
						"stopped_at": "2026-02-24T16:01:30Z",
						"pipeline_id": "pipe-1",
						"pipeline_number": 101,
						"project_slug": "gh/org/repo"
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [
							{"id":"job-1","job_number":10,"name":"build","status":"success","type":"build","duration":30},
							{"id":"job-2","job_number":11,"name":"test","status":"success","type":"build","duration":60}
						]
					}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "token-123"},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"workflowId": "wf-1",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    integrationCtx,
		})
		require.NoError(t, err)

		assert.Equal(t, "circleci.workflow", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		data := payload["data"].(GetWorkflowResult)

		assert.Equal(t, "wf-1", data.Workflow.ID)
		assert.Equal(t, "pipe-1", data.Pipeline.ID)
		assert.Equal(t, 101, data.Pipeline.Number)
		assert.Equal(t, int64(90), data.DurationSeconds)
		require.Len(t, data.Jobs, 2)
		assert.Equal(t, "job-1", data.Jobs[0].ID)
		assert.Equal(t, "job-2", data.Jobs[1].ID)

		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "https://circleci.com/api/v2/workflow/wf-1", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://circleci.com/api/v2/workflow/wf-1/job", httpContext.Requests[1].URL.String())
	})
}

func Test__GetLastWorkflow__Execute(t *testing.T) {
	component := &GetLastWorkflow{}

	t.Run("missing project slug returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "projectSlug is required")
	})

	t.Run("status filter chooses first matching workflow", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [
							{"id":"pipe-1","number":1,"state":"created","created_at":"2026-02-24T15:00:00Z","updated_at":"2026-02-24T15:01:00Z"},
							{"id":"pipe-2","number":2,"state":"created","created_at":"2026-02-24T16:00:00Z","updated_at":"2026-02-24T16:01:00Z"}
						]
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [{"id":"wf-1","name":"build","status":"failed"}]
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [{"id":"wf-2","name":"build","status":"success"}]
					}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "token-123"},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"projectSlug": "gh/org/repo",
				"branch":      "main",
				"status":      "success",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    integrationCtx,
		})
		require.NoError(t, err)

		assert.Equal(t, "circleci.lastWorkflow", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		data := payload["data"].(GetLastWorkflowResult)
		assert.Equal(t, "gh/org/repo", data.ProjectSlug)
		assert.Equal(t, "pipe-1", data.Pipeline.ID)
		assert.Equal(t, "wf-2", data.Workflow.ID)

		require.Len(t, httpContext.Requests, 3)
		assert.Equal(t, "https://circleci.com/api/v2/project/gh/org/repo/pipeline?branch=main", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://circleci.com/api/v2/pipeline/pipe-2/workflow", httpContext.Requests[1].URL.String())
		assert.Equal(t, "https://circleci.com/api/v2/pipeline/pipe-1/workflow", httpContext.Requests[2].URL.String())
	})
}

func Test__GetRecentWorkflowRuns__Execute(t *testing.T) {
	component := &GetRecentWorkflowRuns{}

	t.Run("missing project slug returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "projectSlug is required")
	})

	t.Run("success returns insights payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items":[{"name":"build","metrics":{"success_rate":0.9}}],
						"next_page_token":""
					}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "token-123"},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"projectSlug":     "gh/org/repo",
				"branch":          "main",
				"reportingWindow": "last-90-days",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    integrationCtx,
		})
		require.NoError(t, err)

		assert.Equal(t, "circleci.recentWorkflowRuns", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		data := payload["data"].(GetRecentWorkflowRunsResult)
		assert.Equal(t, "gh/org/repo", data.ProjectSlug)
		assert.Equal(t, "main", data.Branch)
		assert.Equal(t, "last-90-days", data.ReportingWindow)
		require.NotNil(t, data.Insights["items"])

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://circleci.com/api/v2/insights/gh/org/repo/workflows?branch=main&reporting-window=last-90-days", httpContext.Requests[0].URL.String())
	})
}

func Test__GetTestMetrics__Execute(t *testing.T) {
	component := &GetTestMetrics{}

	t.Run("missing project slug or workflow name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectSlug": "gh/org/repo",
			},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "workflowName is required")
	})

	t.Run("workflow name is URL-escaped and metrics are emitted", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"items":[{"total_runs":100,"total_failing":7}]}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "token-123"},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"projectSlug":     "gh/org/repo",
				"workflowName":    "Build and Test",
				"branch":          "main",
				"reportingWindow": "last-7-days",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    integrationCtx,
		})
		require.NoError(t, err)

		assert.Equal(t, "circleci.testMetrics", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		data := payload["data"].(GetTestMetricsResult)
		assert.Equal(t, "gh/org/repo", data.ProjectSlug)
		assert.Equal(t, "Build and Test", data.WorkflowName)
		require.NotNil(t, data.Metrics["items"])

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://circleci.com/api/v2/insights/gh/org/repo/workflows/Build%20and%20Test/test-metrics?branch=main&reporting-window=last-7-days", httpContext.Requests[0].URL.String())
	})
}

func Test__GetFlakyTests__SetupAndExecute(t *testing.T) {
	component := &GetFlakyTests{}

	t.Run("missing project slug returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "projectSlug is required")
	})

	t.Run("success returns flaky tests data", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"items":[{"name":"TestRetries","flaky_rate":0.2}]}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "token-123"},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"projectSlug":     "gh/org/repo",
				"branch":          "main",
				"reportingWindow": "last-30-days",
			},
			HTTP:           httpContext,
			ExecutionState: executionState,
			Integration:    integrationCtx,
		})
		require.NoError(t, err)

		assert.Equal(t, "circleci.flakyTests", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		data := payload["data"].(GetFlakyTestsResult)
		assert.Equal(t, "gh/org/repo", data.ProjectSlug)
		assert.Equal(t, "main", data.Branch)
		require.NotNil(t, data.FlakyTests["items"])

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://circleci.com/api/v2/insights/gh/org/repo/flaky-tests?branch=main&reporting-window=last-30-days", httpContext.Requests[0].URL.String())
	})
}
