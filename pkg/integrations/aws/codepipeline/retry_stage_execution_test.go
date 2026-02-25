package codepipeline

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

func Test__RetryStageExecution__Setup(t *testing.T) {
	component := &RetryStageExecution{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"pipeline":            "my-pipeline",
				"stage":               "Deploy",
				"pipelineExecutionId": "old-exec-123",
				"retryMode":           "FAILED_ACTIONS",
			},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing pipeline -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":              "us-east-1",
				"stage":               "Deploy",
				"pipelineExecutionId": "old-exec-123",
				"retryMode":           "FAILED_ACTIONS",
			},
		})
		require.ErrorContains(t, err, "pipeline is required")
	})

	t.Run("missing stage -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":              "us-east-1",
				"pipeline":            "my-pipeline",
				"pipelineExecutionId": "old-exec-123",
				"retryMode":           "FAILED_ACTIONS",
			},
		})
		require.ErrorContains(t, err, "stage is required")
	})

	t.Run("missing pipeline execution id -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":    "us-east-1",
				"pipeline":  "my-pipeline",
				"stage":     "Deploy",
				"retryMode": "FAILED_ACTIONS",
			},
		})
		require.ErrorContains(t, err, "pipeline execution ID is required")
	})

	t.Run("invalid retry mode -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":              "us-east-1",
				"pipeline":            "my-pipeline",
				"stage":               "Deploy",
				"pipelineExecutionId": "old-exec-123",
				"retryMode":           "INVALID",
			},
		})
		require.ErrorContains(t, err, "retry mode must be one of")
	})

	t.Run("valid configuration -> ok", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region":              "us-east-1",
				"pipeline":            "my-pipeline",
				"stage":               "Deploy",
				"pipelineExecutionId": "old-exec-123",
				"retryMode":           "FAILED_ACTIONS",
			},
		})
		require.NoError(t, err)
	})
}

func Test__RetryStageExecution__Execute(t *testing.T) {
	component := &RetryStageExecution{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":              "us-east-1",
				"pipeline":            "my-pipeline",
				"stage":               "Deploy",
				"pipelineExecutionId": "old-exec-123",
				"retryMode":           "FAILED_ACTIONS",
			},
			Integration:    &contexts.IntegrationContext{Secrets: map[string]core.IntegrationSecret{}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})
		require.ErrorContains(t, err, "AWS session credentials are missing")
	})

	t.Run("api error -> wrapped error includes operation context", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"__type":"ValidationException","message":"invalid stage name"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":              "us-east-1",
				"pipeline":            "my-pipeline",
				"stage":               "Deploy",
				"pipelineExecutionId": "old-exec-123",
				"retryMode":           "FAILED_ACTIONS",
			},
			HTTP:           httpContext,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration: &contexts.IntegrationContext{
				Secrets: validSecrets(),
			},
		})

		require.ErrorContains(t, err, "failed to retry stage execution")
		require.ErrorContains(t, err, "invalid stage name")
	})

	t.Run("http success -> emits retry payload on default channel", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"pipelineExecutionId":"new-exec-456"}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"region":              "us-east-1",
				"pipeline":            "my-pipeline",
				"stage":               "Deploy",
				"pipelineExecutionId": "old-exec-123",
				"retryMode":           "FAILED_ACTIONS",
			},
			HTTP:           httpContext,
			ExecutionState: execState,
			Integration: &contexts.IntegrationContext{
				Secrets: validSecrets(),
			},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, "aws.codepipeline.stage.retry", execState.Type)
		require.Len(t, execState.Payloads, 1)

		wrapped, ok := execState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		pipeline, ok := data["pipeline"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "my-pipeline", pipeline["name"])
		assert.Equal(t, "Deploy", pipeline["stage"])
		assert.Equal(t, "FAILED_ACTIONS", pipeline["retryMode"])
		assert.Equal(t, "old-exec-123", pipeline["sourceExecutionId"])
		assert.Equal(t, "new-exec-456", pipeline["newExecutionId"])

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://codepipeline.us-east-1.amazonaws.com/", httpContext.Requests[0].URL.String())
		body, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), `"pipelineName":"my-pipeline"`)
		assert.Contains(t, string(body), `"stageName":"Deploy"`)
		assert.Contains(t, string(body), `"pipelineExecutionId":"old-exec-123"`)
		assert.Contains(t, string(body), `"retryMode":"FAILED_ACTIONS"`)
	})
}

func Test__RetryStageExecution__Metadata(t *testing.T) {
	component := &RetryStageExecution{}

	assert.Equal(t, "aws.codepipeline.retryStageExecution", component.Name())
	assert.Equal(t, "CodePipeline • Retry Stage Execution", component.Label())
	assert.NotEmpty(t, component.Description())
	assert.Equal(t, "aws", component.Icon())
	assert.Equal(t, "orange", component.Color())

	fields := component.Configuration()
	require.GreaterOrEqual(t, len(fields), 5)

	fieldsByName := map[string]bool{}
	for _, f := range fields {
		fieldsByName[f.Name] = f.Required
	}

	assert.True(t, fieldsByName["region"])
	assert.True(t, fieldsByName["pipeline"])
	assert.True(t, fieldsByName["stage"])
	assert.True(t, fieldsByName["pipelineExecutionId"])
	assert.True(t, fieldsByName["retryMode"])

	channels := component.OutputChannels(nil)
	require.Len(t, channels, 1)
	assert.Equal(t, core.DefaultOutputChannel.Name, channels[0].Name)
}
