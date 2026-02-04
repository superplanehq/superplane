package semaphore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core/contexts"
)

func TestListPipelines_Name(t *testing.T) {
	component := &ListPipelines{}
	assert.Equal(t, "semaphore.listPipelines", component.Name())
}

func TestListPipelines_Label(t *testing.T) {
	component := &ListPipelines{}
	assert.Equal(t, "List Pipelines", component.Label())
}

func TestListPipelines_Setup_RequiresProject(t *testing.T) {
	component := &ListPipelines{}
	ctx := contexts.SetupContext{
		Configuration: map[string]any{},
	}

	err := component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project is required")
}

func TestListPipelines_Setup_ValidatesLimitRange(t *testing.T) {
	component := &ListPipelines{}

	// Test limit too low
	limitZero := 0
	ctx := contexts.SetupContext{
		Configuration: map[string]any{
			"project": "my-project",
			"limit":   &limitZero,
		},
	}

	err := component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "limit must be between 1 and 100")

	// Test limit too high
	limitHigh := 101
	ctx = contexts.SetupContext{
		Configuration: map[string]any{
			"project": "my-project",
			"limit":   &limitHigh,
		},
	}

	err = component.Setup(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "limit must be between 1 and 100")
}

func TestListPipelines_Setup_AcceptsValidConfig(t *testing.T) {
	component := &ListPipelines{}
	limit := 50
	ctx := contexts.SetupContext{
		Configuration: map[string]any{
			"project":    "my-project",
			"branchName": "main",
			"limit":      &limit,
		},
	}

	err := component.Setup(ctx)
	assert.NoError(t, err)
}

func TestListPipelines_OutputChannels(t *testing.T) {
	component := &ListPipelines{}
	channels := component.OutputChannels()
	assert.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func TestListPipelines_Configuration(t *testing.T) {
	component := &ListPipelines{}
	fields := component.Configuration()

	// Should have 8 fields: project, branchName, ymlFilePath, createdAfter, createdBefore, doneAfter, doneBefore, limit
	assert.Len(t, fields, 8)

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
	assert.NotNil(t, projectField)
	assert.True(t, projectField.Required)
}
