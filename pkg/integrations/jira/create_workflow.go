package jira

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const CreateWorkflowPayloadType = "jira.workflow.created"

const (
	workflowScopeGlobal  = "GLOBAL"
	workflowScopeProject = "PROJECT"
)

type CreateWorkflow struct{}

type CreateWorkflowSpec struct {
	Name        string                   `json:"name" mapstructure:"name"`
	Description string                   `json:"description" mapstructure:"description"`
	Scope       string                   `json:"scope" mapstructure:"scope"`
	Project     string                   `json:"project" mapstructure:"project"`
	Statuses    []WorkflowStatusSpec     `json:"statuses" mapstructure:"statuses"`
	Transitions []WorkflowTransitionSpec `json:"transitions" mapstructure:"transitions"`
}

type WorkflowStatusSpec struct {
	Name     string `json:"name" mapstructure:"name"`
	Category string `json:"category" mapstructure:"category"`
}

type WorkflowTransitionSpec struct {
	Name string   `json:"name" mapstructure:"name"`
	From []string `json:"from" mapstructure:"from"`
	To   string   `json:"to" mapstructure:"to"`
	Type string   `json:"type" mapstructure:"type"`
}

type CreateWorkflowOutput struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Version WorkflowVersion `json:"version"`
}

func (c *CreateWorkflow) Name() string {
	return "jira.createWorkflow"
}

func (c *CreateWorkflow) Label() string {
	return "Create Workflow"
}

func (c *CreateWorkflow) Description() string {
	return "Create a Jira workflow"
}

func (c *CreateWorkflow) Documentation() string {
	return `The Create Workflow component creates a Jira workflow with statuses and transitions.

## Use Cases

- **Service request lifecycle**: define a standard request workflow before assigning it through a workflow scheme
- **JSM rollout automation**: create a workflow from a SuperPlane canvas as part of project provisioning
- **Environment parity**: recreate workflow structure across Jira sites

## Configuration

- **Name**: Workflow name.
- **Description**: Optional workflow description.
- **Scope**: Global or project-scoped. Project-scoped workflows require a Jira project.
- **Project**: Required when scope is Project.
- **Statuses**: List of statuses with a category: TODO, IN_PROGRESS, or DONE.
- **Transitions**: List of transitions with a target status. Directed transitions use the From status list; Global transitions are available from any status.

## Output

Returns the created workflow's ` + "`id`" + `, ` + "`name`" + `, and ` + "`version`" + `.

## Notes

- Requires Jira admin permissions (` + "`manage:jira-configuration`" + `).
- Jira creates workflows independently from projects. Use Assign Workflow To Project to apply a workflow scheme to a company-managed project.
- New issues enter the **first listed status**. SuperPlane injects the Jira-required initial transition pointing at that status; the order of the Statuses list determines the starting state.`
}

func (c *CreateWorkflow) Icon() string {
	return "jira"
}

func (c *CreateWorkflow) Color() string {
	return "blue"
}

func (c *CreateWorkflow) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateWorkflow) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Workflow name",
			Placeholder: "Service request workflow",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Optional workflow description",
		},
		{
			Name:        "scope",
			Label:       "Scope",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Create the workflow globally or scoped to a project",
			Default:     workflowScopeGlobal,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Global", Value: workflowScopeGlobal},
						{Label: "Project", Value: workflowScopeProject},
					},
				},
			},
		},
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Project for a project-scoped workflow",
			Placeholder: "Select a project",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "scope", Values: []string{workflowScopeProject}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "scope", Values: []string{workflowScopeProject}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: "project"},
			},
		},
		{
			Name:        "statuses",
			Label:       "Statuses",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "Workflow statuses",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Status",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "name",
								Label:       "Name",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Placeholder: "To Do",
							},
							{
								Name:     "category",
								Label:    "Category",
								Type:     configuration.FieldTypeSelect,
								Required: true,
								Default:  "TODO",
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "To do", Value: "TODO"},
											{Label: "In progress", Value: "IN_PROGRESS"},
											{Label: "Done", Value: "DONE"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "transitions",
			Label:       "Transitions",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "Workflow transitions",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Transition",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "name",
								Label:       "Name",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Placeholder: "Start work",
							},
							{
								Name:     "type",
								Label:    "Type",
								Type:     configuration.FieldTypeSelect,
								Required: true,
								Default:  "directed",
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Directed", Value: "directed"},
											{Label: "Global", Value: "global"},
										},
									},
								},
							},
							{
								Name:        "from",
								Label:       "From",
								Type:        configuration.FieldTypeList,
								Required:    false,
								Description: "Source status names for directed transitions. Use any for a global transition.",
								TypeOptions: &configuration.TypeOptions{
									List: &configuration.ListTypeOptions{
										ItemLabel:      "Status",
										ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
									},
								},
							},
							{
								Name:        "to",
								Label:       "To",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Target status name",
								Placeholder: "In Progress",
							},
						},
					},
				},
			},
		},
	}
}

func (c *CreateWorkflow) Setup(ctx core.SetupContext) error {
	spec := CreateWorkflowSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if strings.TrimSpace(spec.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if err := validateWorkflowSpec(spec); err != nil {
		return err
	}

	meta := NodeMetadata{WorkflowName: strings.TrimSpace(spec.Name)}
	if normalizeWorkflowScope(spec.Scope) == workflowScopeProject {
		project, err := requireProject(ctx.HTTP, ctx.Integration, strings.TrimSpace(spec.Project))
		if err != nil {
			return err
		}
		meta.Project = project
	}

	return ctx.Metadata.Set(meta)
}

func (c *CreateWorkflow) Execute(ctx core.ExecutionContext) error {
	spec := CreateWorkflowSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}
	if strings.TrimSpace(spec.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if err := validateWorkflowSpec(spec); err != nil {
		return err
	}

	var project *Project
	if normalizeWorkflowScope(spec.Scope) == workflowScopeProject {
		var err error
		project, err = requireProject(ctx.HTTP, ctx.Integration, strings.TrimSpace(spec.Project))
		if err != nil {
			return err
		}
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	req, err := buildCreateWorkflowRequest(spec, project)
	if err != nil {
		return err
	}

	resp, err := client.CreateWorkflow(req)
	if err != nil {
		if strings.Contains(err.Error(), "403") {
			return fmt.Errorf("failed to create workflow: %v — the API token must belong to a Jira admin", err)
		}
		return fmt.Errorf("failed to create workflow: %v", err)
	}
	if len(resp.Workflows) == 0 {
		return fmt.Errorf("failed to create workflow: Jira returned no workflows")
	}

	created := resp.Workflows[0]
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CreateWorkflowPayloadType,
		[]any{CreateWorkflowOutput{ID: created.ID, Name: created.Name, Version: created.Version}},
	)
}

func normalizeWorkflowScope(scope string) string {
	switch strings.ToUpper(strings.TrimSpace(scope)) {
	case workflowScopeProject:
		return workflowScopeProject
	default:
		return workflowScopeGlobal
	}
}

func validateWorkflowSpec(spec CreateWorkflowSpec) error {
	if normalizeWorkflowScope(spec.Scope) == workflowScopeProject && strings.TrimSpace(spec.Project) == "" {
		return fmt.Errorf("project is required when scope is Project")
	}
	if len(spec.Statuses) == 0 {
		return fmt.Errorf("at least one status is required")
	}
	if len(spec.Transitions) == 0 {
		return fmt.Errorf("at least one transition is required")
	}

	statusNames := map[string]bool{}
	statusDisplay := make([]string, 0, len(spec.Statuses))
	for i, status := range spec.Statuses {
		name := strings.TrimSpace(status.Name)
		if name == "" {
			return fmt.Errorf("statuses[%d].name is required", i)
		}
		key := strings.ToLower(name)
		if statusNames[key] {
			return fmt.Errorf("duplicate status %q", name)
		}
		statusNames[key] = true
		statusDisplay = append(statusDisplay, name)
		if !slices.Contains([]string{"TODO", "IN_PROGRESS", "DONE"}, strings.ToUpper(strings.TrimSpace(status.Category))) {
			return fmt.Errorf("statuses[%d].category must be TODO, IN_PROGRESS, or DONE", i)
		}
	}

	for i, transition := range spec.Transitions {
		if strings.TrimSpace(transition.Name) == "" {
			return fmt.Errorf("transitions[%d].name is required", i)
		}
		target := strings.TrimSpace(transition.To)
		if target == "" {
			return fmt.Errorf("transitions[%d].to is required", i)
		}
		if !statusNames[strings.ToLower(target)] {
			return fmt.Errorf("transitions[%d].to references unknown status %q (available: %s)", i, target, formatAvailableStatuses(statusDisplay))
		}
		if workflowTransitionType(transition) == "global" {
			continue
		}
		if len(transition.From) == 0 {
			return fmt.Errorf("transitions[%d].from is required for directed transitions", i)
		}
		for _, from := range transition.From {
			source := strings.TrimSpace(from)
			if strings.EqualFold(source, "any") {
				continue
			}
			if !statusNames[strings.ToLower(source)] {
				return fmt.Errorf("transitions[%d].from references unknown status %q (available: %s)", i, source, formatAvailableStatuses(statusDisplay))
			}
		}
	}

	return nil
}

func formatAvailableStatuses(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = fmt.Sprintf("%q", name)
	}
	return strings.Join(quoted, ", ")
}

func buildCreateWorkflowRequest(spec CreateWorkflowSpec, project *Project) (*CreateWorkflowRequest, error) {
	scope := WorkflowScope{Type: normalizeWorkflowScope(spec.Scope)}
	if scope.Type == workflowScopeProject {
		if project == nil || strings.TrimSpace(project.ID) == "" {
			return nil, fmt.Errorf("project id is required for project-scoped workflows")
		}
		scope.Project = &WorkflowScopeProjectRef{ID: strings.TrimSpace(project.ID)}
	}

	statusRefs := map[string]string{}
	statuses := make([]WorkflowStatusUpdate, 0, len(spec.Statuses))
	workflowStatuses := make([]WorkflowCreateStatus, 0, len(spec.Statuses))
	for i, status := range spec.Statuses {
		name := strings.TrimSpace(status.Name)
		ref := workflowStatusReference(name)
		statusRefs[strings.ToLower(name)] = ref
		statuses = append(statuses, WorkflowStatusUpdate{
			Description:     "",
			Name:            name,
			StatusCategory:  strings.ToUpper(strings.TrimSpace(status.Category)),
			StatusReference: ref,
		})
		workflowStatuses = append(workflowStatuses, WorkflowCreateStatus{
			Layout:          WorkflowLayout{X: 115 + float64(i*200), Y: -16},
			Properties:      map[string]any{},
			StatusReference: ref,
		})
	}

	transitions := []WorkflowCreateTransition{
		newWorkflowTransition("1", "Create", "INITIAL", statusRefs[strings.ToLower(strings.TrimSpace(spec.Statuses[0].Name))], nil),
	}
	for i, transition := range spec.Transitions {
		targetRef := statusRefs[strings.ToLower(strings.TrimSpace(transition.To))]
		transitionType := strings.ToUpper(workflowTransitionType(transition))
		var links []WorkflowTransitionLink
		if transitionType == "DIRECTED" {
			links = make([]WorkflowTransitionLink, 0, len(transition.From))
			for _, from := range transition.From {
				if strings.EqualFold(strings.TrimSpace(from), "any") {
					transitionType = "GLOBAL"
					links = nil
					break
				}
				links = append(links, WorkflowTransitionLink{
					FromPort:            0,
					FromStatusReference: statusRefs[strings.ToLower(strings.TrimSpace(from))],
					ToPort:              1,
				})
			}
		}
		transitions = append(transitions, newWorkflowTransition(
			strconv.Itoa((i+2)*10+1),
			strings.TrimSpace(transition.Name),
			transitionType,
			targetRef,
			links,
		))
	}

	return &CreateWorkflowRequest{
		Scope:    scope,
		Statuses: statuses,
		Workflows: []WorkflowCreate{
			{
				Description:      strings.TrimSpace(spec.Description),
				Name:             strings.TrimSpace(spec.Name),
				StartPointLayout: WorkflowLayout{X: -100.00030899047852, Y: -153.00020599365234},
				Statuses:         workflowStatuses,
				Transitions:      transitions,
			},
		},
	}, nil
}

// workflowStatusReference returns a deterministic id used to wire a status to
// the transitions that reference it. Jira treats statusReference as local to a
// single /workflows/create request, so name-derived UUIDs are safe — no
// cross-request stability is implied.
func workflowStatusReference(name string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("superplane:jira:workflow-status:"+strings.ToLower(strings.TrimSpace(name)))).String()
}

func workflowTransitionType(transition WorkflowTransitionSpec) string {
	if strings.EqualFold(strings.TrimSpace(transition.Type), "global") {
		return "global"
	}
	return "directed"
}

func newWorkflowTransition(id, name, transitionType, toStatusReference string, links []WorkflowTransitionLink) WorkflowCreateTransition {
	if links == nil {
		links = []WorkflowTransitionLink{}
	}
	return WorkflowCreateTransition{
		Actions:           []any{},
		Description:       "",
		ID:                id,
		Links:             links,
		Name:              name,
		Properties:        map[string]any{},
		ToStatusReference: toStatusReference,
		Triggers:          []any{},
		Type:              transitionType,
		Validators:        []any{},
	}
}

func (c *CreateWorkflow) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateWorkflow) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateWorkflow) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateWorkflow) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateWorkflow) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateWorkflow) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
