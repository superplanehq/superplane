package github

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/go-github/v74/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateIssueComment struct{}

type CreateIssueCommentConfiguration struct {
	Repository  string `json:"repository" mapstructure:"repository"`
	IssueNumber string `json:"issueNumber" mapstructure:"issueNumber"`
	Body        string `json:"body" mapstructure:"body"`
}

func (c *CreateIssueComment) Name() string {
	return "github.createIssueComment"
}

func (c *CreateIssueComment) Label() string {
	return "Create Issue Comment"
}

func (c *CreateIssueComment) Description() string {
	return "Add a comment to a GitHub issue or pull request"
}

func (c *CreateIssueComment) Documentation() string {
	return `The Create Issue Comment component adds a comment to an existing GitHub issue or pull request.
Issues and pull requests share the same comment API in GitHub.

## Use Cases

- **Deployment updates**: Post deployment status or remediation updates to GitHub issues
- **Runbook linking**: Add runbook links, error details, or status for responders
- **Cross-platform sync**: Sync Slack or PagerDuty notes into GitHub as comments
- **Automated comments**: Add automated comments based on workflow events

## Configuration

- **Repository**: Select the GitHub repository containing the issue
- **Issue Number**: The issue or PR number to comment on (supports expressions)
- **Body**: The comment text (supports Markdown and expressions)

## Output

Returns the created comment object including:
- Comment ID and URL
- Comment body
- Author information
- Created timestamp`
}

func (c *CreateIssueComment) Icon() string {
	return "github"
}

func (c *CreateIssueComment) Color() string {
	return "gray"
}

func (c *CreateIssueComment) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateIssueComment) Configuration() []configuration.Field {
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
			Description: "The issue or pull request number to comment on",
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The comment text. Supports Markdown formatting.",
		},
	}
}

func (c *CreateIssueComment) Setup(ctx core.SetupContext) error {
	var config CreateIssueCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository == "" {
		return errors.New("repository is required")
	}

	if config.IssueNumber == "" {
		return errors.New("issue number is required")
	}

	if config.Body == "" {
		return errors.New("body is required")
	}

	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *CreateIssueComment) Execute(ctx core.ExecutionContext) error {
	var config CreateIssueCommentConfiguration
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

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	comment := &github.IssueComment{
		Body: &config.Body,
	}

	createdComment, _, err := client.Issues.CreateComment(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		issueNumber,
		comment,
	)

	if err != nil {
		return fmt.Errorf("failed to create issue comment: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.issueComment",
		[]any{createdComment},
	)
}

func (c *CreateIssueComment) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateIssueComment) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *CreateIssueComment) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateIssueComment) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateIssueComment) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateIssueComment) Cleanup(ctx core.SetupContext) error {
	return nil
}
