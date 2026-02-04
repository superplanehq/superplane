package semaphore

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

func Test__GetJobLogs__Name(t *testing.T) {
	g := &GetJobLogs{}
	assert.Equal(t, "semaphore.getJobLogs", g.Name())
}

func Test__GetJobLogs__Label(t *testing.T) {
	g := &GetJobLogs{}
	assert.Equal(t, "Get Job Logs", g.Label())
}

func Test__GetJobLogs__OutputChannels(t *testing.T) {
	g := &GetJobLogs{}
	channels := g.OutputChannels(nil)
	require.Len(t, channels, 1)
	assert.Equal(t, "logs", channels[0].Name)
	assert.Equal(t, "Logs", channels[0].Label)
}

func Test__GetJobLogs__Configuration(t *testing.T) {
	g := &GetJobLogs{}
	config := g.Configuration()
	require.Len(t, config, 2)

	// Check Job ID field
	assert.Equal(t, "jobId", config[0].Name)
	assert.Equal(t, "Job ID", config[0].Label)
	assert.True(t, config[0].Required)

	// Check Limit field
	assert.Equal(t, "limit", config[1].Name)
	assert.Equal(t, "Limit", config[1].Label)
	assert.Equal(t, DefaultLogLimit, config[1].Default)
}

func Test__GetJobLogs__Execute(t *testing.T) {
	g := &GetJobLogs{}

	t.Run("successfully fetches job logs", func(t *testing.T) {
		jobResponse := `{"job": {"id": "job-123", "name": "test-job", "status": "finished", "result": "passed"}}`
		logsResponse := `{"logs": [{"number": 1, "content": "Line 1"}, {"number": 2, "content": "Line 2"}]}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(jobResponse)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(logsResponse)),
				},
			},
		}

		executionCtx := &contexts.ExecutionContext{
			Configuration: map[string]any{
				"jobId": "job-123",
				"limit": 100,
			},
			IntegrationConfiguration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		err := g.Execute(core.ExecutionContext{
			Configuration: executionCtx.Configuration,
			HTTP:          httpContext,
			Integration:   executionCtx,
			Logger:        executionCtx.Logger,
			Metadata:      executionCtx,
			ExecutionState: executionCtx,
		})

		require.NoError(t, err)
		require.Len(t, executionCtx.Emitted, 1)
		assert.Equal(t, "logs", executionCtx.Emitted[0].Channel)
	})

	t.Run("error when jobId is empty", func(t *testing.T) {
		executionCtx := &contexts.ExecutionContext{
			Configuration: map[string]any{
				"jobId": "",
			},
		}

		err := g.Execute(core.ExecutionContext{
			Configuration: executionCtx.Configuration,
			Logger:        executionCtx.Logger,
			ExecutionState: executionCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "jobId is required")
	})

	t.Run("uses default limit when not specified", func(t *testing.T) {
		jobResponse := `{"job": {"id": "job-123", "name": "test-job", "status": "finished", "result": "passed"}}`
		logsResponse := `{"logs": []}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(jobResponse)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(logsResponse)),
				},
			},
		}

		executionCtx := &contexts.ExecutionContext{
			Configuration: map[string]any{
				"jobId": "job-123",
				// No limit specified
			},
			IntegrationConfiguration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		err := g.Execute(core.ExecutionContext{
			Configuration: executionCtx.Configuration,
			HTTP:          httpContext,
			Integration:   executionCtx,
			Logger:        executionCtx.Logger,
			Metadata:      executionCtx,
			ExecutionState: executionCtx,
		})

		require.NoError(t, err)
		// Should use default limit
		require.Len(t, executionCtx.Emitted, 1)
	})
}

func Test__Client__GetJob(t *testing.T) {
	t.Run("successfully fetches job details", func(t *testing.T) {
		jobResponse := `{"job": {"id": "job-123", "name": "test-job", "status": "finished", "result": "passed"}}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(jobResponse)),
				},
			},
		}

		client, err := NewClient(httpContext, &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		})
		require.NoError(t, err)

		job, err := client.GetJob("job-123")
		require.NoError(t, err)
		assert.Equal(t, "job-123", job.ID)
		assert.Equal(t, "test-job", job.Name)
		assert.Equal(t, "finished", job.Status)
		assert.Equal(t, "passed", job.Result)
	})

	t.Run("error when job not found", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message": "Job not found"}`)),
				},
			},
		}

		client, err := NewClient(httpContext, &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		})
		require.NoError(t, err)

		_, err = client.GetJob("job-123")
		require.Error(t, err)
	})
}

func Test__Client__GetJobLogs(t *testing.T) {
	t.Run("successfully fetches job logs", func(t *testing.T) {
		logsResponse := `{"logs": [{"number": 1, "content": "Line 1"}, {"number": 2, "content": "Line 2"}]}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(logsResponse)),
				},
			},
		}

		client, err := NewClient(httpContext, &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		})
		require.NoError(t, err)

		logs, err := client.GetJobLogs("job-123", 100)
		require.NoError(t, err)
		assert.Contains(t, logs, "Line 1")
		assert.Contains(t, logs, "Line 2")
	})

	t.Run("returns empty string when no logs", func(t *testing.T) {
		logsResponse := `{"logs": []}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(logsResponse)),
				},
			},
		}

		client, err := NewClient(httpContext, &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		})
		require.NoError(t, err)

		logs, err := client.GetJobLogs("job-123", 100)
		require.NoError(t, err)
		assert.Equal(t, "", logs)
	})
}
