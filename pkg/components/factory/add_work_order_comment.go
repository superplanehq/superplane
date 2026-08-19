package factory

import (
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const AddWorkOrderCommentComponentName = "addWorkOrderComment"

func init() {
	registry.RegisterAction(AddWorkOrderCommentComponentName, &AddWorkOrderComment{})
}

type AddWorkOrderComment struct{}

type AddWorkOrderCommentConfiguration struct {
	OrderID string `json:"orderId" mapstructure:"orderId"`
	Body    string `json:"body" mapstructure:"body"`
}

func (c *AddWorkOrderComment) Name() string {
	return AddWorkOrderCommentComponentName
}

func (c *AddWorkOrderComment) Label() string {
	return "Add Work Order Comment"
}

func (c *AddWorkOrderComment) Description() string {
	return "Append a comment to a work order timeline"
}

func (c *AddWorkOrderComment) Documentation() string {
	return `The Add Work Order Comment component appends a comment from this automation to the work order's activity timeline. Authorship is captured automatically — the timeline shows which canvas node and app wrote the comment, without any configurable label.

` + "`orderId`" + ` explicitly targets the work order — it defaults to ` + "`{{ order().id }}`" + `, the work order driving the current run, which only resolves when the flow was dispatched from a factory line. In a flow triggered by an external event, replace it with e.g. ` + "`{{ previous().data.workOrder.id }}`" + ` after a ` + "`findWorkOrder`" + ` step. This component can only be used in factory-owned apps.`
}

func (c *AddWorkOrderComment) Icon() string {
	return "factory"
}

func (c *AddWorkOrderComment) Color() string {
	return "blue"
}

func (c *AddWorkOrderComment) ExampleOutput() map[string]any {
	return map[string]any{
		"timestamp": "2026-01-01T00:00:00Z",
		"type":      "workOrder.commentAdded",
		"data": map[string]any{
			"body": "Ready for review",
		},
	}
}

func (c *AddWorkOrderComment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddWorkOrderComment) Configuration() []configuration.Field {
	// Author identity is derived from the canvas node — no spoofable
	// label lives in the component config.
	return []configuration.Field{
		{
			Name:        "orderId",
			Label:       "Work Order ID",
			Description: "Work order to target. Defaults to the work order driving the current run (only resolves when this flow was dispatched from a factory line). Replace it with e.g. {{ previous().data.workOrder.id }} otherwise.",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "{{ order().id }}",
		},
		{
			Name:        "body",
			Label:       "Comment",
			Description: "The comment body — plain text or markdown",
			Type:        configuration.FieldTypeText,
			Required:    true,
		},
	}
}

func (c *AddWorkOrderComment) Execute(ctx core.ExecutionContext) error {
	config := AddWorkOrderCommentConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	err := ctx.Factory.AddWorkOrderComment(core.AddWorkOrderCommentParams{
		OrderID: config.OrderID,
		Body:    config.Body,
	})
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"workOrder.commentAdded",
		[]any{map[string]any{
			"body": config.Body,
		}},
	)
}

func (c *AddWorkOrderComment) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *AddWorkOrderComment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddWorkOrderComment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *AddWorkOrderComment) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AddWorkOrderComment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AddWorkOrderComment) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
