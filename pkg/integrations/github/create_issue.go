package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateIssue struct{}

type CreateIssueConfiguration struct {
	Repository string   `mapstructure:"repository"`
	Title      string   `mapstructure:"title"`
	Body       string   `mapstructure:"body"`
	Assignees  []string `mapstructure:"assignees"`
	Labels     []string `mapstructure:"labels"`
}

func (c *CreateIssue) Name() string {
	return "github.createIssue"
}

func (c *CreateIssue) Label() string {
	return "Create Issue"
}

func (c *CreateIssue) Description() string {
	return "Create a new issue in a GitHub repository"
}

func (c *CreateIssue) Documentation() string {
	return `The Create Issue component creates a new issue in a specified GitHub repository.

## Use Cases

- **Automated bug reporting**: Create issues automatically when errors are detected
- **Task creation**: Generate issues from external systems or workflows
- **Notification tracking**: Convert notifications into trackable issues
- **Workflow automation**: Create issues as part of automated processes

## Configuration

- **Repository**: Select the GitHub repository where the issue will be created
- **Title**: The issue title (supports expressions)
- **Body**: The issue body/description (supports markdown and expressions)
- **Assignees**: Optional list of GitHub usernames to assign the issue to
- **Labels**: Optional list of labels to apply to the issue

## Output

Returns the created issue object with details including:
- Issue number
- URL
- State
- Created timestamp
- All issue metadata`
}

func (c *CreateIssue) Icon() string {
	return "github"
}

func (c *CreateIssue) Color() string {
	return "gray"
}

func (c *CreateIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeAppInstallationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:     "title",
			Label:    "Title",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "body",
			Label:    "Body",
			Type:     configuration.FieldTypeText,
			Required: false,
		},
		{
			Name:     "assignees",
			Label:    "Assignees",
			Type:     configuration.FieldTypeList,
			Required: false,
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
			Name:     "labels",
			Label:    "Labels",
			Type:     configuration.FieldTypeList,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
	}
}

func (c *CreateIssue) Setup(ctx core.SetupContext) error {
	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.AppInstallation,
		ctx.Configuration,
	)
}

func (c *CreateIssue) Execute(ctx core.ExecutionContext) error {
	var config CreateIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.AppInstallation.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	client, err := NewClient(ctx.AppInstallation, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	//
	// Prepare the request based on the configuration
	//
	issueRequest := &github.IssueRequest{
		Title: &config.Title,
	}

	if config.Body != "" {
		issueRequest.Body = &config.Body
	}

	if len(config.Assignees) > 0 {
		issueRequest.Assignees = &config.Assignees
	}

	if len(config.Labels) > 0 {
		issueRequest.Labels = &config.Labels
	}

	// Create the issue
	issue, _, err := client.Issues.Create(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		issueRequest,
	)

	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.issue",
		[]any{issue},
	)
}

func (c *CreateIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *CreateIssue) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateIssue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}
