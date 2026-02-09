package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_create_issue.json
var exampleOutputCreateIssue []byte

type CreateIssue struct{}

type CreateIssueConfiguration struct {
	Project   string   `mapstructure:"project"`
	Title     string   `mapstructure:"title"`
	Body      string   `mapstructure:"body"`
	Assignees []string `mapstructure:"assignees"`
	Labels    []string `mapstructure:"labels"`
	Milestone string   `mapstructure:"milestone"`
	DueDate   string   `mapstructure:"dueDate"`
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
	return `The Create Issue component creates a new issue in a specified GitLab project.

## Use Cases

- **Automated Bug Reporting**: Create issues when a monitoring system detects an error
- **Task Management**: Automatically create tasks for new employee onboarding
- **Feedback Loop**: Turn customer feedback into actionable issues

## Configuration

- **Project** (required): The GitLab project where the issue will be created
- **Title** (required): The title of the new issue
- **Description** (optional): The description/body of the issue
- **Assignees** (optional): Users to assign the issue to
- **Labels** (optional): Labels to apply to the issue (e.g., bug, enhancement)
- **Milestone** (optional): Milestone to associate with the issue
- **Due Date** (optional): Date when the issue is due

## Output

The component outputs the created issue object, including:
- **id**: The internal ID of the issue
- **iid**: The project-relative ID of the issue
- **web_url**: The URL to view the issue in GitLab
- **state**: The current state of the issue (opened/closed)`
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
	var example map[string]any
	if err := json.Unmarshal(exampleOutputCreateIssue, &example); err != nil {
		return map[string]any{}
	}
	return example
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
					Type: ResourceTypeProject,
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
			Label:    "Assignees",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeMember,
					Multi: true,
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
		{
			Name:     "milestone",
			Label:    "Milestone",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeMilestone,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "project",
							ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
						},
					},
				},
			},
		},
		{
			Name:     "dueDate",
			Label:    "Due Date",
			Type:     configuration.FieldTypeDate,
			Required: false,
		},
	}
}

func (c *CreateIssue) Setup(ctx core.SetupContext) error {
	var config CreateIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	if config.Title == "" {
		return fmt.Errorf("title is required")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
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

	var assigneeIDs []int
	for _, idStr := range config.Assignees {
		var id int
		if _, err := fmt.Sscanf(idStr, "%d", &id); err == nil {
			assigneeIDs = append(assigneeIDs, id)
		}
	}

	var milestoneID *int
	if config.Milestone != "" {
		var id int
		if _, err := fmt.Sscanf(config.Milestone, "%d", &id); err == nil {
			milestoneID = &id
		}
	}

	req := &IssueRequest{
		Title:       config.Title,
		Description: config.Body,
		Labels:      strings.Join(config.Labels, ","),
		AssigneeIDs: assigneeIDs,
		MilestoneID: milestoneID,
		DueDate:     config.DueDate,
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
