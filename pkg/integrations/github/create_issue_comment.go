package github

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

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
	return `The Create Issue Comment component adds a comment to a GitHub issue or pull request.

## Use Cases

- **Deployment updates**: Post deployment status or remediation notes
- **Incident response**: Add runbook links, error details, or status updates
- **Cross-tool syncing**: Sync Slack or PagerDuty notes into GitHub

## Configuration

- **Repository**: Select the GitHub repository
- **Issue Number**: Issue or PR number to comment on (supports expressions)
- **Body**: Comment text (markdown supported, supports expressions)

## Output

Returns the created comment with details including:
- Comment ID
- Body
- Author
- Created timestamp
- Comment URL
`
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
			Label:       "Issue/PR Number",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., 42 or {{event.data.issue.number}}",
			Description: "Issue or pull request number",
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "Provide status update, runbook links, or context",
			Description: "Comment text (markdown supported)",
		},
	}
}

func (c *CreateIssueComment) Setup(ctx core.SetupContext) error {
	var config CreateIssueCommentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
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

	if strings.TrimSpace(config.IssueNumber) == "" {
		return fmt.Errorf("issue number is required")
	}

	if strings.TrimSpace(config.Body) == "" {
		return fmt.Errorf("comment body is required")
	}

	issueNumber, err := strconv.Atoi(config.IssueNumber)
	if err != nil {
		return fmt.Errorf("issue number is not a number: %v", err)
	}
	if issueNumber <= 0 {
		return fmt.Errorf("issue number is required")
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	commentRequest := &github.IssueComment{}

	if config.Body != "" {
		commentRequest.Body = &config.Body
	}

	comment, _, err := client.Issues.CreateComment(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		issueNumber,
		commentRequest,
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
