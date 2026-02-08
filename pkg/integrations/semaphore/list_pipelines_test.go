package semaphore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__ListPipelines__Name(t *testing.T) {
	component := &ListPipelines{}
	assert.Equal(t, "semaphore.listPipelines", component.Name())
}

func Test__ListPipelines__Label(t *testing.T) {
	component := &ListPipelines{}
	assert.Equal(t, "List Pipelines", component.Label())
}

func Test__ListPipelines__OutputChannels(t *testing.T) {
	component := &ListPipelines{}
	channels := component.OutputChannels(nil)
	require.Len(t, channels, 1)
	assert.Equal(t, core.DefaultOutputChannel, channels[0])
}

func Test__ListPipelines__Configuration(t *testing.T) {
	component := &ListPipelines{}
	fields := component.Configuration()

	require.Len(t, fields, 9)

	// Project ID field
	assert.Equal(t, "projectId", fields[0].Name)
	assert.False(t, fields[0].Required)

	// Workflow ID field
	assert.Equal(t, "workflowId", fields[1].Name)
	assert.False(t, fields[1].Required)

	// Branch Name field
	assert.Equal(t, "branchName", fields[2].Name)
	assert.False(t, fields[2].Required)

	// YAML File Path field
	assert.Equal(t, "ymlFilePath", fields[3].Name)
	assert.False(t, fields[3].Required)

	// Created After field
	assert.Equal(t, "createdAfter", fields[4].Name)
	assert.False(t, fields[4].Required)

	// Created Before field
	assert.Equal(t, "createdBefore", fields[5].Name)
	assert.False(t, fields[5].Required)

	// Done After field
	assert.Equal(t, "doneAfter", fields[6].Name)
	assert.False(t, fields[6].Required)

	// Done Before field
	assert.Equal(t, "doneBefore", fields[7].Name)
	assert.False(t, fields[7].Required)

	// Result Limit field
	assert.Equal(t, "resultLimit", fields[8].Name)
	assert.False(t, fields[8].Required)
}

func Test__ListPipelines__Setup(t *testing.T) {
	component := &ListPipelines{}

	t.Run("requires project ID or workflow ID", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "either Project ID or Workflow ID is required")
	})

	t.Run("accepts project ID only", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectId": "a426b4db-1919-483d-926a-1234567890ab",
			},
		})

		require.NoError(t, err)
	})

	t.Run("accepts workflow ID only", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"workflowId": "65c398bb-57ab-4459-90b5-1234567890ab",
			},
		})

		require.NoError(t, err)
	})

	t.Run("accepts both project ID and workflow ID", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"projectId":  "a426b4db-1919-483d-926a-1234567890ab",
				"workflowId": "65c398bb-57ab-4459-90b5-1234567890ab",
			},
		})

		require.NoError(t, err)
	})
}
