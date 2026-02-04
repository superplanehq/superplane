package github

import (
	"context"
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
	Repository  string `mapstructure:"repository"`
	IssueNumber string `mapstructure:"issueNumber"`
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
	return `The Create Issue Comment component adds a comment to an existing GitHub issue or pull request.

## Use Cases

- **Deployment notifications**: Post deployment status or remediation updates to GitHub issues
- **Runbook integration**: Add runbook links, error details, or status information for responders
- **Cross-platform sync**: Sync Slack or PagerDuty notes into GitHub as comments
- **Automated updates**: Post automated status updates from CI/CD pipelines

## Configuration

- **Repository**: Select the GitHub repository containing the issue
- **Issue Number**: The issue or PR number to comment on (supports expressions)
- **Body**: The comment text (supports Markdown and expressions)

## Output

Returns the created comment object with details including:
- Comment ID
- Body text
- Author information
- Created timestamp
- HTML URL to the comment

## Notes

- The same API works for both issues and pull requests
- Comments support full GitHub Markdown formatting
- Use expressions to dynamically set the issue number from upstream data`
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
			Placeholder: "e.g., 42 or {{$.data.number}}",
			Description: "The issue or PR number to comment on",
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "Enter your comment text (Markdown supported)",
			Description: "The comment text to post",
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
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	issueNumber, err := strconv.Atoi(config.IssueNumber)
	if err != nil {
		return fmt.Errorf("invalid issue number %q: %w", config.IssueNumber, err)
	}

	// Create the comment
	comment, _, err := client.Issues.CreateComment(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		issueNumber,
		&github.IssueComment{
			Body: &config.Body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	// Build output data
	commentData := map[string]any{
		"id":         comment.GetID(),
		"node_id":    comment.GetNodeID(),
		"body":       comment.GetBody(),
		"html_url":   comment.GetHTMLURL(),
		"issue_url":  comment.GetIssueURL(),
		"created_at": comment.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
		"updated_at": comment.GetUpdatedAt().Format("2006-01-02T15:04:05Z"),
	}

	if comment.User != nil {
		commentData["user"] = map[string]any{
			"login":      comment.User.GetLogin(),
			"id":         comment.User.GetID(),
			"avatar_url": comment.User.GetAvatarURL(),
			"html_url":   comment.User.GetHTMLURL(),
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.issueComment",
		[]any{commentData},
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
