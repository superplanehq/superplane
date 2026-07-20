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

type CreateIssue struct{}

type CreateIssueSpec struct {
	Team        string   `json:"team" mapstructure:"team"`
	Title       string   `json:"title" mapstructure:"title"`
	Description string   `json:"description" mapstructure:"description"`
	State       string   `json:"state" mapstructure:"state"`
	Assignee    string   `json:"assignee" mapstructure:"assignee"`
	Priority    string   `json:"priority" mapstructure:"priority"`
	Labels      []string `json:"labels" mapstructure:"labels"`
}

func (c *CreateIssue) Name() string {
	return "linear.createIssue"
}

func (c *CreateIssue) Label() string {
	return "Create Issue"
}

func (c *CreateIssue) Description() string {
	return "Create a new issue in Linear"
}

func (c *CreateIssue) Documentation() string {
	return `The Create Issue component creates a new issue in a Linear team.

## Use Cases

- **Bug tracking**: Open a Linear issue when an alert fires or a pipeline fails
- **Task creation**: Turn incoming requests into tracked work automatically
- **Traceability**: Mirror an issue from another tracker into Linear

## Configuration

- **Team** (required): The Linear team to create the issue in
- **Title** (required): The issue title
- **Description** (optional): Issue description, written in Markdown
- **Status** (optional): Workflow state for the new issue. Leave empty to use the team's default — Linear
  puts new issues in the first Backlog state, or in Triage when the team has triage enabled.
- **Assignee** (optional): Team member to assign the issue to
- **Priority** (optional): No priority, Urgent, High, Medium or Low
- **Labels** (optional): Labels to apply to the issue

## Output

Returns the created issue, including its ` + "`identifier`" + ` (e.g. ENG-142), ` + "`url`" + `, ` + "`title`" + `,
` + "`state`" + `, ` + "`team`" + `, ` + "`assignee`" + `, ` + "`priorityLabel`" + ` and ` + "`labels`" + `.

## Permissions

The API key owner must be a member of the selected team, and the key needs **Write** or
**Create issues** permission.`
}

func (c *CreateIssue) Icon() string {
	return "linear"
}

func (c *CreateIssue) Color() string {
	return "indigo"
}

func (c *CreateIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "team",
			Label:       "Team",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Linear team to create the issue in",
			Placeholder: "Select a team",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeTeam,
				},
			},
		},
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The issue title",
			Placeholder: "Issue title",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Issue description, written in Markdown",
		},
		{
			Name:        "state",
			Label:       "Status",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Workflow state for the new issue",
			Placeholder: "Leave empty to use the team default",
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
			Name:        "assignee",
			Label:       "Assignee",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Team member to assign the issue to",
			Placeholder: "Leave empty to create the issue unassigned",
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
			Name:        "priority",
			Label:       "Priority",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Issue priority",
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
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Labels to apply to the issue",
			Placeholder: "Select labels",
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
	}
}

func (c *CreateIssue) Setup(ctx core.SetupContext) error {
	spec := CreateIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Team == "" {
		return fmt.Errorf("team is required")
	}

	if strings.TrimSpace(spec.Title) == "" {
		return fmt.Errorf("title is required")
	}

	team, err := requireTeam(ctx.Integration, spec.Team)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(NodeMetadata{Team: team})
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

	input, err := buildCreateIssueInput(spec)
	if err != nil {
		return err
	}

	issue, err := client.CreateIssue(input)
	if err != nil {
		return fmt.Errorf("failed to create issue: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		IssuePayloadType,
		[]any{issue},
	)
}

func buildCreateIssueInput(spec CreateIssueSpec) (map[string]any, error) {
	input := map[string]any{
		"teamId": spec.Team,
		"title":  strings.TrimSpace(spec.Title),
	}

	if description := strings.TrimSpace(spec.Description); description != "" {
		input["description"] = description
	}

	if state := strings.TrimSpace(spec.State); state != "" {
		input["stateId"] = state
	}

	if assignee := strings.TrimSpace(spec.Assignee); assignee != "" {
		input["assigneeId"] = assignee
	}

	if priority := strings.TrimSpace(spec.Priority); priority != "" {
		value, err := strconv.Atoi(priority)
		if err != nil {
			return nil, fmt.Errorf("invalid priority %q: must be a number between 0 and 4", priority)
		}

		if value < 0 || value > 4 {
			return nil, fmt.Errorf("invalid priority %d: must be between 0 and 4", value)
		}

		input["priority"] = value
	}

	labels := []string{}
	for _, label := range spec.Labels {
		if trimmed := strings.TrimSpace(label); trimmed != "" {
			labels = append(labels, trimmed)
		}
	}

	if len(labels) > 0 {
		input["labelIds"] = labels
	}

	return input, nil
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
