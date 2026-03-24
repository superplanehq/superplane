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

type UpdateIssueNodeMetadata struct {
	IssueTitle string `json:"issueTitle,omitempty" mapstructure:"issueTitle"`
}

type UpdateIssueConfiguration struct {
	IssueID string `json:"issueId" mapstructure:"issueId"`
	Status  string `json:"status" mapstructure:"status"`
}

func (c *UpdateIssue) Name() string {
	return "sentry.updateIssue"
}

func (c *UpdateIssue) Label() string {
	return "Update Issue"
}

func (c *UpdateIssue) Description() string {
	return "Update a Sentry issue status"
}

func (c *UpdateIssue) Documentation() string {
	return `The Update Issue component updates an existing issue in Sentry.

## Use Cases

- **Resolve issues automatically** after a remediation workflow succeeds
- **Reopen issues** when a related deployment regresses

## Configuration

- **Issue**: Select the Sentry issue to update
- **Status**: New issue status

## Output

Returns the updated Sentry issue object.`
}

func (c *UpdateIssue) Icon() string {
	return "bug"
}

func (c *UpdateIssue) Color() string {
	return "gray"
}

func (c *UpdateIssue) ExampleOutput() map[string]any {
	return sentryIssueExample()
}

func (c *UpdateIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "issueId",
			Label:       "Issue",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the Sentry issue to update",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeIssue,
				},
			},
		},
		{
			Name:        "status",
			Label:       "Status",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "The new issue status",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Unresolved", Value: "unresolved"},
						{Label: "Resolved", Value: "resolved"},
						{Label: "Resolved In Next Release", Value: "resolvedInNextRelease"},
						{Label: "Ignored", Value: "ignored"},
					},
				},
			},
		},
	}
}

func (c *UpdateIssue) Setup(ctx core.SetupContext) error {
	config := UpdateIssueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.IssueID == "" {
		return errors.New("issueId is required")
	}

	if config.Status == "" {
		return errors.New("status is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	issue, err := client.GetIssue(config.IssueID)
	if err != nil {
		return fmt.Errorf("failed to retrieve sentry issue: %w", err)
	}

	return ctx.Metadata.Set(UpdateIssueNodeMetadata{
		IssueTitle: normalizedIssueTitle(issue.Title),
	})
}

func (c *UpdateIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIssue) Execute(ctx core.ExecutionContext) error {
	config := UpdateIssueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	issue, err := client.UpdateIssue(config.IssueID, UpdateIssueRequest{
		Status: config.Status,
	})
	if err != nil {
		return fmt.Errorf("failed to update sentry issue: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "sentry.issue", []any{issue})
}

func (c *UpdateIssue) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateIssue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}
