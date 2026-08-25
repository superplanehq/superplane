package annotation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func TestAnnotationMetadata(t *testing.T) {
	widget := &Annotation{}

	assert.Equal(t, "annotation", widget.Name())
	assert.Equal(t, "Annotation", widget.Label())
	assert.Equal(t, "Add text annotations and notes to your workflow for documentation and clarity", widget.Description())
	assert.Equal(t, "sticky-note", widget.Icon())
	assert.Equal(t, "yellow", widget.Color())
}

func TestAnnotationConfiguration(t *testing.T) {
	fields := (&Annotation{}).Configuration()

	require.Len(t, fields, 1)
	field := fields[0]
	assert.Equal(t, "text", field.Name)
	assert.Equal(t, "Annotation Text", field.Label)
	assert.Equal(t, configuration.FieldTypeText, field.Type)
	assert.True(t, field.Required)
	assert.NotEmpty(t, field.Default)
	assert.Equal(t, "Text content for the annotation", field.Description)
	require.NotNil(t, field.TypeOptions)
	require.NotNil(t, field.TypeOptions.Text)
	require.NotNil(t, field.TypeOptions.Text.MaxLength)
	assert.Equal(t, MaxAnnotationTextLength, *field.TypeOptions.Text.MaxLength)
}
