package gitlab

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateIssue struct{}

type CreateIssueConfiguration struct {
	Project   string   `mapstructure:"project"`
	Title     string   `mapstructure:"title"`
	Body      string   `mapstructure:"body"`
	Assignees []int    `mapstructure:"assignees"`
	Labels    []string `mapstructure:"labels"`
}

func (c *CreateIssue) Name() string {
	return "gitlab.createIssue"
}

func (c *CreateIssue) Label() string {
	return "Create Issue"
}

func (c *CreateIssue) Description() string {
	return "Create a new issue in a GitLab project"
}

func (c *CreateIssue) Documentation() string {
	return `The Create Issue component creates a new issue in a specified GitLab project.`
}

func (c *CreateIssue) Icon() string {
	return "gitlab"
}

func (c *CreateIssue) Color() string {
	return "orange"
}

func (c *CreateIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateIssue) ExampleOutput() map[string]any {
	return map[string]any{}
}

func (c *CreateIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository", // Using repository type to match fetched resource
					UseNameAsValue: false,        // We need ID for API calls
				},
			},
		},
		{
			Name:     "title",
			Label:    "Title",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "body",
			Label:    "Description",
			Type:     configuration.FieldTypeText,
			Required: false,
		},
		{
			Name:     "assignees",
			Label:    "Assignee IDs",
			Type:     configuration.FieldTypeList,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Assignee ID",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeNumber,
					},
				},
			},
		},
		{
			Name:     "labels",
			Label:    "Labels",
			Type:     configuration.FieldTypeList,
			Required: false,
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

func (c *CreateIssue) Setup(ctx core.SetupContext) error {
	return ensureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.Configuration,
	)
}

func (c *CreateIssue) Execute(ctx core.ExecutionContext) error {
	var config CreateIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	req := &IssueRequest{
		Title:       config.Title,
		Description: config.Body,
		Labels:      config.Labels,
		AssigneeIDs: config.Assignees,
	}

	issue, err := client.CreateIssue(context.Background(), config.Project, req)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.issue",
		[]any{issue},
	)
}

func (c *CreateIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *CreateIssue) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateIssue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}
