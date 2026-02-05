package semaphore

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetPipeline__Name(t *testing.T) {
	component := &GetPipeline{}
	assert.Equal(t, "semaphore.getPipeline", component.Name())
}

func Test__GetPipeline__Label(t *testing.T) {
	component := &GetPipeline{}
	assert.Equal(t, "Get Pipeline", component.Label())
}

func Test__GetPipeline__Configuration(t *testing.T) {
	component := &GetPipeline{}
	config := component.Configuration()

	require.Len(t, config, 1)
	assert.Equal(t, "pipelineId", config[0].Name)
	assert.Equal(t, "Pipeline ID", config[0].Label)
	assert.True(t, config[0].Required)
}

func Test__GetPipeline__OutputChannels(t *testing.T) {
	component := &GetPipeline{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, GetPipelineOutputChannel, channels[0].Name)
	assert.Equal(t, "Default", channels[0].Label)
}

func Test__GetPipeline__Setup(t *testing.T) {
	component := &GetPipeline{}

	t.Run("pipeline ID is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"pipelineId": "",
			},
		})
		require.ErrorContains(t, err, "pipeline ID is required")
	})

	t.Run("valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"pipelineId": "test-pipeline-123",
			},
		})
		require.NoError(t, err)
	})

	t.Run("invalid configuration type", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid-config",
		})
		require.ErrorContains(t, err, "failed to decode configuration")
	})
}

func Test__GetPipeline__Execute(t *testing.T) {
	component := &GetPipeline{}

	t.Run("successful execution", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"pipeline":{"ppl_id":"test-pipeline-123","wf_id":"wf-456","name":"test-pipeline","state":"done","result":"passed"}}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"pipelineId": "test-pipeline-123",
			},
			ExecutionState: executionState,
			HTTP:           httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"organizationUrl": "https://test.semaphoreci.com",
					"apiToken":      "test-token",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, 1, len(executionState.Payloads))
		assert.Equal(t, GetPipelineOutputChannel, executionState.Channel)
		assert.Equal(t, PipelinePayloadType, executionState.Type)
	})

	t.Run("API error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message":"Pipeline not found"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"pipelineId": "test-pipeline-123",
			},
			ExecutionState: &contexts.ExecutionStateContext{
				KVs: make(map[string]string),
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"organizationUrl": "https://test.semaphoreci.com",
					"apiToken":      "test-token",
				},
			},
		})

		require.ErrorContains(t, err, "error fetching pipeline")
	})
}

func Test__GetPipeline__Cleanup(t *testing.T) {
	component := &GetPipeline{}
	err := component.Cleanup(core.SetupContext{})
	assert.NoError(t, err)
}
