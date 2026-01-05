package annotation

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/registry"
)

const MaxAnnotationTextLength = 5000

func init() {
	registry.RegisterWidget("annotation", &Annotation{})
}

type Annotation struct{}

func (a *Annotation) Name() string {
	return "annotation"
}

func (a *Annotation) Label() string {
	return "Annotation"
}

func (a *Annotation) Description() string {
	return "Add text annotations and notes to your workflow for documentation and clarity"
}

func (a *Annotation) Icon() string {
	return "sticky-note"
}

func (a *Annotation) Color() string {
	return "yellow"
}

func (a *Annotation) Configuration() []configuration.Field {
	maxLength := MaxAnnotationTextLength
	return []configuration.Field{
		{
			Name:        "text",
			Label:       "Annotation Text",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Default:     "## Annotation\n\nHow to use annotations:\n\n- Annotations are displayed as sticky notes in the canvas editor\n- It supports basic markdown formatting",
			Description: "Text content for the annotation",
			TypeOptions: &configuration.TypeOptions{
				Text: &configuration.TextTypeOptions{
					MaxLength: &maxLength,
				},
			},
		},
	}
}
