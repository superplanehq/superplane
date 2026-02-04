package semaphore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__GetPipeline__Setup(t *testing.T) {
	component := &GetPipeline{}

	t.Run("missing pipeline ID -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "pipeline ID is required")
	})

	t.Run("empty pipeline ID -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"pipelineId": "",
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "pipeline ID is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"pipelineId": "00000000-0000-0000-0000-000000000000",
			},
		})

		require.NoError(t, err)
	})
}

func Test__GetPipeline__Configuration(t *testing.T) {
	component := &GetPipeline{}

	fields := component.Configuration()
	require.Len(t, fields, 1)
	assert.Equal(t, "pipelineId", fields[0].Name)
	assert.True(t, fields[0].Required)
}

func Test__GetPipeline__OutputChannels(t *testing.T) {
	component := &GetPipeline{}

	channels := component.OutputChannels(nil)
	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func Test__GetPipeline__Metadata(t *testing.T) {
	component := &GetPipeline{}

	assert.Equal(t, "semaphore.getPipeline", component.Name())
	assert.Equal(t, "Get Pipeline", component.Label())
	assert.Equal(t, "Get a Semaphore pipeline by ID", component.Description())
	assert.Equal(t, "workflow", component.Icon())
	assert.Equal(t, "gray", component.Color())
}
