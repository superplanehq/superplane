package factory

import (
	"net/http"

	"github.com/go-viper/mapstructure/v2"
	"github.com/google/uuid"
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
	WorkOrderID string `json:"workOrderId" mapstructure:"workOrderId"`
	Body        string `json:"body" mapstructure:"body"`
	AuthorKind  string `json:"authorKind" mapstructure:"authorKind"`
	AuthorLabel string `json:"authorLabel" mapstructure:"authorLabel"`
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
	return `The Add Work Order Comment component appends a comment (from a human, an LLM, or the system) to the work order's activity timeline. This component can only be used in factory-owned apps.`
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
			"workOrderId": "wo-123",
			"body":        "Ready for review",
			"authorKind":  "llm",
		},
	}
}

func (c *AddWorkOrderComment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddWorkOrderComment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "workOrderId",
			Label:       "Work Order ID",
			Description: "The ID of the work order to comment on",
			Type:        configuration.FieldTypeString,
			Required:    true,
		},
		{
			Name:        "body",
			Label:       "Comment",
			Description: "The comment body — plain text or markdown",
			Type:        configuration.FieldTypeText,
			Required:    true,
		},
		{
			Name:  "authorKind",
			Label: "Author Kind",
			//
			// Canvas automation is never a human — the "Human" (`user`)
			// author kind is deliberately absent here. Human attribution
			// only happens through the interactive API path, which reads
			// the authenticated caller's id. Offering `user` on the
			// canvas would leave the comment attributed to "Someone"
			// because there is no acting user to attach.
			//
			Description: "Where the comment came from (canvas comments are LLM or system, never a human)",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "llm",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "LLM", Value: "llm"},
						{Label: "System", Value: "system"},
					},
				},
			},
		},
		{
			Name:        "authorLabel",
			Label:       "Author Label",
			Description: "Optional display label (e.g. \"Claude\", \"Reviewer bot\")",
			Type:        configuration.FieldTypeString,
			Required:    false,
		},
	}
}

func (c *AddWorkOrderComment) Execute(ctx core.ExecutionContext) error {
	config := AddWorkOrderCommentConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}

	err := ctx.Factory.AddWorkOrderComment(core.AddWorkOrderCommentParams{
		WorkOrderID: config.WorkOrderID,
		Body:        config.Body,
		AuthorKind:  config.AuthorKind,
		AuthorLabel: config.AuthorLabel,
	})
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"workOrder.commentAdded",
		[]any{map[string]any{
			"workOrderId": config.WorkOrderID,
			"body":        config.Body,
			"authorKind":  config.AuthorKind,
		}},
	)
}

func (c *AddWorkOrderComment) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
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
