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

type GetIssue struct{}

type GetIssueConfiguration struct {
	Repository  string `json:"repository" mapstructure:"repository"`
	IssueNumber string `json:"issueNumber" mapstructure:"issueNumber"`
}

func (c *GetIssue) Name() string {
	return "github.getIssue"
}

func (c *GetIssue) Label() string {
	return "Get Issue"
}

func (c *GetIssue) Description() string {
	return "Get a GitHub issue by number"
}

func (c *GetIssue) Documentation() string {
	return `The Get Issue component retrieves a specific issue from a GitHub repository by its issue number.

## Use Cases

- **Issue lookup**: Fetch issue details for processing or display
- **Workflow automation**: Get issue information to make decisions in workflows
- **Data enrichment**: Retrieve issue data to combine with other information
- **Status checking**: Check issue status before performing actions

## Configuration

- **Repository**: Select the GitHub repository containing the issue
- **Issue Number**: The issue number to retrieve (supports expressions)

## Output

Returns the complete issue object including:
- Issue number, title, and body
- State (open/closed)
- Labels and assignees
- Created and updated timestamps
- Author information
- Comments count and other metadata`
}

func (c *GetIssue) Icon() string {
	return "github"
}

func (c *GetIssue) Color() string {
	return "gray"
}

func (c *GetIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetIssue) Configuration() []configuration.Field {
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
			Name:     "issueNumber",
			Label:    "Issue Number",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
	}
}

func (c *GetIssue) Setup(ctx core.SetupContext) error {
	var config GetIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.IssueNumber == "" {
		return errors.New("issue number is required")
	}

	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *GetIssue) Execute(ctx core.ExecutionContext) error {
	var config GetIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	issueNumber, err := strconv.Atoi(config.IssueNumber)
	if err != nil {
		return fmt.Errorf("issue number is not a number: %v", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	// Initialize GitHub client
	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Get the issue
	issue, _, err := client.Issues.Get(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		issueNumber,
	)

	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.issue",
		[]any{issue},
	)
}

func (c *GetIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *GetIssue) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetIssue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}
