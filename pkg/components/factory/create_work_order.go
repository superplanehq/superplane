package factory

import (
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "createWorkOrder"

func init() {
	registry.RegisterAction(ComponentName, &CreateWorkOrder{})
}

type CreateWorkOrder struct{}

type CreateWorkOrderConfiguration struct {
	Title       string `json:"title" mapstructure:"title"`
	Description string `json:"description" mapstructure:"description"`
}

func (c *CreateWorkOrder) Name() string {
	return ComponentName
}

func (c *CreateWorkOrder) Label() string {
	return "Create Work Order"
}

func (c *CreateWorkOrder) Description() string {
	return "Create a new work order"
}

func (c *CreateWorkOrder) Documentation() string {
	return `The Create Work Order component creates a new work order in the factory. This component can only be used in factory-owned apps.`
}

func (c *CreateWorkOrder) Icon() string {
	return "factory"
}

func (c *CreateWorkOrder) Color() string {
	return "blue"
}

func (c *CreateWorkOrder) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "workOrder.created",
		"data": map[string]any{
			"workOrder": map[string]any{
				"id":          "123",
				"title":       "Work Order 1",
				"description": "Work Order 1 description",
			},
		},
	}
}

func (c *CreateWorkOrder) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateWorkOrder) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "title",
			Label:       "Title",
			Description: "The title of the work order",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
		{
			Name:        "description",
			Label:       "Description",
			Description: "The description of the work order",
			Type:        configuration.FieldTypeString,
			Required:    false,
		},
	}
}

func (c *CreateWorkOrder) Execute(ctx core.ExecutionContext) error {
	config := CreateWorkOrderConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	workOrder, err := ctx.Factory.CreateWorkOrder(core.WorkOrderParams{
		Title:       config.Title,
		Description: config.Description,
	})

	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"workOrder.created",
		[]any{map[string]any{
			"workOrder": workOrder,
		}},
	)
}

func (c *CreateWorkOrder) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateWorkOrder) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateWorkOrder) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateWorkOrder) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateWorkOrder) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateWorkOrder) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
