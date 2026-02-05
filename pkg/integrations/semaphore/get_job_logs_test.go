package semaphore

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetJobLogs__Name(t *testing.T) {
	g := &GetJobLogs{}
	assert.Equal(t, "semaphore.getJobLogs", g.Name())
}

func Test__GetJobLogs__Label(t *testing.T) {
	g := &GetJobLogs{}
	assert.Equal(t, "Get Job Logs", g.Label())
}

func Test__GetJobLogs__Description(t *testing.T) {
	g := &GetJobLogs{}
	assert.Equal(t, "Fetch logs for a Semaphore job", g.Description())
}

func Test__GetJobLogs__OutputChannels(t *testing.T) {
	g := &GetJobLogs{}
	channels := g.OutputChannels(nil)
	require.Len(t, channels, 1)
	assert.Equal(t, "success", channels[0].Name)
	assert.Equal(t, "Success", channels[0].Label)
}

func Test__GetJobLogs__Configuration(t *testing.T) {
	g := &GetJobLogs{}
	config := g.Configuration()
	require.Len(t, config, 2)

	// Job ID field
	assert.Equal(t, "jobId", config[0].Name)
	assert.Equal(t, "Job ID", config[0].Label)
	assert.True(t, config[0].Required)

	// Limit field
	assert.Equal(t, "limit", config[1].Name)
	assert.Equal(t, "Line Limit", config[1].Label)
	assert.False(t, config[1].Required)
	assert.Equal(t, GetJobLogsDefaultLimit, config[1].Default)
}

func Test__GetJobLogs__Setup(t *testing.T) {
	g := &GetJobLogs{}

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"jobId": "test-job-id",
				"limit": 500,
			},
		}

		err := g.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("valid configuration with default limit", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"jobId": "test-job-id",
			},
		}

		err := g.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("negative limit", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"jobId": "test-job-id",
				"limit": -1,
			},
		}

		err := g.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "limit must be a positive number")
	})

	t.Run("limit exceeds maximum", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"jobId": "test-job-id",
				"limit": GetJobLogsMaxLimit + 1,
			},
		}

		err := g.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "limit cannot exceed")
	})
}

func Test__GetJobLogs__Execute(t *testing.T) {
	g := &GetJobLogs{}

	t.Run("successfully fetches job logs", func(t *testing.T) {
		jobResponse := JobResponse{
			Metadata: JobMetadata{
				ID:         "job-123",
				Name:       "Test Job",
				CreateTime: "2024-01-01T00:00:00Z",
				StartTime:  "2024-01-01T00:00:01Z",
				FinishTime: "2024-01-01T00:01:00Z",
			},
			Status: JobStatus{
				State:  "finished",
				Result: "passed",
			},
		}
		jobResponseBytes, _ := json.Marshal(jobResponse)

		logResponse := JobLogResponse{
			Events: []JobLogEvent{
				{Event: "cmd_output", Output: "Line 1", Timestamp: 1000},
				{Event: "cmd_output", Output: "Line 2", Timestamp: 1001},
				{Event: "cmd_output", Output: "Line 3", Timestamp: 1002},
			},
		}
		logResponseBytes, _ := json.Marshal(logResponse)

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(string(jobResponseBytes))),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(string(logResponseBytes))),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestsCtx := &contexts.RequestsContext{}
		loggerCtx := &contexts.LoggerContext{}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"jobId": "job-123",
				"limit": 100,
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Metadata:    metadataCtx,
			Requests:    requestsCtx,
			Logger:      loggerCtx,
		}

		err := g.Execute(ctx)
		require.NoError(t, err)

		// Verify HTTP requests
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "https://example.semaphoreci.com/api/v1alpha/jobs/job-123", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://example.semaphoreci.com/api/v1alpha/jobs/job-123/log", httpContext.Requests[1].URL.String())

		// Verify emission
		require.Len(t, requestsCtx.Emissions, 1)
		assert.Equal(t, "success", requestsCtx.Emissions[0].Channel)
		assert.Equal(t, GetJobLogsPayloadType, requestsCtx.Emissions[0].PayloadType)
	})

	t.Run("missing job ID returns error", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{},
		}

		err := g.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "job ID is required")
	})

	t.Run("job not found returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("job not found")),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		loggerCtx := &contexts.LoggerContext{}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"jobId": "nonexistent-job",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Logger:      loggerCtx,
		}

		err := g.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error fetching job")
	})

	t.Run("applies line limit correctly", func(t *testing.T) {
		jobResponse := JobResponse{
			Metadata: JobMetadata{ID: "job-123", Name: "Test Job"},
			Status:   JobStatus{State: "finished", Result: "passed"},
		}
		jobResponseBytes, _ := json.Marshal(jobResponse)

		// Create more log lines than the limit
		var events []JobLogEvent
		for i := 0; i < 20; i++ {
			events = append(events, JobLogEvent{
				Event:     "cmd_output",
				Output:    "Log line " + string(rune('A'+i)),
				Timestamp: int64(1000 + i),
			})
		}
		logResponse := JobLogResponse{Events: events}
		logResponseBytes, _ := json.Marshal(logResponse)

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(jobResponseBytes)))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(logResponseBytes)))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestsCtx := &contexts.RequestsContext{}
		loggerCtx := &contexts.LoggerContext{}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"jobId": "job-123",
				"limit": 5, // Only get last 5 lines
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Metadata:    metadataCtx,
			Requests:    requestsCtx,
			Logger:      loggerCtx,
		}

		err := g.Execute(ctx)
		require.NoError(t, err)

		// Verify we got only the last 5 lines
		require.Len(t, requestsCtx.Emissions, 1)
		payload := requestsCtx.Emissions[0].Payload.([]any)[0].(GetJobLogsOutput)
		assert.Len(t, payload.LogLines, 5)
	})
}

func Test__GetJobLogs__Cancel(t *testing.T) {
	g := &GetJobLogs{}
	err := g.Cancel(core.ExecutionContext{})
	require.NoError(t, err)
}

func Test__GetJobLogs__Actions(t *testing.T) {
	g := &GetJobLogs{}
	actions := g.Actions()
	assert.Empty(t, actions)
}

func Test__GetJobLogs__HandleAction(t *testing.T) {
	g := &GetJobLogs{}
	err := g.HandleAction(core.ActionContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no actions available")
}

func Test__GetJobLogs__Cleanup(t *testing.T) {
	g := &GetJobLogs{}
	err := g.Cleanup(core.SetupContext{})
	require.NoError(t, err)
}
