package jira

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const CreateIssuePayloadType = "jira.issue"

type CreateIssue struct{}

type CreateIssueSpec struct {
	Project     string `json:"project" mapstructure:"project"`
	IssueType   string `json:"issueType" mapstructure:"issueType"`
	Summary     string `json:"summary" mapstructure:"summary"`
	Description string `json:"description" mapstructure:"description"`
	Assignee    string `json:"assignee" mapstructure:"assignee"`
	Status      string `json:"status" mapstructure:"status"`
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
- **Issue Type**: The type of issue (scoped to the chosen project)
- **Summary**: The issue summary/title
- **Description**: Optional description text
- **Assignee**: Optional Jira user to assign the issue to
- **Status**: Optional initial status. Jira always creates issues in the workflow's initial state, so when this is set the component executes a transition immediately after create. The status must be reachable via a transition from the initial state.

## Output

Returns the created issue including:
- **id**: The issue ID
- **key**: The issue key (e.g. PROJ-123)
- **self**: API URL for the issue
- **fields**: Full issue fields after any status transition`
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
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The type of issue (e.g. Task, Bug, Story)",
			Placeholder: "Select an issue type",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "issueType",
					UseNameAsValue: true,
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
			Name:        "summary",
			Label:       "Summary",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The issue summary/title",
			Placeholder: "Issue summary",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional description text",
		},
		{
			Name:        "assignee",
			Label:       "Assignee",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "User to assign the new issue to",
			Placeholder: "Leave empty to keep the project default",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "assignee",
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
			Name:        "status",
			Label:       "Initial Status",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Move the new issue to this status via a transition (must be reachable from the workflow's initial state)",
			Placeholder: "Leave empty to keep the workflow default",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "issueStatus",
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "project",
							ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
						},
					},
				},
			},
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

	project, err := requireProject(ctx.HTTP, ctx.Integration, spec.Project)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(NodeMetadata{
		Project:   project,
		IssueType: spec.IssueType,
		Status:    spec.Status,
	})
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
	if strings.TrimSpace(spec.Assignee) != "" {
		req.Fields.Assignee = &UserRef{AccountID: strings.TrimSpace(spec.Assignee)}
	}

	created, err := client.CreateIssue(req)
	if err != nil {
		return fmt.Errorf("failed to create issue: %v", err)
	}

	if status := strings.TrimSpace(spec.Status); status != "" {
		if err := applyStatus(client, created.Key, status); err != nil {
			return fmt.Errorf("issue %s created, but failed to apply status %q: %v", created.Key, status, err)
		}
	}

	issue, err := client.GetIssue(created.Key)
	if err != nil {
		return fmt.Errorf("failed to fetch created issue: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CreateIssuePayloadType,
		[]any{issue},
	)
}

func (c *CreateIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateIssue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateIssue) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
