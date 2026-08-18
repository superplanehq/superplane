package linear

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type AddIssueComment struct{}

type AddIssueCommentSpec struct {
	Issue string `json:"issue" mapstructure:"issue"`
	Body  string `json:"body" mapstructure:"body"`
}

func (c *AddIssueComment) Name() string {
	return "linear.addIssueComment"
}

func (c *AddIssueComment) Label() string {
	return "Add Issue Comment"
}

func (c *AddIssueComment) Description() string {
	return "Add a comment to an issue in Linear"
}

func (c *AddIssueComment) Documentation() string {
	return `The Add Issue Comment component adds a comment to an existing Linear issue.

## Use Cases

- **Automated updates**: Post deployment status or workflow results onto the issue
- **Runbook linking**: Drop runbook links, error details, or context for responders
- **Cross-tool sync**: Mirror notes from Slack or another tracker into Linear

## Configuration

- **Issue** (required): The issue to comment on, by identifier (e.g. ENG-142) or ID. Supports
  expressions.
- **Body** (required): The comment text, written in Markdown. Supports expressions.

## Output

Returns the created comment, including its ` + "`id`" + `, ` + "`body`" + `, ` + "`url`" + `,
` + "`createdAt`" + `, the ` + "`user`" + ` who authored it, and the ` + "`issue`" + ` it was added to.

## Permissions

The user who authorized the Linear connection must be able to comment on the issue. SuperPlane's
OAuth connection includes the **write** scope, which covers creating comments.`
}

func (c *AddIssueComment) Icon() string {
	return "linear"
}

func (c *AddIssueComment) Color() string {
	return "indigo"
}

func (c *AddIssueComment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddIssueComment) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "issue",
			Label:       "Issue",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The issue to comment on, by identifier (e.g. ENG-142) or ID",
			Placeholder: "ENG-142",
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The comment text, written in Markdown",
		},
	}
}

func (c *AddIssueComment) Setup(ctx core.SetupContext) error {
	spec := AddIssueCommentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if strings.TrimSpace(spec.Issue) == "" {
		return fmt.Errorf("issue is required")
	}

	if strings.TrimSpace(spec.Body) == "" {
		return fmt.Errorf("body is required")
	}

	return nil
}

func (c *AddIssueComment) Execute(ctx core.ExecutionContext) error {
	spec := AddIssueCommentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	issueID := strings.TrimSpace(spec.Issue)
	if issueID == "" {
		return fmt.Errorf("issue is required")
	}

	body := strings.TrimSpace(spec.Body)
	if body == "" {
		return fmt.Errorf("body is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	// issueId accepts the human identifier (e.g. ENG-142), so no lookup is needed.
	comment, err := client.CreateComment(map[string]any{"issueId": issueID, "body": body})
	if err != nil {
		return fmt.Errorf("failed to add comment: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CommentPayloadType,
		[]any{comment},
	)
}

func (c *AddIssueComment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddIssueComment) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *AddIssueComment) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AddIssueComment) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AddIssueComment) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
