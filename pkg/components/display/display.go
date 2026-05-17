package display

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterAction("display", &Display{})
}

type Display struct{}

type Spec struct {
	Message string `json:"message"`
	Color   string `json:"color"`
}

type Result struct {
	Message string `json:"message"`
	Color   string `json:"color"`
}

func (c *Display) Name() string {
	return "display"
}

func (c *Display) Label() string {
	return "Display"
}

func (c *Display) Description() string {
	return "Display a debug message from the latest execution"
}

func (c *Display) Documentation() string {
	return `The Display component displays a debug message from the latest execution.

## Use Cases

- **Debugging**: Display a message from the latest execution to help debug the workflow.
- **Notifications**: Display a message from the latest execution to notify the user.
- **Logging**: Display a message from the latest execution to log the workflow.
`
}

func (c *Display) Icon() string {
	return "monitor"
}

func (c *Display) Color() string {
	return "gray"
}

func (c *Display) ExampleOutput() map[string]any {
	return ExampleOutput()
}

func (c *Display) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *Display) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "message",
			Label:       "Message",
			Type:        configuration.FieldTypeText,
			Description: "Text to display. Supports {{ }} expressions.",
			Required:    true,
		},
		{
			Name:        "color",
			Label:       "Color",
			Type:        configuration.FieldTypeSelect,
			Description: "Background color of the display.",
			Required:    true,
			Default:     "gray",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{
							Label:       "Gray",
							Value:       "gray",
							Description: "Gray background color.",
						},
						{
							Label:       "Green",
							Value:       "green",
							Description: "Green background color.",
						},
						{
							Label:       "Red",
							Value:       "red",
							Description: "Red background color.",
						},
						{
							Label:       "Yellow",
							Value:       "yellow",
							Description: "Yellow background color.",
						},
						{
							Label:       "Blue",
							Value:       "blue",
							Description: "Blue background color.",
						},
						{
							Label:       "Calculate color",
							Value:       "calculate_color",
							Description: "Calculate background color using {{ }} expressions.",
						},
					},
				},
			},
		},
		{
			Name:        "color_expression",
			Label:       "Color Expression",
			Type:        configuration.FieldTypeString,
			Placeholder: "{{ previous().data.result == 'success' ? 'green' : 'red' }}",
			Required:    false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "color", Values: []string{"calculate_color"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "color", Values: []string{"calculate_color"}},
			},
		},
	}
}

type DisplayExecutionResult struct {
	Message string `json:"message"`
	Color   string `json:"color"`
}

func (c *Display) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	displayExecutionResult := DisplayExecutionResult{
		Message: spec.Message,
		Color:   spec.Color,
	}

	if err := ctx.Metadata.Set(displayExecutionResult); err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"display.executed",
		[]any{map[string]any{}},
	)
}

func (c *Display) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *Display) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *Display) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *Display) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *Display) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *Display) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *Display) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
