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

type CreateReview struct{}

type CreateReviewConfiguration struct {
	Repository string                `mapstructure:"repository"`
	PullNumber string                `mapstructure:"pullNumber"`
	Event      string                `mapstructure:"event"`
	Body       *string               `mapstructure:"body,omitempty"`
	CommitID   *string               `mapstructure:"commitId,omitempty"`
	Comments   []ReviewCommentConfig `mapstructure:"comments,omitempty"`
}

type ReviewCommentConfig struct {
	Path string `mapstructure:"path"`
	Line int    `mapstructure:"line"`
	Body string `mapstructure:"body"`
}

func (c *CreateReview) Name() string {
	return "github.createReview"
}

func (c *CreateReview) Label() string {
	return "Create Review"
}

func (c *CreateReview) Description() string {
	return "Submit a pull request review on GitHub"
}

func (c *CreateReview) Documentation() string {
	return `The Create Review component submits a pull request review (approve, request changes, or comment) on a GitHub PR.

## Use Cases

- **Automated approvals**: Submit automated approval when CI passes from SuperPlane
- **Request changes**: Automatically request changes when code quality checks fail
- **Post review comments**: Add review comments (inline or general) from automation or bots
- **Sync review state**: Sync review state with external systems (Jira, Slack)

## Configuration

- **Repository**: Select the GitHub repository containing the PR
- **Pull Number**: The PR number to review (supports expressions)
- **Event**: Review action - APPROVE, REQUEST_CHANGES, or COMMENT
- **Body**: Review body text (optional; required for REQUEST_CHANGES with feedback)
- **Commit ID**: Specific commit SHA to review (optional)
- **Comments**: Inline review comments with path, line, and body (optional)

## Output

Returns the created review object with details including:
- Review ID
- State (APPROVED, CHANGES_REQUESTED, COMMENTED)
- Submitted timestamp
- HTML URL to the review

## Notes

- APPROVE and REQUEST_CHANGES events require appropriate permissions
- Inline comments require the file path and line number
- Use expressions to dynamically set the PR number from upstream data`
}

func (c *CreateReview) Icon() string {
	return "github"
}

func (c *CreateReview) Color() string {
	return "gray"
}

func (c *CreateReview) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateReview) Configuration() []configuration.Field {
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
			Name:        "pullNumber",
			Label:       "Pull Request Number",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., 42 or {{$.data.number}}",
			Description: "The pull request number to review",
		},
		{
			Name:     "event",
			Label:    "Event",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Approve", Value: "APPROVE"},
						{Label: "Request Changes", Value: "REQUEST_CHANGES"},
						{Label: "Comment", Value: "COMMENT"},
					},
				},
			},
			Description: "The review action to perform",
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Placeholder: "Enter review body text (optional for APPROVE)",
			Description: "Review body text (required for REQUEST_CHANGES)",
		},
		{
			Name:        "commitId",
			Label:       "Commit ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., abc123...",
			Description: "Specific commit SHA to review (optional)",
		},
		{
			Name:     "comments",
			Label:    "Inline Comments",
			Type:     configuration.FieldTypeList,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Comment",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "path",
								Label:       "File Path",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Placeholder: "e.g., src/main.go",
								Description: "Relative path to the file",
							},
							{
								Name:        "line",
								Label:       "Line",
								Type:        configuration.FieldTypeNumber,
								Required:    true,
								Placeholder: "e.g., 42",
								Description: "Line number in the diff",
							},
							{
								Name:        "body",
								Label:       "Comment",
								Type:        configuration.FieldTypeText,
								Required:    true,
								Placeholder: "Enter comment text",
								Description: "The comment text",
							},
						},
					},
				},
			},
			Description: "Inline review comments (optional)",
		},
	}
}

func (c *CreateReview) Setup(ctx core.SetupContext) error {
	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *CreateReview) Execute(ctx core.ExecutionContext) error {
	var config CreateReviewConfiguration
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

	pullNumber, err := strconv.Atoi(config.PullNumber)
	if err != nil {
		return fmt.Errorf("invalid pull request number %q: %w", config.PullNumber, err)
	}

	// Build the review request
	reviewRequest := &github.PullRequestReviewRequest{
		Event: &config.Event,
	}

	if config.Body != nil && *config.Body != "" {
		reviewRequest.Body = config.Body
	}

	if config.CommitID != nil && *config.CommitID != "" {
		reviewRequest.CommitID = config.CommitID
	}

	// Add inline comments if provided
	if len(config.Comments) > 0 {
		comments := make([]*github.DraftReviewComment, 0, len(config.Comments))
		for _, c := range config.Comments {
			comment := &github.DraftReviewComment{
				Path: &c.Path,
				Line: &c.Line,
				Body: &c.Body,
			}
			comments = append(comments, comment)
		}
		reviewRequest.Comments = comments
	}

	// Create the review
	review, _, err := client.PullRequests.CreateReview(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		pullNumber,
		reviewRequest,
	)
	if err != nil {
		return fmt.Errorf("failed to create review: %w", err)
	}

	// Build output data
	reviewData := map[string]any{
		"id":           review.GetID(),
		"node_id":      review.GetNodeID(),
		"state":        review.GetState(),
		"body":         review.GetBody(),
		"html_url":     review.GetHTMLURL(),
		"pull_request": review.GetPullRequestURL(),
		"submitted_at": review.GetSubmittedAt().Format("2006-01-02T15:04:05Z"),
	}

	if review.User != nil {
		reviewData["user"] = map[string]any{
			"login":      review.User.GetLogin(),
			"id":         review.User.GetID(),
			"avatar_url": review.User.GetAvatarURL(),
			"html_url":   review.User.GetHTMLURL(),
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.pullRequestReview",
		[]any{reviewData},
	)
}

func (c *CreateReview) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateReview) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *CreateReview) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateReview) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateReview) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateReview) Cleanup(ctx core.SetupContext) error {
	return nil
}
