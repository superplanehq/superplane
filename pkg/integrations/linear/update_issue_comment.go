package linear

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateIssueComment struct{}

type UpdateIssueCommentSpec struct {
	Issue   string `json:"issue" mapstructure:"issue"`
	Comment string `json:"comment" mapstructure:"comment"`
	Body    string `json:"body" mapstructure:"body"`
}

func (c *UpdateIssueComment) Name() string {
	return "linear.updateIssueComment"
}

func (c *UpdateIssueComment) Label() string {
	return "Update Issue Comment"
}

func (c *UpdateIssueComment) Description() string {
	return "Update an existing comment on an issue in Linear"
}

func (c *UpdateIssueComment) Documentation() string {
	return `The Update Issue Comment component edits an existing comment on a Linear issue.

## Use Cases

- **Living status**: Keep one comment up to date with the latest deployment or test results instead of
  posting a new one on every run
- **Progressive detail**: Start with a short note and append findings as a workflow discovers them
- **Correct mistakes**: Fix a comment an earlier step posted with incomplete information

## Configuration

- **Issue** (optional): The issue whose comments fill the picker below. Only used to populate the
  picker, so it can be left empty when the comment comes from an expression.
- **Comment** (required): The comment to update. Pick one from the issue, or switch the field to an
  expression to supply an ID resolved at run time.
- **Body** (required): The new comment text, written in Markdown. It is the only field Linear lets a
  comment update change, and it cannot be empty. Supports expressions.

## Finding the comment ID

Comments have no human-readable identifier, so this component needs the comment's UUID. Either pick it
from the issue, or carry it through from a previous step: the **Add Issue Comment** output and the
**On Issue Comment** trigger payload both expose ` + "`id`" + `.

## Output

Returns the updated comment, including its ` + "`id`" + `, ` + "`body`" + `, ` + "`url`" + `,
` + "`createdAt`" + `, ` + "`updatedAt`" + `, ` + "`editedAt`" + `, the ` + "`user`" + ` who authored
it, and the ` + "`issue`" + ` it belongs to. Linear sets ` + "`editedAt`" + ` on the first edit, so it
marks the comment as edited from then on.

## Permissions

Linear only lets the comment's author edit it, so the user who authorized the Linear connection must be
the one who posted the comment. SuperPlane's OAuth connection includes the **write** scope, which
covers editing comments.`
}

func (c *UpdateIssueComment) Icon() string {
	return "linear"
}

func (c *UpdateIssueComment) Color() string {
	return "indigo"
}

func (c *UpdateIssueComment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIssueComment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "issue",
			Label:       "Issue",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "The issue whose comments fill the picker below, by identifier (e.g. ENG-142) or ID",
			Placeholder: "ENG-142",
		},
		{
			Name:        "comment",
			Label:       "Comment",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The comment to update. Switch to an expression to supply an ID resolved at run time",
			Placeholder: "Select a comment",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeComment,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "issue",
							ValueFrom: &configuration.ParameterValueFrom{Field: "issue"},
						},
					},
				},
			},
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The new comment text, written in Markdown",
		},
	}
}

func (c *UpdateIssueComment) Setup(ctx core.SetupContext) error {
	spec := UpdateIssueCommentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if strings.TrimSpace(spec.Comment) == "" {
		return fmt.Errorf("comment is required")
	}

	if strings.TrimSpace(spec.Body) == "" {
		return fmt.Errorf("body is required")
	}

	return nil
}

func (c *UpdateIssueComment) Execute(ctx core.ExecutionContext) error {
	spec := UpdateIssueCommentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	//
	// The picker yields a comment ID directly, and an expression resolves to one
	// before Execute runs, so neither path needs a lookup here.
	//
	commentID := strings.TrimSpace(spec.Comment)
	if commentID == "" {
		return fmt.Errorf("comment is required")
	}

	//
	// Body is the only field Linear lets a comment update change, so it is
	// always required - Linear also rejects a comment with an empty body.
	//
	body := strings.TrimSpace(spec.Body)
	if body == "" {
		return fmt.Errorf("body is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	comment, err := client.UpdateComment(commentID, map[string]any{"body": body})
	if err != nil {
		return fmt.Errorf("failed to update comment: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CommentPayloadType,
		[]any{comment},
	)
}

func (c *UpdateIssueComment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateIssueComment) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIssueComment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateIssueComment) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateIssueComment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateIssueComment) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
