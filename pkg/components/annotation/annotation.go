package annotation

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "annotation"

func init() {
	registry.RegisterComponent(ComponentName, &Annotation{})
}

type Annotation struct{}

func (c *Annotation) Name() string {
	return ComponentName
}

func (c *Annotation) Label() string {
	return "Annotation"
}

func (c *Annotation) Description() string {
	return "Display rich text annotation on the canvas (display-only)"
}

func (c *Annotation) Icon() string {
	return "sticky-note"
}

func (c *Annotation) Color() string {
	return "gray"
}

func (c *Annotation) OutputChannels(configuration any) []core.OutputChannel {
	// No output channels - component cannot be connected in flows
	return []core.OutputChannel{}
}

func (c *Annotation) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "content",
			Label:       "Annotation Content",
			Type:        configuration.FieldTypeText,
			Description: "Rich text annotation to display on the canvas",
			Required:    false,
			Placeholder: "Enter your annotation here...",
		},
	}
}

func (c *Annotation) Setup(ctx core.SetupContext) error {
	// No setup required for display-only component
	return nil
}

func (c *Annotation) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	// Annotation is display-only - dequeue without creating execution
	if err := ctx.DequeueItem(); err != nil {
		return nil, err
	}
	return nil, nil
}

func (c *Annotation) Execute(ctx core.ExecutionContext) error {
	// Pass through without emitting any events
	return ctx.ExecutionStateContext.Pass()
}

func (c *Annotation) Actions() []core.Action {
	// No custom actions for display-only component
	return []core.Action{}
}

func (c *Annotation) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("annotation does not support actions")
}

func (c *Annotation) Cancel(ctx core.ExecutionContext) error {
	// Nothing to cancel for display-only component
	return nil
}

func (c *Annotation) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
