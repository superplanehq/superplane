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
	IssueID    string  `json:"issueId"`
	Status     *string `json:"status"`
	AssignedTo *string `json:"assignedTo"`
}

func (c *UpdateIssue) Name() string {
	return "sentry.updateIssue"
}

func (c *UpdateIssue) Label() string {
	return "Update Issue"
}

func (c *UpdateIssue) Description() string {
	return "Update an existing issue in Sentry"
}

func (c *UpdateIssue) Documentation() string {
	return `The Update Issue component modifies an existing Sentry issue.

## Use Cases

- **Status updates**: Update issue status (resolved, ignored, unresolved)
- **Assignment**: Assign issues to users

## Configuration

- **Issue ID**: The ID of the issue to update (e.g., 1234567890)
- **Status**: Update issue status (resolved, ignored, unresolved)
- **Assigned To**: Assign to a user ID or email

## Output

Returns the updated issue object with all current information.`
}

func (c *UpdateIssue) Icon() string {
	return "edit"
}

func (c *UpdateIssue) Color() string {
	return "purple"
}

func (c *UpdateIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "issueId",
			Label:       "Issue ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the issue to update (e.g., 1234567890)",
			Placeholder: "e.g., 1234567890",
		},
		{
			Name:        "status",
			Label:       "Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Update the issue status",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Resolved", Value: "resolved"},
						{Label: "Ignored", Value: "ignored"},
						{Label: "Unresolved", Value: "unresolved"},
					},
				},
			},
		},
		{
			Name:        "assignedTo",
			Label:       "Assigned To",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Assign to a user ID or email (leave empty to unassign)",
			Placeholder: "e.g., user@example.com or user_id",
		},
	}
}

func (c *UpdateIssue) Setup(ctx core.SetupContext) error {
	spec := UpdateIssueSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IssueID == "" {
		return errors.New("issueId is required")
	}

	// Validate that at least one field to update is provided
	if spec.Status == nil && spec.AssignedTo == nil {
		return errors.New("at least one field to update must be provided (status or assignedTo)")
	}

	// Store minimal metadata (no external API call needed for setup)
	return ctx.Metadata.Set(NodeMetadata{})
}

func (c *UpdateIssue) Execute(ctx core.ExecutionContext) error {
	spec := UpdateIssueSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	updateRequest := IssueUpdateRequest{}

	if spec.Status != nil {
		updateRequest.Status = spec.Status
	}

	if spec.AssignedTo != nil {
		updateRequest.AssignedTo = spec.AssignedTo
	}

	issue, err := client.UpdateIssue(spec.IssueID, updateRequest)
	if err != nil {
		return fmt.Errorf("failed to update issue: %v", err)
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
	return []core.Action{}
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
