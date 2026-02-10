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

type CreateReview struct{}

type CreateReviewConfiguration struct {
	Repository string  `mapstructure:"repository" json:"repository"`
	PullNumber string  `mapstructure:"pullNumber" json:"pullNumber"`
	Event      string  `mapstructure:"event" json:"event"`
	Body       *string `mapstructure:"body,omitempty" json:"body,omitempty"`
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
	return `The Create Review component submits a pull request review (approve, request changes, or comment) on a GitHub pull request.

## Use Cases

- **Automation**: Auto-approve when checks pass
- **Quality gates**: Request changes when checks fail
- **Bots**: Post review feedback

## Configuration

- **Repository**: Select the GitHub repository
- **Pull Number**: Pull request number
- **Event**: APPROVE, REQUEST_CHANGES, or COMMENT
- **Body**: Optional review body (required for REQUEST_CHANGES and COMMENT)

## Output

Emits the submitted review object including:
- id, state, submitted_at
- body and user`
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
			Name:     "pullNumber",
			Label:    "Pull Number",
			Type:     configuration.FieldTypeString,
			Required: true,
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
						{Label: "Request changes", Value: "REQUEST_CHANGES"},
						{Label: "Comment", Value: "COMMENT"},
					},
				},
			},
		},
		{
			Name:        "body",
			Label:       "Body",
			Type:        configuration.FieldTypeText,
			Description: "Review body (required for REQUEST_CHANGES and COMMENT).",
			RequiredConditions: []configuration.RequiredCondition{
				{
					Field:  "event",
					Values: []string{"REQUEST_CHANGES", "COMMENT"},
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

	if config.Repository == "" {
		return errors.New("repository is required")
	}
	if config.PullNumber == "" {
		return errors.New("pull number is required")
	}

	pullNumber, err := strconv.Atoi(config.PullNumber)
	if err != nil {
		return fmt.Errorf("pull number is not a number: %v", err)
	}

	event := strings.ToUpper(strings.TrimSpace(config.Event))
	if event != "APPROVE" && event != "REQUEST_CHANGES" && event != "COMMENT" {
		return fmt.Errorf("invalid event: %s", config.Event)
	}

	if (event == "REQUEST_CHANGES" || event == "COMMENT") && (config.Body == nil || strings.TrimSpace(*config.Body) == "") {
		return fmt.Errorf("body is required for %s", event)
	}

	var appMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	req := &github.PullRequestReviewRequest{
		Event: github.String(event),
	}
	if config.Body != nil && strings.TrimSpace(*config.Body) != "" {
		req.Body = config.Body
	}

	review, _, err := client.PullRequests.CreateReview(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		pullNumber,
		req,
	)
	if err != nil {
		return fmt.Errorf("failed to create review: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.pullRequestReview",
		[]any{review},
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
