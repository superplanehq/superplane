package github

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type RemoveIssueAssignee struct{}

type RemoveIssueAssigneeConfiguration struct {
	Repository     string   `json:"repository" mapstructure:"repository"`
	IssueNumber    string   `json:"issueNumber" mapstructure:"issueNumber"`
	Assignees      []string `json:"assignees" mapstructure:"assignees"`
	FailIfNotFound bool     `json:"failIfNotFound" mapstructure:"failIfNotFound"`
}

func (c *RemoveIssueAssignee) Name() string {
	return "github.removeIssueAssignee"
}

func (c *RemoveIssueAssignee) Label() string {
	return "Remove Issue Assignee"
}

func (c *RemoveIssueAssignee) Description() string {
	return "Remove assignees from a GitHub issue"
}

func (c *RemoveIssueAssignee) Documentation() string {
	return `The Remove Issue Assignee component removes one or more assignees from an existing GitHub issue without affecting other assignees.

## Use Cases

- **De-escalation**: Remove assignees when issues are resolved or transferred
- **Rotation**: Remove previous on-call assignees when rotating responsibilities
- **Cleanup**: Remove assignees who are no longer involved with an issue

## Configuration

- **Repository**: Select the GitHub repository containing the issue
- **Issue Number**: The issue number to remove assignees from (supports expressions)
- **Assignees**: List of GitHub usernames to remove from the issue

## Output

Returns the updated issue object with all current information.`
}

func (c *RemoveIssueAssignee) Icon() string {
	return "github"
}

func (c *RemoveIssueAssignee) Color() string {
	return "gray"
}

func (c *RemoveIssueAssignee) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RemoveIssueAssignee) Configuration() []configuration.Field {
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
			Description: "The issue number to remove assignees from",
		},
		{
			Name:        "assignees",
			Label:       "Assignees",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "GitHub usernames to remove (e.g. octocat)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Assignee",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "failIfNotFound",
			Label:       "Fail if not found",
			Type:        configuration.FieldTypeBool,
			Description: "Fail the execution if an assignee is not present on the issue",
			Default:     false,
		},
	}
}

func (c *RemoveIssueAssignee) Setup(ctx core.SetupContext) error {
	var config RemoveIssueAssigneeConfiguration
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

func (c *RemoveIssueAssignee) Execute(ctx core.ExecutionContext) error {
	var config RemoveIssueAssigneeConfiguration
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

	assignees := sanitizeAssignees(config.Assignees)

	issue, _, err := client.Issues.RemoveAssignees(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		issueNumber,
		assignees,
	)
	if err != nil {
		return fmt.Errorf("failed to remove assignees from issue: %w", err)
	}

	if config.FailIfNotFound {
		for _, requested := range assignees {
			for _, a := range issue.Assignees {
				if strings.EqualFold(a.GetLogin(), requested) {
					return fmt.Errorf("failed to remove assignee %s: user not found on issue", requested)
				}
			}
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.issue",
		[]any{issue},
	)
}

func (c *RemoveIssueAssignee) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RemoveIssueAssignee) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *RemoveIssueAssignee) Actions() []core.Action {
	return []core.Action{}
}

func (c *RemoveIssueAssignee) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *RemoveIssueAssignee) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RemoveIssueAssignee) Cleanup(ctx core.SetupContext) error {
	return nil
}
