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

type AddIssueLabel struct{}

type AddIssueLabelConfiguration struct {
	Repository  string   `json:"repository" mapstructure:"repository"`
	IssueNumber string   `json:"issueNumber" mapstructure:"issueNumber"`
	Labels      []string `json:"labels" mapstructure:"labels"`
}

func (c *AddIssueLabel) Name() string {
	return "github.addIssueLabel"
}

func (c *AddIssueLabel) Label() string {
	return "Add Issue Label"
}

func (c *AddIssueLabel) Description() string {
	return "Add labels to a GitHub issue"
}

func (c *AddIssueLabel) Documentation() string {
	return `The Add Issue Label component adds one or more labels to an existing GitHub issue without affecting existing labels.

## Use Cases

- **Triage automation**: Automatically label issues based on content or source
- **Status tracking**: Add status labels as issues move through workflows
- **Priority tagging**: Apply priority labels based on external signals

## Configuration

- **Repository**: Select the GitHub repository containing the issue
- **Issue Number**: The issue number to add labels to (supports expressions)
- **Labels**: List of label names to add to the issue

## Output

Returns the full list of labels currently on the issue after the addition.`
}

func (c *AddIssueLabel) Icon() string {
	return "github"
}

func (c *AddIssueLabel) Color() string {
	return "gray"
}

func (c *AddIssueLabel) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddIssueLabel) Configuration() []configuration.Field {
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
			Description: "The issue number to add labels to",
		},
		{
			Name:     "labels",
			Label:    "Labels",
			Type:     configuration.FieldTypeList,
			Required: true,
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

func (c *AddIssueLabel) Setup(ctx core.SetupContext) error {
	var config AddIssueLabelConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Repository == "" {
		return errors.New("repository is required")
	}

	if config.IssueNumber == "" {
		return errors.New("issue number is required")
	}

	if len(config.Labels) == 0 {
		return errors.New("at least one label is required")
	}

	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *AddIssueLabel) Execute(ctx core.ExecutionContext) error {
	var config AddIssueLabelConfiguration
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

	labels, _, err := client.Issues.AddLabelsToIssue(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		issueNumber,
		config.Labels,
	)
	if err != nil {
		return fmt.Errorf("failed to add labels to issue: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.labels",
		[]any{labels},
	)
}

func (c *AddIssueLabel) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AddIssueLabel) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *AddIssueLabel) Actions() []core.Action {
	return []core.Action{}
}

func (c *AddIssueLabel) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *AddIssueLabel) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddIssueLabel) Cleanup(ctx core.SetupContext) error {
	return nil
}
