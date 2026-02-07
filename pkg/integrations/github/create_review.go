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

type CreateReview struct{}

// ReviewComment represents a single comment to be added as part of the review
type ReviewComment struct {
	Path      string `json:"path" mapstructure:"path"`
	Body      string `json:"body" mapstructure:"body"`
	Line      int    `json:"line" mapstructure:"line"`
	Side      string `json:"side" mapstructure:"side"`
	StartLine int    `json:"startLine" mapstructure:"startLine"`
	StartSide string `json:"startSide" mapstructure:"startSide"`
}

type CreateReviewConfiguration struct {
	Repository       string          `json:"repository" mapstructure:"repository"`
	PullRequestNumber int             `json:"pullRequestNumber" mapstructure:"pullRequestNumber"`
	Event            string          `json:"event" mapstructure:"event"`
	Body             string          `json:"body" mapstructure:"body"`
	CommitID         string          `json:"commitId" mapstructure:"commitId"`
	Comments         []ReviewComment `json:"comments" mapstructure:"comments"`
}

func (c *CreateReview) Name() string {
	return "github.createReview"
}

func (c *CreateReview) Label() string {
	return "Create Review"
}

func (c *CreateReview) Description() string {
	return "Create a review on a GitHub pull request"
}

func (c *CreateReview) Documentation() string {
	return `The Create Review component creates a review on a GitHub pull request.

## Use Cases

- **Automated code review**: Post automated review comments on pull requests
- **CI/CD feedback**: Approve or request changes based on CI results
- **Quality gates**: Programmatically review PRs based on code analysis
- **Bot reviews**: Add bot-generated feedback with inline comments

## Configuration

- **Repository**: Select the GitHub repository containing the pull request
- **Pull Request Number**: The number of the pull request to review
- **Event**: The review action (APPROVE, REQUEST_CHANGES, COMMENT, or PENDING)
- **Body**: The review body/summary (supports markdown)
- **Commit ID**: Optional specific commit SHA to review (defaults to latest)
- **Comments**: Optional list of inline review comments with file path, line number, and body

## Review Events

- **APPROVE**: Approve the pull request
- **REQUEST_CHANGES**: Request changes before the PR can be merged
- **COMMENT**: Leave a neutral review comment
- **PENDING**: Create a pending review (can be submitted later)

## Output

Returns the created review object with details including:
- Review ID
- State
- URL
- Submitted timestamp
- Review body`
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
			Name:     "pullRequestNumber",
			Label:    "Pull Request Number",
			Type:     configuration.FieldTypeNumber,
			Required: true,
		},
		{
			Name:     "event",
			Label:    "Event",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Description: "The review action to perform",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Approve", Value: "APPROVE"},
						{Label: "Request Changes", Value: "REQUEST_CHANGES"},
						{Label: "Comment", Value: "COMMENT"},
						{Label: "Pending", Value: "PENDING"},
					},
				},
			},
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "The review summary/comment (supports markdown)",
		},
		{
			Name:        "commitId",
			Label:       "Commit ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional SHA of the commit to review. Defaults to the latest commit.",
		},
		{
			Name:        "comments",
			Label:       "Comments",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Inline review comments to add to specific lines of code",
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
								Description: "The relative path of the file to comment on",
							},
							{
								Name:        "body",
								Label:       "Comment Body",
								Type:        configuration.FieldTypeText,
								Required:    true,
								Description: "The text of the review comment",
							},
							{
								Name:        "line",
								Label:       "Line",
								Type:        configuration.FieldTypeNumber,
								Required:    true,
								Description: "The line number to comment on",
							},
							{
								Name:     "side",
								Label:    "Side",
								Type:     configuration.FieldTypeSelect,
								Required: false,
								Default:  "RIGHT",
								Description: "Which side of the diff to comment on",
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Right (new code)", Value: "RIGHT"},
											{Label: "Left (old code)", Value: "LEFT"},
										},
									},
								},
							},
							{
								Name:        "startLine",
								Label:       "Start Line",
								Type:        configuration.FieldTypeNumber,
								Required:    false,
								Description: "For multi-line comments, the first line of the range",
							},
							{
								Name:     "startSide",
								Label:    "Start Side",
								Type:     configuration.FieldTypeSelect,
								Required: false,
								Description: "For multi-line comments, which side the start line is on",
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Right (new code)", Value: "RIGHT"},
											{Label: "Left (old code)", Value: "LEFT"},
										},
									},
								},
							},
						},
					},
				},
			},
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

	// Validate that body is provided for events that require it
	if (config.Event == "REQUEST_CHANGES" || config.Event == "COMMENT") && config.Body == "" {
		return fmt.Errorf("body is required when event is REQUEST_CHANGES or COMMENT")
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	// Build the review request
	reviewRequest := &github.PullRequestReviewRequest{
		Event: &config.Event,
	}

	// Set body if provided
	if config.Body != "" {
		reviewRequest.Body = &config.Body
	}

	// Set commit ID if provided
	if config.CommitID != "" {
		reviewRequest.CommitID = &config.CommitID
	}

	// Convert comments if provided
	if len(config.Comments) > 0 {
		draftComments := make([]*github.DraftReviewComment, 0, len(config.Comments))
		for _, comment := range config.Comments {
			draftComment := &github.DraftReviewComment{
				Path: &comment.Path,
				Body: &comment.Body,
				Line: &comment.Line,
			}

			// Set side (default to RIGHT if not specified)
			side := comment.Side
			if side == "" {
				side = "RIGHT"
			}
			draftComment.Side = &side

			// Set multi-line comment fields if provided
			if comment.StartLine > 0 {
				draftComment.StartLine = &comment.StartLine
				if comment.StartSide != "" {
					draftComment.StartSide = &comment.StartSide
				}
			}

			draftComments = append(draftComments, draftComment)
		}
		reviewRequest.Comments = draftComments
	}

	// Create the review
	createdReview, _, err := client.PullRequests.CreateReview(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		config.PullRequestNumber,
		reviewRequest,
	)

	if err != nil {
		return fmt.Errorf("failed to create pull request review: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.pullRequestReview",
		[]any{createdReview},
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
