package jira

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const CreateIssuePayloadType = "jira.issue"

type CreateIssue struct{}

type CreateIssueSpec struct {
	Project     string `json:"project"`
	IssueType   string `json:"issueType"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
}

func (c *CreateIssue) Name() string {
	return "jira.createIssue"
}

func (c *CreateIssue) Label() string {
	return "Create Issue"
}

func (c *CreateIssue) Description() string {
	return "Create a new issue in Jira"
}

func (c *CreateIssue) Documentation() string {
	return `The Create Issue component creates a new issue in Jira.

## Use Cases

- **Task creation**: Automatically create tasks from workflow events
- **Bug tracking**: Create bugs from error detection systems
- **Feature requests**: Generate feature request issues from external inputs

## Configuration

- **Project**: The Jira project to create the issue in
- **Issue Type**: The type of issue (e.g. Task, Bug, Story)
- **Summary**: The issue summary/title
- **Description**: Optional description text

## Output

Returns the created issue including:
- **id**: The issue ID
- **key**: The issue key (e.g. PROJ-123)
- **self**: API URL for the issue`
}

func (c *CreateIssue) Icon() string {
	return "jira"
}

func (c *CreateIssue) Color() string {
	return "blue"
}

func (c *CreateIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Jira project to create the issue in",
			Placeholder: "Select a project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:        "issueType",
			Label:       "Issue Type",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The type of issue (e.g. Task, Bug, Story)",
			Placeholder: "Task",
		},
		{
			Name:        "summary",
			Label:       "Summary",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The issue summary/title",
			Placeholder: "Issue summary",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Description: "Optional description text",
		},
	}
}

func (c *CreateIssue) Setup(ctx core.SetupContext) error {
	spec := CreateIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Project == "" {
		return fmt.Errorf("project is required")
	}

	if spec.IssueType == "" {
		return fmt.Errorf("issueType is required")
	}

	if spec.Summary == "" {
		return fmt.Errorf("summary is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	projects, err := client.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %v", err)
	}

	var project *Project
	for _, p := range projects {
		if p.Key == spec.Project {
			project = &p
			break
		}
	}

	if project == nil {
		return fmt.Errorf("project %s not found", spec.Project)
	}

	return ctx.Metadata.Set(NodeMetadata{Project: project})
}

func (c *CreateIssue) Execute(ctx core.ExecutionContext) error {
	spec := CreateIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	req := &CreateIssueRequest{
		Fields: CreateIssueFields{
			Project:     ProjectRef{Key: spec.Project},
			IssueType:   IssueType{Name: spec.IssueType},
			Summary:     spec.Summary,
			Description: WrapInADF(spec.Description),
		},
	}

	response, err := client.CreateIssue(req)
	if err != nil {
		return fmt.Errorf("failed to create issue: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CreateIssuePayloadType,
		[]any{response},
	)
}

func (c *CreateIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateIssue) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateIssue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}
