package semaphore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__ListPipelines__Setup(t *testing.T) {
	component := &ListPipelines{}

	t.Run("missing project -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("empty project -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project": "",
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("limit too low -> error", func(t *testing.T) {
		limit := 0
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project": "my-project",
				"limit":   &limit,
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "limit must be between 1 and 100")
	})

	t.Run("limit too high -> error", func(t *testing.T) {
		limit := 101
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project": "my-project",
				"limit":   &limit,
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "limit must be between 1 and 100")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project": "my-project",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration with all options -> success", func(t *testing.T) {
		limit := 50
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project":    "my-project",
				"branchName": "main",
				"limit":      &limit,
			},
		})

		require.NoError(t, err)
	})
}

func Test__ListPipelines__Configuration(t *testing.T) {
	component := &ListPipelines{}

	fields := component.Configuration()
	// Should have 8 fields: project, branchName, ymlFilePath, createdAfter, createdBefore, doneAfter, doneBefore, limit
	require.Len(t, fields, 8)

	// Check project field is required
	var projectField *struct {
		Name     string
		Required bool
	}
	for _, f := range fields {
		if f.Name == "project" {
			projectField = &struct {
				Name     string
				Required bool
			}{Name: f.Name, Required: f.Required}
			break
		}
	}
	require.NotNil(t, projectField)
	assert.True(t, projectField.Required)
}

func Test__ListPipelines__OutputChannels(t *testing.T) {
	component := &ListPipelines{}

	channels := component.OutputChannels(nil)
	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func Test__ListPipelines__Metadata(t *testing.T) {
	component := &ListPipelines{}

	assert.Equal(t, "semaphore.listPipelines", component.Name())
	assert.Equal(t, "List Pipelines", component.Label())
	assert.Equal(t, "list", component.Icon())
	assert.Equal(t, "gray", component.Color())
}
