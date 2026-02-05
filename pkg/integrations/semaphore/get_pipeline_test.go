package semaphore

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetPipeline__Setup(t *testing.T) {
	component := GetPipeline{}

	t.Run("valid configuration decodes successfully", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"pipelineId": "test-pipeline-id",
			},
		})

		require.NoError(t, err)
	})

	t.Run("empty pipelineId is allowed in setup (validated at execution)", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"pipelineId": "",
			},
		})

		require.NoError(t, err)
	})

	t.Run("invalid configuration returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})
}

func Test__GetPipeline__Execute(t *testing.T) {
	component := GetPipeline{}

	t.Run("returns error when pipelineId is empty", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://test.semaphoreci.com",
				"apiToken":        "test-token",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Configuration: map[string]any{
				"pipelineId": "",
			},
		})

		require.ErrorContains(t, err, "pipeline ID is required")
	})

	t.Run("returns error when pipelineId is nil", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://test.semaphoreci.com",
				"apiToken":        "test-token",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Configuration: map[string]any{
				"pipelineId": nil,
			},
		})

		require.ErrorContains(t, err, "pipeline ID is required")
	})

	t.Run("emits pipeline data on success", func(t *testing.T) {
		pipelineResponse := `{
			"pipeline": {
				"name": "Test Pipeline",
				"ppl_id": "test-pipeline-123",
				"wf_id": "test-workflow-456",
				"state": "done",
				"result": "passed"
			}
		}`

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(pipelineResponse)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://test.semaphoreci.com",
				"apiToken":        "test-token",
			},
		}

		executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: executionStateCtx,
			Configuration: map[string]any{
				"pipelineId": "test-pipeline-123",
			},
		})

		require.NoError(t, err)
		assert.True(t, executionStateCtx.Finished)
		assert.True(t, executionStateCtx.Passed)
		assert.Equal(t, "default", executionStateCtx.Channel)
		assert.Equal(t, "semaphore.pipeline", executionStateCtx.Type)

		// Verify the HTTP request was made correctly
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "GET", httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/api/v1alpha/pipelines/test-pipeline-123")
	})

	t.Run("returns error when API call fails", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewBufferString(`{"error": "not found"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://test.semaphoreci.com",
				"apiToken":        "test-token",
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Configuration: map[string]any{
				"pipelineId": "nonexistent-pipeline",
			},
		})

		require.ErrorContains(t, err, "failed to get pipeline")
	})

	t.Run("returns error when client creation fails", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           &contexts.HTTPContext{},
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Configuration: map[string]any{
				"pipelineId": "test-pipeline-123",
			},
		})

		require.ErrorContains(t, err, "failed to create client")
	})
}

func Test__GetPipeline__Metadata(t *testing.T) {
	component := GetPipeline{}

	t.Run("Name returns correct value", func(t *testing.T) {
		assert.Equal(t, "semaphore.getPipeline", component.Name())
	})

	t.Run("Label returns correct value", func(t *testing.T) {
		assert.Equal(t, "Get Pipeline", component.Label())
	})

	t.Run("Description returns correct value", func(t *testing.T) {
		assert.Equal(t, "Fetch a Semaphore pipeline by ID", component.Description())
	})

	t.Run("Icon returns workflow", func(t *testing.T) {
		assert.Equal(t, "workflow", component.Icon())
	})

	t.Run("Color returns gray", func(t *testing.T) {
		assert.Equal(t, "gray", component.Color())
	})

	t.Run("OutputChannels returns default channel", func(t *testing.T) {
		channels := component.OutputChannels(nil)
		require.Len(t, channels, 1)
		assert.Equal(t, core.DefaultOutputChannel, channels[0])
	})

	t.Run("Actions returns empty list", func(t *testing.T) {
		actions := component.Actions()
		assert.Empty(t, actions)
	})

	t.Run("Configuration returns pipelineId field", func(t *testing.T) {
		config := component.Configuration()
		require.Len(t, config, 1)
		assert.Equal(t, "pipelineId", config[0].Name)
		assert.Equal(t, "Pipeline ID", config[0].Label)
		assert.True(t, config[0].Required)
	})
}
