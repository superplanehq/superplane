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

type RemoveIssueLabel struct{}

type RemoveIssueLabelConfiguration struct {
	Repository     string   `json:"repository" mapstructure:"repository"`
	IssueNumber    string   `json:"issueNumber" mapstructure:"issueNumber"`
	Labels         []string `json:"labels" mapstructure:"labels"`
	FailIfNotFound bool     `json:"failIfNotFound" mapstructure:"failIfNotFound"`
}

func (c *RemoveIssueLabel) Name() string {
	return "github.removeIssueLabel"
}

func (c *RemoveIssueLabel) Label() string {
	return "Remove Issue Label"
}

func (c *RemoveIssueLabel) Description() string {
	return "Remove labels from a GitHub issue"
}

func (c *RemoveIssueLabel) Documentation() string {
	return `The Remove Issue Label component removes one or more labels from an existing GitHub issue without affecting other labels.

## Use Cases

- **Triage cleanup**: Remove temporary triage labels after processing
- **Status transitions**: Remove old status labels when issues move forward
- **Automated cleanup**: Strip labels that no longer apply based on workflow events

## Configuration

- **Repository**: Select the GitHub repository containing the issue
- **Issue Number**: The issue number to remove labels from (supports expressions)
- **Labels**: List of label names to remove from the issue

## Output

Returns the remaining list of labels on the issue after the removal.`
}

func (c *RemoveIssueLabel) Icon() string {
	return "github"
}

func (c *RemoveIssueLabel) Color() string {
	return "gray"
}

func (c *RemoveIssueLabel) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RemoveIssueLabel) Configuration() []configuration.Field {
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
			Description: "The issue number to remove labels from",
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
		{
			Name:        "failIfNotFound",
			Label:       "Fail if not found",
			Type:        configuration.FieldTypeBool,
			Description: "Fail the execution if a label is not present on the issue",
			Default:     false,
		},
	}
}

func (c *RemoveIssueLabel) Setup(ctx core.SetupContext) error {
	var config RemoveIssueLabelConfiguration
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

func (c *RemoveIssueLabel) Execute(ctx core.ExecutionContext) error {
	var config RemoveIssueLabelConfiguration
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

	for _, label := range config.Labels {
		_, err = client.Issues.RemoveLabelForIssue(
			context.Background(),
			appMetadata.Owner,
			config.Repository,
			issueNumber,
			label,
		)
		if err != nil {
			if !config.FailIfNotFound {
				continue
			}

			return fmt.Errorf("failed to remove label %s from issue: %w", label, err)
		}
	}

	remainingLabels, _, err := client.Issues.ListLabelsByIssue(
		context.Background(),
		appMetadata.Owner,
		config.Repository,
		issueNumber,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to list remaining labels: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.labels",
		[]any{remainingLabels},
	)
}

func (c *RemoveIssueLabel) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RemoveIssueLabel) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *RemoveIssueLabel) Actions() []core.Action {
	return []core.Action{}
}

func (c *RemoveIssueLabel) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *RemoveIssueLabel) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RemoveIssueLabel) Cleanup(ctx core.SetupContext) error {
	return nil
}
