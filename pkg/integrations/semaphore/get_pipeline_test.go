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

func Test__GetPipeline__Setup(t *testing.T) {
	component := &GetPipeline{}

	t.Run("pipelineId is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: GetPipelineSpec{PipelineID: ""},
		})

		require.ErrorContains(t, err, "pipelineId is required")
	})

	t.Run("valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: GetPipelineSpec{PipelineID: "pipeline-123"},
		})

		require.NoError(t, err)
	})

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})
}

func Test__GetPipeline__Execute(t *testing.T) {
	component := &GetPipeline{}

	t.Run("successful pipeline fetch", func(t *testing.T) {
		responseBody := `{
			"pipeline": {
				"ppl_id": "pipeline-123",
				"name": "Build Pipeline",
				"wf_id": "workflow-456",
				"state": "done",
				"result": "passed",
				"result_reason": "test",
				"yaml_file_name": "semaphore.yml",
				"working_directory": ".semaphore"
			}
		}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseBody)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Configuration:  GetPipelineSpec{PipelineID: "pipeline-123"},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Finished)
		assert.True(t, executionState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, GetPipelinePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		// The mock wraps payloads as {type, timestamp, data}
		wrapped := executionState.Payloads[0].(map[string]any)
		pipelineData := wrapped["data"].(map[string]any)
		assert.Equal(t, "pipeline-123", pipelineData["ppl_id"])
		assert.Equal(t, "Build Pipeline", pipelineData["name"])
		assert.Equal(t, "workflow-456", pipelineData["wf_id"])
		assert.Equal(t, "done", pipelineData["state"])
		assert.Equal(t, "passed", pipelineData["result"])

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://example.semaphoreci.com/api/v1alpha/pipelines/pipeline-123", httpContext.Requests[0].URL.String())
	})

	t.Run("API error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("not found")),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Configuration:  GetPipelineSpec{PipelineID: "invalid-id"},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get pipeline")
		assert.False(t, executionState.Finished)
	})
}
