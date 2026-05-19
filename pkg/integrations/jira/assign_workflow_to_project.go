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

const AssignWorkflowToProjectPayloadType = "jira.workflowScheme.assigned"

type AssignWorkflowToProject struct{}

type AssignWorkflowToProjectSpec struct {
	Project        string `json:"project" mapstructure:"project"`
	WorkflowScheme string `json:"workflowScheme" mapstructure:"workflowScheme"`
	DryRun         bool   `json:"dryRun" mapstructure:"dryRun"`
}

type WorkflowSchemeAssignmentOutput struct {
	ProjectID        string `json:"projectId"`
	WorkflowSchemeID string `json:"workflowSchemeId"`
	DraftCreated     bool   `json:"draftCreated"`
	DryRun           bool   `json:"dryRun,omitempty"`
	TaskID           string `json:"taskId,omitempty"`
	TaskStatus       string `json:"taskStatus,omitempty"`
	TaskSelf         string `json:"taskSelf,omitempty"`
}

func (c *AssignWorkflowToProject) Name() string {
	return "jira.assignWorkflowToProject"
}

func (c *AssignWorkflowToProject) Label() string {
	return "Assign Workflow To Project"
}

func (c *AssignWorkflowToProject) Description() string {
	return "Assign a Jira workflow scheme to a company-managed project"
}

func (c *AssignWorkflowToProject) Documentation() string {
	return `The Assign Workflow To Project component switches a Jira project to an existing workflow scheme.

## Use Cases

- **Project provisioning**: apply a known workflow scheme after creating or preparing a Jira project
- **Workflow rollout**: move company-managed projects to an updated workflow scheme
- **Canvas validation**: run in dry-run mode to validate the selected project and scheme without changing Jira

## Configuration

- **Project**: Company-managed Jira project to update.
- **Workflow Scheme**: Existing Jira workflow scheme to assign.
- **Dry Run**: Validate inputs and emit the planned assignment without changing Jira.

## Output

Returns ` + "`projectId`" + `, ` + "`workflowSchemeId`" + `, ` + "`draftCreated`" + `, and any Jira task metadata returned by the workflow scheme switch.

## Notes

- Requires Jira admin permissions (` + "`manage:jira-configuration`" + `).
- Workflow schemes can only be assigned to company-managed projects. Team-managed projects reject workflow scheme changes.
- Jira may start a background task when switching schemes, especially when existing issues need migration.`
}

func (c *AssignWorkflowToProject) Icon() string {
	return "jira"
}

func (c *AssignWorkflowToProject) Color() string {
	return "blue"
}

func (c *AssignWorkflowToProject) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AssignWorkflowToProject) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Company-managed Jira project",
			Placeholder: "Select a project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: "project"},
			},
		},
		{
			Name:        "workflowScheme",
			Label:       "Workflow Scheme",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Existing Jira workflow scheme",
			Placeholder: "Select a workflow scheme",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: "workflowScheme"},
			},
		},
		{
			Name:        "dryRun",
			Label:       "Dry Run",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Validate the assignment without changing Jira",
			Default:     false,
		},
	}
}

func (c *AssignWorkflowToProject) Setup(ctx core.SetupContext) error {
	spec := AssignWorkflowToProjectSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if strings.TrimSpace(spec.Project) == "" {
		return fmt.Errorf("project is required")
	}
	if strings.TrimSpace(spec.WorkflowScheme) == "" {
		return fmt.Errorf("workflowScheme is required")
	}

	project, scheme, err := loadWorkflowSchemeAssignmentSetup(ctx.HTTP, ctx.Integration, spec.Project, spec.WorkflowScheme)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(NodeMetadata{Project: project, WorkflowScheme: scheme})
}

func (c *AssignWorkflowToProject) Execute(ctx core.ExecutionContext) error {
	spec := AssignWorkflowToProjectSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	projectKey := strings.TrimSpace(spec.Project)
	schemeID := strings.TrimSpace(spec.WorkflowScheme)
	if projectKey == "" {
		return fmt.Errorf("project is required")
	}
	if schemeID == "" {
		return fmt.Errorf("workflowScheme is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	project, err := client.GetProject(projectKey)
	if err != nil {
		return fmt.Errorf("failed to fetch project: %v", err)
	}
	if isTeamManagedProject(project) {
		return fmt.Errorf("workflow schemes can only be assigned to company-managed projects")
	}

	if spec.DryRun {
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			AssignWorkflowToProjectPayloadType,
			[]any{WorkflowSchemeAssignmentOutput{
				ProjectID:        project.ID,
				WorkflowSchemeID: schemeID,
				DraftCreated:     false,
				DryRun:           true,
			}},
		)
	}

	resp, err := client.AssignWorkflowSchemeToProject(project.ID, schemeID)
	if err != nil {
		if strings.Contains(err.Error(), "403") {
			return fmt.Errorf("failed to assign workflow scheme: %v — the API token must belong to a Jira admin", err)
		}
		return fmt.Errorf("failed to assign workflow scheme: %v", err)
	}

	output := WorkflowSchemeAssignmentOutput{
		ProjectID:        resp.ProjectID,
		WorkflowSchemeID: resp.WorkflowSchemeID,
		DraftCreated:     false,
	}
	if resp.Task != nil {
		output.TaskID = resp.Task.ID.String()
		output.TaskStatus = resp.Task.Status
		output.TaskSelf = resp.Task.Self
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		AssignWorkflowToProjectPayloadType,
		[]any{output},
	)
}

func loadWorkflowSchemeAssignmentSetup(
	httpCtx core.HTTPContext,
	integration core.IntegrationContext,
	projectKey,
	schemeID string,
) (*Project, *WorkflowScheme, error) {
	projectKey = strings.TrimSpace(projectKey)
	schemeID = strings.TrimSpace(schemeID)

	if httpCtx == nil {
		project, err := requireProjectFromMetadata(integration, projectKey)
		return project, &WorkflowScheme{ID: FlexibleString(schemeID)}, err
	}

	client, err := NewClient(httpCtx, integration)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create client: %v", err)
	}

	project, err := client.GetProject(projectKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch project: %v", err)
	}
	if isTeamManagedProject(project) {
		return nil, nil, fmt.Errorf("workflow schemes can only be assigned to company-managed projects")
	}

	schemes, err := client.ListWorkflowSchemes()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list workflow schemes: %v", err)
	}
	for _, scheme := range schemes {
		if scheme.ID.String() == schemeID {
			s := scheme
			return project, &s, nil
		}
	}

	return nil, nil, fmt.Errorf("workflow scheme %s not found", schemeID)
}

func isTeamManagedProject(project *Project) bool {
	if project == nil {
		return false
	}
	style := strings.ToLower(strings.TrimSpace(project.Style))
	return project.Simplified || style == "next-gen" || style == "nextgen" || style == "team-managed"
}

func (c *AssignWorkflowToProject) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AssignWorkflowToProject) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AssignWorkflowToProject) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *AssignWorkflowToProject) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AssignWorkflowToProject) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AssignWorkflowToProject) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
