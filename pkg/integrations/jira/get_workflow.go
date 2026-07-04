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

const GetWorkflowPayloadType = "jira.workflow"

type GetWorkflow struct{}

type GetWorkflowSpec struct {
	Project  string `json:"project" mapstructure:"project"`
	IssueKey string `json:"issueKey" mapstructure:"issueKey"`
}

// WorkflowStatus is a status inside a workflow definition. Includes whether
// it's the issue's current status so the canvas can render the workflow as
// a state machine with the current location highlighted.
type WorkflowStatus struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Category  string `json:"category,omitempty"`
	IsCurrent bool   `json:"isCurrent,omitempty"`
}

// WorkflowAvailableTransition is one transition the issue can take right now
// from its current status.
type WorkflowAvailableTransition struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ToStatusID string `json:"toStatusId,omitempty"`
	ToStatus   string `json:"toStatus"`
}

// GetWorkflowOutput summarizes the workflow currently bound to an issue:
// where the issue is now, every status the workflow defines, and every
// transition it can take from the current state.
type GetWorkflowOutput struct {
	IssueKey             string                        `json:"issueKey"`
	IssueType            string                        `json:"issueType,omitempty"`
	ProjectKey           string                        `json:"projectKey,omitempty"`
	WorkflowName         string                        `json:"workflowName,omitempty"`
	WorkflowSchemeID     string                        `json:"workflowSchemeId,omitempty"`
	WorkflowSchemeName   string                        `json:"workflowSchemeName,omitempty"`
	CurrentStatus        string                        `json:"currentStatus,omitempty"`
	CurrentStatusID      string                        `json:"currentStatusId,omitempty"`
	Statuses             []WorkflowStatus              `json:"statuses,omitempty"`
	AvailableTransitions []WorkflowAvailableTransition `json:"availableTransitions,omitempty"`
}

func (c *GetWorkflow) Name() string {
	return "jira.getWorkflow"
}

func (c *GetWorkflow) Label() string {
	return "Get Workflow"
}

func (c *GetWorkflow) Description() string {
	return "Get the Jira workflow bound to an issue, including its current status and reachable transitions"
}

func (c *GetWorkflow) Documentation() string {
	return `The Get Workflow component returns the Jira workflow that governs a given issue.

## Use Cases

- **State-machine introspection**: see every status in the workflow plus where the issue is right now
- **Routing decisions**: branch on which transitions are currently reachable before running ` + "`transitionIssue`" + `
- **Operator dashboards**: render the workflow as a graph next to the issue

## Configuration

- **Project**: The Jira project the issue belongs to.
- **Issue Key**: Jira issue key, for example ` + "`PROJ-123`" + `.

## Output

Returns:

- ` + "`workflowName`" + ` and ` + "`workflowSchemeName`" + ` — the workflow scheme assigned to the project and the workflow it routes the issue's type to.
- ` + "`currentStatus`" + ` / ` + "`currentStatusId`" + ` — where the issue is now.
- ` + "`statuses`" + ` — every status the workflow defines (with ` + "`isCurrent`" + ` set on the current one).
- ` + "`availableTransitions`" + ` — transitions reachable from the issue's current state, each with the transition id, name, and target status.

## Notes

- Resolving the bound workflow goes ` + "`issue → project + issue type → workflow scheme → workflow`" + `. Team-managed (next-gen) projects don't expose a workflow scheme; in that case ` + "`workflowName`" + ` and ` + "`statuses`" + ` are empty but ` + "`currentStatus`" + ` and ` + "`availableTransitions`" + ` are still populated.
- The ` + "`availableTransitions`" + ` list reflects workflow rules, conditions, and the calling user's permissions — it is exactly what Jira would offer in the issue view.`
}

func (c *GetWorkflow) Icon() string {
	return "jira"
}

func (c *GetWorkflow) Color() string {
	return "blue"
}

func (c *GetWorkflow) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetWorkflow) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Jira project the issue belongs to",
			Placeholder: "Select a project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: "project"},
			},
		},
		{
			Name:        "issueKey",
			Label:       "Issue Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The issue key (e.g. PROJ-123)",
			Placeholder: "PROJ-123",
		},
	}
}

func (c *GetWorkflow) Setup(ctx core.SetupContext) error {
	spec := GetWorkflowSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if strings.TrimSpace(spec.Project) == "" {
		return fmt.Errorf("project is required")
	}
	if strings.TrimSpace(spec.IssueKey) == "" {
		return fmt.Errorf("issueKey is required")
	}

	project, err := requireProject(ctx.HTTP, ctx.Integration, spec.Project)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(NodeMetadata{Project: project})
}

func (c *GetWorkflow) Execute(ctx core.ExecutionContext) error {
	spec := GetWorkflowSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	issueKey := strings.TrimSpace(spec.IssueKey)
	if issueKey == "" {
		return fmt.Errorf("issueKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	issue, err := client.GetIssue(issueKey)
	if err != nil {
		return fmt.Errorf("failed to fetch issue: %v", err)
	}

	output := GetWorkflowOutput{IssueKey: issueKey}
	currentStatusID, currentStatusName := extractIssueStatus(issue)
	output.CurrentStatus = currentStatusName
	output.CurrentStatusID = currentStatusID

	issueTypeName, issueTypeID, projectID, projectKey := extractIssueTypeAndProject(issue)
	output.IssueType = issueTypeName
	output.ProjectKey = projectKey

	transitions, err := client.GetIssueTransitions(issueKey)
	if err != nil {
		return fmt.Errorf("failed to fetch transitions: %v", err)
	}
	output.AvailableTransitions = make([]WorkflowAvailableTransition, 0, len(transitions))
	for _, t := range transitions {
		output.AvailableTransitions = append(output.AvailableTransitions, WorkflowAvailableTransition{
			ID:         t.ID,
			Name:       t.Name,
			ToStatusID: t.To.ID,
			ToStatus:   t.To.Name,
		})
	}

	// Resolving the workflow itself (statuses + scheme) requires a company-managed
	// project. Team-managed projects bind workflows differently and Jira's scheme
	// APIs return an empty list (not an error) for them — we degrade gracefully
	// in that case and still emit current status + available transitions. Any
	// other failure is surfaced so callers don't get partial output that looks
	// successful.
	if projectID != "" {
		scheme, err := client.GetWorkflowSchemeForProject(projectID)
		if err != nil {
			return fmt.Errorf("failed to fetch workflow scheme for project %s: %v", projectID, err)
		}
		if scheme != nil {
			output.WorkflowSchemeID = scheme.ID.String()
			output.WorkflowSchemeName = scheme.Name

			workflowName := resolveWorkflowForIssueType(scheme, issueTypeID)
			output.WorkflowName = workflowName
			if workflowName != "" {
				statuses, err := client.GetWorkflowStatusesByName(workflowName)
				if err != nil {
					return fmt.Errorf("failed to load statuses for workflow %q: %v", workflowName, err)
				}
				output.Statuses = make([]WorkflowStatus, 0, len(statuses))
				for _, s := range statuses {
					output.Statuses = append(output.Statuses, WorkflowStatus{
						ID:        s.ID,
						Name:      s.Name,
						Category:  s.Category,
						IsCurrent: statusMatches(s, currentStatusID, currentStatusName),
					})
				}
			}
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetWorkflowPayloadType,
		[]any{output},
	)
}

// extractIssueStatus pulls the issue's current status id and name out of the
// loosely-typed fields map returned by GetIssue.
func extractIssueStatus(issue *Issue) (id, name string) {
	if issue == nil {
		return "", ""
	}
	status, ok := issue.Fields["status"].(map[string]any)
	if !ok {
		return "", ""
	}
	if v, ok := status["id"].(string); ok {
		id = v
	}
	if v, ok := status["name"].(string); ok {
		name = v
	}
	return id, name
}

// extractIssueTypeAndProject pulls the issue's issue type id + name and the
// project's id + key from the loosely-typed fields map returned by GetIssue.
func extractIssueTypeAndProject(issue *Issue) (issueType, issueTypeID, projectID, projectKey string) {
	if issue == nil {
		return "", "", "", ""
	}
	if it, ok := issue.Fields["issuetype"].(map[string]any); ok {
		if v, ok := it["id"].(string); ok {
			issueTypeID = v
		}
		if v, ok := it["name"].(string); ok {
			issueType = v
		}
	}
	if p, ok := issue.Fields["project"].(map[string]any); ok {
		if v, ok := p["id"].(string); ok {
			projectID = v
		}
		if v, ok := p["key"].(string); ok {
			projectKey = v
		}
	}
	return issueType, issueTypeID, projectID, projectKey
}

// resolveWorkflowForIssueType maps an issue type id to the workflow that the
// scheme routes it through. Jira keys issueTypeMappings by issue type id; we
// read the id from the issue itself rather than create-metadata, which only
// lists types the caller can create (sub-tasks, epics, etc. are often omitted).
func resolveWorkflowForIssueType(scheme *WorkflowSchemeDetail, issueTypeID string) string {
	if scheme == nil {
		return ""
	}
	if id := strings.TrimSpace(issueTypeID); id != "" {
		if wf := strings.TrimSpace(scheme.IssueTypeMappings[id]); wf != "" {
			return wf
		}
	}
	return strings.TrimSpace(scheme.DefaultWorkflow)
}

func statusMatches(s Status, currentID, currentName string) bool {
	if currentID != "" && strings.EqualFold(strings.TrimSpace(s.ID), currentID) {
		return true
	}
	if currentName != "" && strings.EqualFold(strings.TrimSpace(s.Name), strings.TrimSpace(currentName)) {
		return true
	}
	return false
}

func (c *GetWorkflow) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetWorkflow) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetWorkflow) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetWorkflow) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetWorkflow) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetWorkflow) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
