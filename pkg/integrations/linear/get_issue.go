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

type GetIssue struct{}

type GetIssueSpec struct {
	Issue string `json:"issue" mapstructure:"issue"`
}

func (c *GetIssue) Name() string {
	return "linear.getIssue"
}

func (c *GetIssue) Label() string {
	return "Get Issue"
}

func (c *GetIssue) Description() string {
	return "Fetch a single issue from Linear"
}

func (c *GetIssue) Documentation() string {
	return `The Get Issue component fetches a single issue from Linear by its identifier or ID.

## Use Cases

- **Read state, then act**: Look up an issue's current status, assignee, or labels before deciding what to do next
- **Data enrichment**: Pull issue details into a workflow to combine with other information
- **Status checking**: Check whether an issue is done before performing an action

## Configuration

- **Issue** (required): The issue to fetch. Accepts either the human-readable identifier (e.g. ENG-142)
  or the issue's UUID. Supports expressions, so the value can come from an upstream event.

## Output

Returns the issue, including its ` + "`identifier`" + ` (e.g. ENG-142), ` + "`url`" + `, ` + "`title`" + `,
` + "`description`" + `, ` + "`state`" + `, ` + "`team`" + `, ` + "`assignee`" + `, ` + "`creator`" + `, ` + "`project`" + `,
` + "`priorityLabel`" + ` and ` + "`labels`" + `.

## Permissions

SuperPlane's OAuth connection includes the **read** scope, which covers reading issues. The issue must
be visible to the user who authorized the Linear connection.`
}

func (c *GetIssue) Icon() string {
	return "linear"
}

func (c *GetIssue) Color() string {
	return "indigo"
}

func (c *GetIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "issue",
			Label:       "Issue",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The issue to fetch, by identifier (e.g. ENG-142) or ID",
			Placeholder: "ENG-142",
		},
	}
}

func (c *GetIssue) Setup(ctx core.SetupContext) error {
	spec := GetIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if strings.TrimSpace(spec.Issue) == "" {
		return fmt.Errorf("issue is required")
	}

	return nil
}

func (c *GetIssue) Execute(ctx core.ExecutionContext) error {
	spec := GetIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	issueID := strings.TrimSpace(spec.Issue)
	if issueID == "" {
		return fmt.Errorf("issue is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	issue, err := client.GetIssue(issueID)
	if err != nil {
		return fmt.Errorf("failed to get issue: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		IssuePayloadType,
		[]any{issue},
	)
}

func (c *GetIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetIssue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetIssue) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
