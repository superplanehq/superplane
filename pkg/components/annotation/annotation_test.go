package annotation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

func TestAnnotation_Name(t *testing.T) {
	component := &Annotation{}
	assert.Equal(t, "annotation", component.Name())
}

func TestAnnotation_Label(t *testing.T) {
	component := &Annotation{}
	assert.Equal(t, "Annotation", component.Label())
}

func TestAnnotation_Description(t *testing.T) {
	component := &Annotation{}
	assert.Equal(t, "Display rich text annotation on the canvas (display-only)", component.Description())
}

func TestAnnotation_Icon(t *testing.T) {
	component := &Annotation{}
	assert.Equal(t, "sticky-note", component.Icon())
}

func TestAnnotation_Color(t *testing.T) {
	component := &Annotation{}
	assert.Equal(t, "gray", component.Color())
}

func TestAnnotation_OutputChannels(t *testing.T) {
	component := &Annotation{}
	channels := component.OutputChannels(nil)

	// Annotation should have no output channels (cannot be connected in flows)
	assert.Empty(t, channels)
}

func TestAnnotation_Configuration(t *testing.T) {
	component := &Annotation{}
	config := component.Configuration()

	// Should have exactly one field: content
	assert.Len(t, config, 1)

	// Verify content field properties
	contentField := config[0]
	assert.Equal(t, "content", contentField.Name)
	assert.Equal(t, "Annotation Content", contentField.Label)
	assert.Equal(t, configuration.FieldTypeText, contentField.Type)
	assert.Equal(t, "Rich text annotation to display on the canvas", contentField.Description)
	assert.False(t, contentField.Required)
	assert.Equal(t, "Enter your annotation here...", contentField.Placeholder)
}

func TestAnnotation_Actions(t *testing.T) {
	component := &Annotation{}
	actions := component.Actions()

	// Annotation should have no custom actions
	assert.Empty(t, actions)
}

func TestAnnotation_HandleAction(t *testing.T) {
	component := &Annotation{}
	ctx := core.ActionContext{}
	err := component.HandleAction(ctx)

	// Should return error as annotation doesn't support actions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "annotation does not support actions")
}

func TestAnnotation_Setup(t *testing.T) {
	component := &Annotation{}
	ctx := core.SetupContext{}
	err := component.Setup(ctx)

	// Setup should succeed with no validation
	assert.NoError(t, err)
}

func TestAnnotation_Cancel(t *testing.T) {
	component := &Annotation{}
	ctx := core.ExecutionContext{}
	err := component.Cancel(ctx)

	// Cancel should succeed (nothing to cancel)
	assert.NoError(t, err)
}
