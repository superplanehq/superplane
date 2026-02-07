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

type CreateIssueComment struct{}

type CreateIssueCommentConfiguration struct {
	Repository  string `mapstructure:"repository"`
	IssueNumber int    `mapstructure:"issueNumber"`
	Body        string `mapstructure:"body"`
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
	return `The Create Issue Comment component adds a comment to a GitHub issue or pull request.

## Use Cases

- **Deployment status updates**: Post deployment status or remediation updates to GitHub issues from SuperPlane
- **Runbook links**: Add runbook links, error details, or status for responders
- **Cross-platform sync**: Sync Slack or PagerDuty notes into GitHub as comments
- **Automated feedback**: Add automated test results, build status, or review comments

## Configuration

- **Repository**: Select the GitHub repository containing the issue/PR
- **Issue Number**: The issue or pull request number to comment on (supports expressions)
- **Body**: The comment text (Markdown supported, supports expressions)

## Output

Returns the created comment object with details including:
- Comment ID
- Body text
- Author information
- Created timestamp
- HTML URL to the comment`
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
			Name:     "issueNumber",
			Label:    "Issue Number",
			Type:     configuration.FieldTypeNumber,
			Required: true,
		},
		{
			Name:     "body",
			Label:    "Body",
			Type:     configuration.FieldTypeText,
			Required: true,
		},
	}
}

func (c *CreateIssueComment) Setup(ctx core.SetupContext) error {
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

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Create the comment
	comment, _, err := client.Issues.CreateComment(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		config.IssueNumber,
		&github.IssueComment{
			Body: &config.Body,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to create issue comment: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.issueComment",
		[]any{comment},
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
