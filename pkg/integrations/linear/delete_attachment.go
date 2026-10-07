package linear

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteAttachment struct{}

type DeleteAttachmentSpec struct {
	Issue      string `json:"issue" mapstructure:"issue"`
	Attachment string `json:"attachment" mapstructure:"attachment"`
}

// DeleteAttachmentResult is the emitted payload. Linear's attachmentDelete
// returns only a success flag, so the deleted ID is echoed back for downstream steps.
type DeleteAttachmentResult struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

func (c *DeleteAttachment) Name() string {
	return "linear.deleteAttachment"
}

func (c *DeleteAttachment) Label() string {
	return "Delete Attachment"
}

func (c *DeleteAttachment) Description() string {
	return "Remove an attachment from an issue in Linear"
}

func (c *DeleteAttachment) Documentation() string {
	return `The Delete Attachment component removes a link attachment from a Linear issue.

## Use Cases

- **Clear stale links**: Drop a preview environment attachment once the environment is torn down
- **Tidy up**: Remove an attachment a workflow added earlier in the run
- **Replace**: Delete an attachment before attaching a different URL in its place

## Configuration

- **Issue** (optional): The issue whose attachments fill the picker below. Only used to populate the
  picker, so it can be left empty when the attachment comes from an expression.
- **Attachment** (required): The attachment to delete. Pick one from the issue, or switch the field
  to an expression to supply an ID resolved at run time.

## Finding the attachment ID

Linear has no delete-by-URL, so this component needs the attachment's ID. Either pick it from the
issue, or carry it through from a previous step: the **Create Attachment** output and the
**On Issue Attachment** trigger payload both expose ` + "`id`" + `.

## Output

Returns the deleted attachment's ` + "`id`" + ` and a ` + "`deleted`" + ` flag.

## Permissions

The user who authorized the Linear connection must be able to edit the issue. SuperPlane's OAuth
connection includes the **write** scope, which covers deleting attachments.`
}

func (c *DeleteAttachment) Icon() string {
	return "linear"
}

func (c *DeleteAttachment) Color() string {
	return "indigo"
}

func (c *DeleteAttachment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteAttachment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "issue",
			Label:       "Issue",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The issue whose attachments fill the picker below, by identifier (e.g. ENG-142) or ID",
			Placeholder: "ENG-142",
		},
		{
			Name:        "attachment",
			Label:       "Attachment",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The attachment to delete. Switch to an expression to supply an ID resolved at run time",
			Placeholder: "Select an attachment",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeAttachment,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "issue",
							ValueFrom: &configuration.ParameterValueFrom{Field: "issue"},
						},
					},
				},
			},
		},
	}
}

func (c *DeleteAttachment) Setup(ctx core.SetupContext) error {
	spec := DeleteAttachmentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if strings.TrimSpace(spec.Attachment) == "" {
		return fmt.Errorf("attachment is required")
	}

	return nil
}

func (c *DeleteAttachment) Execute(ctx core.ExecutionContext) error {
	spec := DeleteAttachmentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	//
	// The picker yields an attachment ID directly, and an expression resolves to
	// one before Execute runs, so neither path needs a lookup here.
	//
	attachmentID := strings.TrimSpace(spec.Attachment)
	if attachmentID == "" {
		return fmt.Errorf("attachment is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	if err := client.DeleteAttachment(attachmentID); err != nil {
		return fmt.Errorf("failed to delete attachment: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		AttachmentPayloadType,
		[]any{&DeleteAttachmentResult{ID: attachmentID, Deleted: true}},
	)
}

func (c *DeleteAttachment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteAttachment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteAttachment) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteAttachment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteAttachment) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
