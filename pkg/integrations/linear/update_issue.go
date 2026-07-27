package linear

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateIssue struct{}

type UpdateIssueSpec struct {
	Team        string   `json:"team" mapstructure:"team"`
	Issue       string   `json:"issue" mapstructure:"issue"`
	Title       string   `json:"title" mapstructure:"title"`
	Description string   `json:"description" mapstructure:"description"`
	State       string   `json:"state" mapstructure:"state"`
	Assignee    string   `json:"assignee" mapstructure:"assignee"`
	Priority    string   `json:"priority" mapstructure:"priority"`
	Labels      []string `json:"labels" mapstructure:"labels"`
	Project     string   `json:"project" mapstructure:"project"`
}

// updateIssueToggles records which optional fields were switched on in the UI.
// mapstructure decodes both "never toggled on" and "toggled on but cleared" to
// the same Go zero value, so presence is read from the raw configuration map to
// tell them apart - clearing the assignee (toggled on, empty) must reach Linear
// as an unassign, not be dropped as "no change".
type updateIssueToggles struct {
	Title       bool
	Description bool
	State       bool
	Assignee    bool
	Priority    bool
	Labels      bool
	Project     bool
}

func newUpdateIssueToggles(raw map[string]any) updateIssueToggles {
	enabled := func(field string) bool {
		v, ok := raw[field]
		return ok && v != nil
	}

	return updateIssueToggles{
		Title:       enabled("title"),
		Description: enabled("description"),
		State:       enabled("state"),
		Assignee:    enabled("assignee"),
		Priority:    enabled("priority"),
		Labels:      enabled("labels"),
		Project:     enabled("project"),
	}
}

func (t updateIssueToggles) hasUpdates() bool {
	return t.Title || t.Description || t.State || t.Assignee || t.Priority || t.Labels || t.Project
}

func (c *UpdateIssue) Name() string {
	return "linear.updateIssue"
}

func (c *UpdateIssue) Label() string {
	return "Update Issue"
}

func (c *UpdateIssue) Description() string {
	return "Update an existing issue in Linear"
}

func (c *UpdateIssue) Documentation() string {
	return `The Update Issue component modifies an existing Linear issue: its title, description, status,
assignee, priority, labels, or project.

## Use Cases

- **Status updates**: Move an issue to a new workflow state based on workflow results
- **Automated triage**: Set the priority, assignee, or labels when a workflow processes the issue
- **Cross-tool sync**: Mirror state from another system into Linear

## Configuration

- **Team** (required): The Linear team whose statuses, members, labels and projects populate the pickers
  below. Select the team the issue belongs to.
- **Issue** (required): The issue to update, by identifier (e.g. ENG-142) or ID. Supports expressions.
- **Title** (toggle): New title for the issue
- **Description** (toggle): New description, written in Markdown
- **Status** (toggle): New workflow state for the issue
- **Assignee** (toggle): Team member to assign the issue to
- **Priority** (toggle): No priority, Urgent, High, Medium or Low
- **Labels** (toggle): Labels to set on the issue, replacing any existing labels
- **Project** (toggle): Project to move the issue to

Each field besides Team and Issue is toggled on individually, so only the fields you enable are sent in
the update. At least one must be enabled. Enabling a field with an empty value clears it - toggling on
Assignee with no one selected unassigns the issue, toggling on Labels with nothing selected removes all
of them, and toggling on Project with nothing selected removes the issue from its project. Title and
Status are the exception: Linear does not allow a blank title or a missing status, so they must have a
value when enabled.

## Output

Returns the updated issue, including its ` + "`identifier`" + `, ` + "`url`" + `, ` + "`title`" + `, ` + "`state`" + `,
` + "`team`" + `, ` + "`assignee`" + `, ` + "`priorityLabel`" + `, ` + "`project`" + ` and ` + "`labels`" + `.

## Permissions

The user who authorized the Linear connection must be able to edit the issue. SuperPlane's OAuth
connection includes the **write** scope, which covers updating issues.`
}

func (c *UpdateIssue) Icon() string {
	return "linear"
}

func (c *UpdateIssue) Color() string {
	return "indigo"
}

func (c *UpdateIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "team",
			Label:       "Team",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Linear team whose statuses, members, labels and projects populate the pickers below",
			Placeholder: "Select a team",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeTeam,
				},
			},
		},
		{
			Name:        "issue",
			Label:       "Issue",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The issue to update, by identifier (e.g. ENG-142) or ID",
			Placeholder: "ENG-142",
		},
		{
			Name:      "title",
			Label:     "Title",
			Type:      configuration.FieldTypeString,
			Required:  false,
			Togglable: true,
		},
		{
			Name:      "description",
			Label:     "Description",
			Type:      configuration.FieldTypeText,
			Required:  false,
			Togglable: true,
		},
		{
			Name:      "state",
			Label:     "Status",
			Type:      configuration.FieldTypeIntegrationResource,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeWorkflowState,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "team",
							ValueFrom: &configuration.ParameterValueFrom{Field: "team"},
						},
					},
				},
			},
		},
		{
			Name:      "assignee",
			Label:     "Assignee",
			Type:      configuration.FieldTypeIntegrationResource,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeMember,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "team",
							ValueFrom: &configuration.ParameterValueFrom{Field: "team"},
						},
					},
				},
			},
		},
		{
			Name:      "priority",
			Label:     "Priority",
			Type:      configuration.FieldTypeSelect,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "No priority", Value: "0"},
						{Label: "Urgent", Value: "1"},
						{Label: "High", Value: "2"},
						{Label: "Medium", Value: "3"},
						{Label: "Low", Value: "4"},
					},
				},
			},
		},
		{
			Name:      "labels",
			Label:     "Labels",
			Type:      configuration.FieldTypeIntegrationResource,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeLabel,
					Multi: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "team",
							ValueFrom: &configuration.ParameterValueFrom{Field: "team"},
						},
					},
				},
			},
		},
		{
			Name:      "project",
			Label:     "Project",
			Type:      configuration.FieldTypeIntegrationResource,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "team",
							ValueFrom: &configuration.ParameterValueFrom{Field: "team"},
						},
					},
				},
			},
		},
	}
}

func (c *UpdateIssue) Setup(ctx core.SetupContext) error {
	spec := UpdateIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Team == "" {
		return fmt.Errorf("team is required")
	}

	if strings.TrimSpace(spec.Issue) == "" {
		return fmt.Errorf("issue is required")
	}

	raw, _ := ctx.Configuration.(map[string]any)
	toggles := newUpdateIssueToggles(raw)
	if !toggles.hasUpdates() {
		return fmt.Errorf("at least one field must be enabled to update")
	}

	if _, err := buildUpdateIssueInput(spec, toggles); err != nil {
		return err
	}

	team, err := requireTeam(ctx.Integration, spec.Team)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(NodeMetadata{Team: team})
}

func (c *UpdateIssue) Execute(ctx core.ExecutionContext) error {
	spec := UpdateIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	issueID := strings.TrimSpace(spec.Issue)
	if issueID == "" {
		return fmt.Errorf("issue is required")
	}

	raw, _ := ctx.Configuration.(map[string]any)
	toggles := newUpdateIssueToggles(raw)
	if !toggles.hasUpdates() {
		return fmt.Errorf("at least one field must be enabled to update")
	}

	input, err := buildUpdateIssueInput(spec, toggles)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	issue, err := client.UpdateIssue(issueID, input)
	if err != nil {
		return fmt.Errorf("failed to update issue: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		IssuePayloadType,
		[]any{issue},
	)
}

// buildUpdateIssueInput turns the enabled toggles into an IssueUpdateInput. Only
// enabled fields are included, and a field enabled with an empty value clears
// the corresponding value on the issue where Linear allows it (assignee,
// labels, project). Title and Status cannot be cleared, so they are rejected
// when enabled but empty.
func buildUpdateIssueInput(spec UpdateIssueSpec, toggles updateIssueToggles) (map[string]any, error) {
	input := map[string]any{}

	if toggles.Title {
		title := strings.TrimSpace(spec.Title)
		if title == "" {
			return nil, fmt.Errorf("title cannot be empty")
		}
		input["title"] = title
	}

	if toggles.Description {
		// An enabled but blank description clears the issue description.
		input["description"] = strings.TrimSpace(spec.Description)
	}

	if toggles.State {
		state := strings.TrimSpace(spec.State)
		if state == "" {
			return nil, fmt.Errorf("status cannot be empty")
		}
		input["stateId"] = state
	}

	if toggles.Assignee {
		// An enabled but empty assignee unassigns the issue.
		if assignee := strings.TrimSpace(spec.Assignee); assignee != "" {
			input["assigneeId"] = assignee
		} else {
			input["assigneeId"] = nil
		}
	}

	if toggles.Priority {
		priority := strings.TrimSpace(spec.Priority)
		if priority == "" {
			return nil, fmt.Errorf("priority cannot be empty")
		}

		value, err := strconv.Atoi(priority)
		if err != nil {
			return nil, fmt.Errorf("invalid priority %q: must be a number between 0 and 4", priority)
		}

		if value < 0 || value > 4 {
			return nil, fmt.Errorf("invalid priority %d: must be between 0 and 4", value)
		}

		input["priority"] = value
	}

	if toggles.Labels {
		// An enabled but empty label set removes all labels from the issue.
		labels := []string{}
		for _, label := range spec.Labels {
			if trimmed := strings.TrimSpace(label); trimmed != "" {
				labels = append(labels, trimmed)
			}
		}
		input["labelIds"] = labels
	}

	if toggles.Project {
		// An enabled but empty project removes the issue from its project.
		if project := strings.TrimSpace(spec.Project); project != "" {
			input["projectId"] = project
		} else {
			input["projectId"] = nil
		}
	}

	return input, nil
}

func (c *UpdateIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateIssue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateIssue) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
