package github

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type AddIssueAssignee struct{}

type AddIssueAssigneeConfiguration struct {
	Repository  string   `json:"repository" mapstructure:"repository"`
	IssueNumber string   `json:"issueNumber" mapstructure:"issueNumber"`
	Assignees   []string `json:"assignees" mapstructure:"assignees"`
}

func (c *AddIssueAssignee) Name() string {
	return "github.addIssueAssignee"
}

func (c *AddIssueAssignee) Label() string {
	return "Add Issue Assignee"
}

func (c *AddIssueAssignee) Description() string {
	return "Add assignees to a GitHub issue"
}

func (c *AddIssueAssignee) Documentation() string {
	return `The Add Issue Assignee component adds one or more assignees to an existing GitHub issue without affecting existing assignees.

## Use Cases

- **Auto-assignment**: Automatically assign issues to team members based on workflow triggers
- **Escalation**: Add additional assignees when issues require attention from specific people
- **On-call routing**: Assign issues to the current on-call engineer

## Configuration

- **Repository**: Select the GitHub repository containing the issue
- **Issue Number**: The issue number to add assignees to (supports expressions)
- **Assignees**: List of GitHub usernames to assign to the issue

## Output

Returns the updated issue object with all current information.`
}

func (c *AddIssueAssignee) Icon() string {
	return "github"
}

func (c *AddIssueAssignee) Color() string {
	return "gray"
}

func (c *AddIssueAssignee) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddIssueAssignee) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "issueNumber",
			Label:       "Issue Number",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The issue number to add assignees to",
		},
		{
			Name:        "assignees",
			Label:       "Assignees",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "GitHub usernames to assign (e.g. octocat)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Assignee",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
	}
}

func (c *AddIssueAssignee) Setup(ctx core.SetupContext) error {
	var config AddIssueAssigneeConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository == "" {
		return errors.New("repository is required")
	}

	if config.IssueNumber == "" {
		return errors.New("issue number is required")
	}

	if len(config.Assignees) == 0 {
		return errors.New("at least one assignee is required")
	}

	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *AddIssueAssignee) Execute(ctx core.ExecutionContext) error {
	var config AddIssueAssigneeConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	issueNumber, err := strconv.Atoi(config.IssueNumber)
	if err != nil {
		return fmt.Errorf("issue number is not a number: %v", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	issue, _, err := client.Issues.AddAssignees(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		issueNumber,
		sanitizeAssignees(config.Assignees),
	)
	if err != nil {
		return fmt.Errorf("failed to add assignees to issue: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.issue",
		[]any{issue},
	)
}

func (c *AddIssueAssignee) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AddIssueAssignee) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *AddIssueAssignee) Actions() []core.Action {
	return []core.Action{}
}

func (c *AddIssueAssignee) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *AddIssueAssignee) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddIssueAssignee) Cleanup(ctx core.SetupContext) error {
	return nil
}
