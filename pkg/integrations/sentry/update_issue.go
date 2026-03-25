package sentry

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateIssue struct{}

type UpdateIssueNodeMetadata struct {
	IssueTitle    string `json:"issueTitle,omitempty" mapstructure:"issueTitle"`
	AssigneeLabel string `json:"assigneeLabel,omitempty" mapstructure:"assigneeLabel"`
}

type UpdateIssueConfiguration struct {
	IssueID      string `json:"issueId" mapstructure:"issueId"`
	Status       string `json:"status" mapstructure:"status"`
	Priority     string `json:"priority" mapstructure:"priority"`
	AssignedTo   string `json:"assignedTo" mapstructure:"assignedTo"`
	HasSeen      *bool  `json:"hasSeen,omitempty" mapstructure:"hasSeen"`
	IsPublic     *bool  `json:"isPublic,omitempty" mapstructure:"isPublic"`
	IsSubscribed *bool  `json:"isSubscribed,omitempty" mapstructure:"isSubscribed"`
}

func (c *UpdateIssue) Name() string {
	return "sentry.updateIssue"
}

func (c *UpdateIssue) Label() string {
	return "Update Issue"
}

func (c *UpdateIssue) Description() string {
	return "Update a Sentry issue status, priority, assignee, or visibility flags"
}

func (c *UpdateIssue) Documentation() string {
	return `The Update Issue component updates an existing issue in Sentry.

## Use Cases

- **Resolve issues automatically** after a remediation workflow succeeds
- **Reopen issues** when a related deployment regresses
- **Route ownership** by assigning issues to a user or team
- **Escalate triage** by changing issue priority
- **Mark issues reviewed** after automation handles the first response
- **Manage visibility and subscriptions** for follow-up workflows

## Configuration

- **Issue**: Select the Sentry issue to update
- **Status**: Optional new issue status
- **Priority**: Optional issue priority
- **Assigned To**: Optional assignee from the selected issue's Sentry project
- **Seen**: Optional reviewed flag for the connected user
- **Public**: Optional issue sharing visibility
- **Subscribed**: Optional workflow subscription for the connected user

## Output

Returns the updated Sentry issue object.`
}

func (c *UpdateIssue) Icon() string {
	return "bug"
}

func (c *UpdateIssue) Color() string {
	return "gray"
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
			Required:    false,
			Description: "Optional new issue status",
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
		{
			Name:        "priority",
			Label:       "Priority",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Optional new issue priority",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "High", Value: "high"},
						{Label: "Medium", Value: "medium"},
						{Label: "Low", Value: "low"},
					},
				},
			},
		},
		{
			Name:        "assignedTo",
			Label:       "Assigned To",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional assignee from the issue's Sentry project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeAssignee,
					Parameters: []configuration.ParameterRef{
						{
							Name: "issueId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "issueId",
							},
						},
					},
				},
			},
		},
		{
			Name:        "hasSeen",
			Label:       "Seen",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Togglable:   false,
			Description: "Optionally mark the issue as seen for the connected user",
		},
		{
			Name:        "isPublic",
			Label:       "Public",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Togglable:   false,
			Description: "Optionally make the issue public or private",
		},
		{
			Name:        "isSubscribed",
			Label:       "Subscribed",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Togglable:   false,
			Description: "Optionally subscribe or unsubscribe the connected user",
		},
	}
}

func (c *UpdateIssue) Setup(ctx core.SetupContext) error {
	config, err := decodeUpdateIssueConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	normalizeUpdateIssueConfiguration(&config)

	if err := validateUpdateIssueConfiguration(config); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	issue, err := client.GetIssue(config.IssueID)
	if err != nil {
		return fmt.Errorf("failed to retrieve sentry issue: %w", err)
	}

	nodeMetadata := UpdateIssueNodeMetadata{
		IssueTitle: displayIssueLabel(issue.ShortID, issue.Title),
	}

	if config.AssignedTo != "" && issue.Project != nil && strings.TrimSpace(issue.Project.Slug) != "" {
		assignees, err := client.ListProjectAssignees(issue.Project.Slug)
		if err != nil {
			return fmt.Errorf("failed to list sentry issue assignees: %w", err)
		}

		for _, assignee := range assignees {
			if assignee.ID == config.AssignedTo {
				nodeMetadata.AssigneeLabel = assignee.Name
				break
			}
		}
	}

	return ctx.Metadata.Set(nodeMetadata)
}

func validateUpdateIssueConfiguration(config UpdateIssueConfiguration) error {
	if config.IssueID == "" {
		return errors.New("issueId is required")
	}

	if config.Status == "" &&
		config.Priority == "" &&
		config.AssignedTo == "" &&
		config.HasSeen == nil &&
		config.IsPublic == nil &&
		config.IsSubscribed == nil {
		return errors.New("at least one field to update must be provided")
	}

	return nil
}

func normalizeUpdateIssueConfiguration(config *UpdateIssueConfiguration) {
	if config == nil {
		return
	}

	config.IssueID = strings.TrimSpace(config.IssueID)
	config.Status = strings.TrimSpace(config.Status)
	config.Priority = strings.TrimSpace(config.Priority)
	config.AssignedTo = strings.TrimSpace(config.AssignedTo)
}

func (c *UpdateIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIssue) Execute(ctx core.ExecutionContext) error {
	config, err := decodeUpdateIssueConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	normalizeUpdateIssueConfiguration(&config)

	if err := validateUpdateIssueConfiguration(config); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	issue, err := client.UpdateIssue(config.IssueID, UpdateIssueRequest{
		Status:       config.Status,
		Priority:     config.Priority,
		AssignedTo:   config.AssignedTo,
		HasSeen:      config.HasSeen,
		IsPublic:     config.IsPublic,
		IsSubscribed: config.IsSubscribed,
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

func decodeUpdateIssueConfiguration(input any) (UpdateIssueConfiguration, error) {
	config := UpdateIssueConfiguration{}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &config,
		TagName:          "mapstructure",
		WeaklyTypedInput: true,
	})
	if err != nil {
		return UpdateIssueConfiguration{}, err
	}

	if err := decoder.Decode(input); err != nil {
		return UpdateIssueConfiguration{}, err
	}

	return config, nil
}
