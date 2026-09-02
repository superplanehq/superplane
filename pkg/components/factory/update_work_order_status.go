package factory

import (
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const UpdateWorkOrderStatusComponentName = "updateWorkOrderStatus"

func init() {
	registry.RegisterAction(UpdateWorkOrderStatusComponentName, &UpdateWorkOrderStatus{})
}

type UpdateWorkOrderStatus struct{}

type UpdateWorkOrderStatusConfiguration struct {
	OrderID string `json:"orderId" mapstructure:"orderId"`
	Status  string `json:"status" mapstructure:"status"`
	Result  string `json:"result" mapstructure:"result"`
}

func (c *UpdateWorkOrderStatus) Name() string {
	return UpdateWorkOrderStatusComponentName
}

func (c *UpdateWorkOrderStatus) Label() string {
	return "Update Task Status"
}

func (c *UpdateWorkOrderStatus) Description() string {
	return "Transition a task between lifecycle states"
}

func (c *UpdateWorkOrderStatus) Documentation() string {
	return `The Update Task Status component transitions a task through the lifecycle: draft → open → closed, plus open ↔ draft (back to draft), closed → open (reopen), and draft → closed (abandon before dispatch). When closing, a result must be provided; from open any of completed / rejected / failed is valid, from draft only rejected is valid (an unopened task never ran).

` + "`orderId`" + ` explicitly targets the task — it defaults to ` + "`{{ order().id }}`" + `, the task driving the current run, which only resolves when the flow was dispatched from a factory line. In a flow triggered by an external event such as ` + "`github.onPullRequest`" + `, replace it with ` + "`{{ previous().data.workOrder.id }}`" + ` after a ` + "`findWorkOrder`" + ` step. This component can only be used in factory-owned apps.`
}

func (c *UpdateWorkOrderStatus) Icon() string {
	return "factory"
}

func (c *UpdateWorkOrderStatus) Color() string {
	return "blue"
}

func (c *UpdateWorkOrderStatus) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "workOrder.statusUpdated",
		"data": map[string]any{
			"workOrder": map[string]any{
				"id":     "wo-123",
				"title":  "Task 1",
				"state":  "closed",
				"result": "completed",
			},
		},
	}
}

func (c *UpdateWorkOrderStatus) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateWorkOrderStatus) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "orderId",
			Label:       "Task ID",
			Description: "Task to target. Defaults to the task driving the current run (only resolves when this flow was dispatched from a factory line). Replace it with e.g. {{ previous().data.workOrder.id }} otherwise.",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "{{ order().id }}",
		},
		{
			Name:        "status",
			Label:       "Status",
			Description: "The new lifecycle state for the task",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Draft", Value: "draft"},
						{Label: "Open", Value: "open"},
						{Label: "Closed", Value: "closed"},
					},
				},
			},
		},
		{
			Name:        "result",
			Label:       "Result",
			Description: "Required when closing the task",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "status", Values: []string{"closed"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "status", Values: []string{"closed"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Completed", Value: "completed"},
						{Label: "Rejected", Value: "rejected"},
						{Label: "Failed", Value: "failed"},
					},
				},
			},
		},
	}
}

func (c *UpdateWorkOrderStatus) Execute(ctx core.ExecutionContext) error {
	config := UpdateWorkOrderStatusConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	workOrder, changed, err := ctx.Factory.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		OrderID: config.OrderID,
		State:   config.Status,
		Result:  config.Result,
	})
	if err != nil {
		return err
	}

	// Re-runs land here with the work order already in the target state;
	// emitting `workOrder.statusUpdated` on a no-op would trick downstream
	// nodes into treating a replay as a fresh transition.
	if !changed {
		return ctx.ExecutionState.Pass()
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"workOrder.statusUpdated",
		[]any{map[string]any{
			"workOrder": workOrder,
		}},
	)
}

func (c *UpdateWorkOrderStatus) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateWorkOrderStatus) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateWorkOrderStatus) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateWorkOrderStatus) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateWorkOrderStatus) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateWorkOrderStatus) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
