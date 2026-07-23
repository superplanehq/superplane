package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_add_issue_label.json
var exampleOutputAddIssueLabel []byte

type AddIssueLabel struct{}

type AddIssueLabelConfiguration struct {
	Project  string   `mapstructure:"project"`
	IssueIID string   `mapstructure:"issueIid"`
	Labels   []string `mapstructure:"labels"`
}

func (c *AddIssueLabel) Name() string {
	return "gitlab.addIssueLabel"
}

func (c *AddIssueLabel) Label() string {
	return "Add Issue Label"
}

func (c *AddIssueLabel) Description() string {
	return "Add labels to a GitLab issue"
}

func (c *AddIssueLabel) Documentation() string {
	return `The Add Issue Label component adds one or more labels to an existing GitLab issue without affecting existing labels.

## Use Cases

- **Triage automation**: Automatically label issues based on content or source
- **Status tracking**: Add status labels as issues move through workflows
- **Priority tagging**: Apply priority labels based on external signals

## Configuration

- **Project** (required): The GitLab project containing the issue
- **Issue IID** (required): The internal ID (IID) of the issue to add labels to (supports expressions)
- **Labels** (required): List of label names to add to the issue. If a label does not already exist, GitLab creates it.

## Output

Returns the full list of labels currently on the issue after the addition.`
}

func (c *AddIssueLabel) Icon() string {
	return "gitlab"
}

func (c *AddIssueLabel) Color() string {
	return "orange"
}

func (c *AddIssueLabel) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddIssueLabel) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputAddIssueLabel, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *AddIssueLabel) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
		{
			Name:        "issueIid",
			Label:       "Issue IID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The internal ID (IID) of the issue to add labels to",
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "Labels to add to the issue's existing labels",
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

	if config.Project == "" {
		return errors.New("project is required")
	}

	if config.IssueIID == "" {
		return errors.New("issue IID is required")
	}

	if len(config.Labels) == 0 {
		return errors.New("at least one label is required")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *AddIssueLabel) Execute(ctx core.ExecutionContext) error {
	var config AddIssueLabelConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if len(config.Labels) == 0 {
		return errors.New("at least one label is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	addLabels := strings.Join(config.Labels, ",")
	issue, err := client.UpdateIssue(context.Background(), config.Project, config.IssueIID, &UpdateIssueRequest{
		AddLabels: &addLabels,
	})
	if err != nil {
		return fmt.Errorf("failed to add labels to issue: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.labels",
		[]any{issue.Labels},
	)
}

func (c *AddIssueLabel) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AddIssueLabel) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *AddIssueLabel) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddIssueLabel) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AddIssueLabel) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AddIssueLabel) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
