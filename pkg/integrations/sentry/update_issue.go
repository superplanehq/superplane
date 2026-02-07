package sentry

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateIssue struct{}

type UpdateIssueSpec struct {
	Organization string `json:"organization"`
	IssueID      string `json:"issueId"`
	Status       string `json:"status"`
	AssignedTo   string `json:"assignedTo"`
}

func (c *UpdateIssue) Name() string {
	return "sentry.updateIssue"
}

func (c *UpdateIssue) Label() string {
	return "Update Issue"
}

func (c *UpdateIssue) Description() string {
	return "Update a Sentry issue (resolve, assign, ignore, etc.)"
}

func (c *UpdateIssue) Documentation() string {
	return `The Update Issue component updates an existing Sentry issue.

## Use Cases

- **Resolve after deploy**: Resolve the issue when a fix is deployed
- **Assignment**: Assign the issue to a user or team from a workflow
- **Ignore**: Ignore or archive issues based on workflow logic

## Configuration

- **Organization**: Sentry organization slug (e.g. from trigger payload or your org name)
- **Issue ID**: The issue ID to update (e.g. from trigger: ` + "`$['On Issue Event'].issue.id`" + `)
- **Status**: resolved, resolvedInNextRelease, unresolved, or ignored
- **Assigned To**: Actor id or username to assign the issue to

## Output

Returns the updated issue object from Sentry.`
}

func (c *UpdateIssue) Icon() string {
	return "edit"
}

func (c *UpdateIssue) Color() string {
	return "gray"
}

func (c *UpdateIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIssue) ExampleOutput() map[string]any {
	return exampleOutputUpdateIssue()
}

func (c *UpdateIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "organization",
			Label:       "Organization",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Sentry organization slug",
			Placeholder: "my-org",
		},
		{
			Name:        "issueId",
			Label:       "Issue ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The Sentry issue ID to update (e.g. from trigger payload)",
			Placeholder: "1234567890",
		},
		{
			Name:        "status",
			Label:       "Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "New issue status",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Resolved", Value: "resolved"},
						{Label: "Resolved in next release", Value: "resolvedInNextRelease"},
						{Label: "Unresolved", Value: "unresolved"},
						{Label: "Ignored", Value: "ignored"},
					},
				},
			},
		},
		{
			Name:        "assignedTo",
			Label:       "Assigned To",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Actor id or username to assign the issue to",
			Placeholder: "user@example.com",
		},
	}
}

func (c *UpdateIssue) Setup(ctx core.SetupContext) error {
	var spec UpdateIssueSpec
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}
	if spec.Organization == "" {
		return errors.New("organization is required")
	}
	if spec.IssueID == "" {
		return errors.New("issueId is required")
	}
	hasUpdate := spec.Status != "" || spec.AssignedTo != ""
	if !hasUpdate {
		return errors.New("at least one of status or assignedTo must be set")
	}
	return ctx.Metadata.Set(NodeMetadata{})
}

func (c *UpdateIssue) Execute(ctx core.ExecutionContext) error {
	var spec UpdateIssueSpec
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	req := UpdateIssueRequest{
		Status:     spec.Status,
		AssignedTo: spec.AssignedTo,
	}
	issue, err := client.UpdateIssue(spec.Organization, spec.IssueID, req)
	if err != nil {
		return fmt.Errorf("update issue: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"sentry.issue",
		[]any{issue},
	)
}

func (c *UpdateIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIssue) Actions() []core.Action {
	return nil
}

func (c *UpdateIssue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *UpdateIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}
