package semaphore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetPipeline__Configuration(t *testing.T) {
	component := &GetPipeline{}

	t.Run("has correct name", func(t *testing.T) {
		assert.Equal(t, "semaphore.getPipeline", component.Name())
	})

	t.Run("has correct label", func(t *testing.T) {
		assert.Equal(t, "Get Pipeline", component.Label())
	})

	t.Run("returns configuration fields", func(t *testing.T) {
		fields := component.Configuration()
		require.NotEmpty(t, fields)

		assert.Len(t, fields, 1)
		assert.Equal(t, "pipelineId", fields[0].Name)
		assert.Equal(t, "Pipeline ID", fields[0].Label)
		assert.True(t, fields[0].Required)
	})
}

func Test__GetPipeline__Execute__Validation(t *testing.T) {
	component := &GetPipeline{}

	t.Run("returns error when pipelineId is missing", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://test.semaphoreci.com",
				"apiToken":        "test-token",
			},
		}

		ctx := core.ExecutionContext{
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{},
		}

		_, err := component.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pipelineId is required")
	})

	t.Run("returns error when pipelineId is empty string", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://test.semaphoreci.com",
				"apiToken":        "test-token",
			},
		}

		ctx := core.ExecutionContext{
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"pipelineId": "",
			},
		}

		_, err := component.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pipelineId is required")
	})
}

func Test__GetPipeline__OutputChannels(t *testing.T) {
	component := &GetPipeline{}

	t.Run("returns default output channel", func(t *testing.T) {
		channels := component.OutputChannels(nil)
		require.Len(t, channels, 1)
		assert.Equal(t, core.DefaultOutputChannel.Name, channels[0].Name)
	})
}
