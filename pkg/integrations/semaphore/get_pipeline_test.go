package semaphore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/contexts"
)

func TestGetPipeline_Setup_RequiresPipelineID(t *testing.T) {
	component := &GetPipeline{}

	ctx := contexts.SetupContext{
		Configuration: map[string]any{},
	}

	err := component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline ID is required")
}

func TestGetPipeline_Setup_ValidConfiguration(t *testing.T) {
	component := &GetPipeline{}

	ctx := contexts.SetupContext{
		Configuration: map[string]any{
			"pipelineId": "00000000-0000-0000-0000-000000000000",
		},
	}

	err := component.Setup(ctx)
	assert.NoError(t, err)
}

func TestGetPipeline_Configuration(t *testing.T) {
	component := &GetPipeline{}

	fields := component.Configuration()
	assert.Len(t, fields, 1)
	assert.Equal(t, "pipelineId", fields[0].Name)
	assert.True(t, fields[0].Required)
}

func TestGetPipeline_OutputChannels(t *testing.T) {
	component := &GetPipeline{}

	channels := component.OutputChannels(nil)
	assert.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func TestGetPipeline_Metadata(t *testing.T) {
	component := &GetPipeline{}

	assert.Equal(t, "semaphore.getPipeline", component.Name())
	assert.Equal(t, "Get Pipeline", component.Label())
	assert.Equal(t, "Get a Semaphore pipeline by ID", component.Description())
	assert.Equal(t, "workflow", component.Icon())
	assert.Equal(t, "gray", component.Color())
}
