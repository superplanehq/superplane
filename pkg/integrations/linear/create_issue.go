package linear

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const createIssuePayloadType = "linear.issue"

type CreateIssue struct{}

type CreateIssueSpec struct {
	Team        string   `json:"team"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	AssigneeID  *string  `json:"assigneeId,omitempty"`
	LabelIDs    []string `json:"labelIds,omitempty"`
	Priority    any      `json:"priority,omitempty"` // string from select "0"-"4" or number
	StateID     *string  `json:"stateId,omitempty"`
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
	return `The Create Issue component creates a new issue in Linear.

## Use Cases

- **Task creation**: Automatically create issues from workflow events
- **Bug tracking**: Create issues from alerts or external systems
- **Feature requests**: Generate issues from forms or other triggers

## Configuration

- **Team**: The Linear team to create the issue in
- **Title**: Issue title (required)
- **Description**: Optional description
- **Assignee**, **Labels**, **Priority**, **Status**: Optional

## Output

Returns the created issue: id, identifier, title, description, teamId, stateId, priority.`
}

func (c *CreateIssue) Icon() string {
	return "linear"
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
			Name:        "team",
			Label:       "Team",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Linear team to create the issue in",
			Placeholder: "Select a team",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "team",
				},
			},
		},
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "Issue title",
			Placeholder: "Issue title",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Description: "Optional issue description",
		},
		{
			Name:        "assigneeId",
			Label:       "Assignee ID",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Description: "Optional Linear user ID to assign the issue to",
			Placeholder: "e.g. $['Step'].userId",
		},
		{
			Name:        "labelIds",
			Label:       "Labels",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional labels",
			Placeholder: "Select labels",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "label",
					Multi: true,
				},
			},
		},
		{
			Name:        "priority",
			Label:       "Priority",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Optional priority (0 = none, 1 = urgent, 2 = high, 3 = medium, 4 = low)",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "None", Value: "0"},
						{Label: "Urgent", Value: "1"},
						{Label: "High", Value: "2"},
						{Label: "Medium", Value: "3"},
						{Label: "Low", Value: "4"},
					},
				},
			},
		},
		{
			Name:        "stateId",
			Label:       "Status",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Description: "Optional workflow state ID",
			Placeholder: "e.g. state UUID",
		},
	}
}

func (c *CreateIssue) Setup(ctx core.SetupContext) error {
	spec := CreateIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}
	if spec.Team == "" {
		return fmt.Errorf("team is required")
	}
	if spec.Title == "" {
		return fmt.Errorf("title is required")
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	teams, err := client.ListTeams()
	if err != nil {
		return fmt.Errorf("list teams: %w", err)
	}
	var team *Team
	for i := range teams {
		if teams[i].ID == spec.Team {
			team = &teams[i]
			break
		}
	}
	if team == nil {
		return fmt.Errorf("team %s not found", spec.Team)
	}
	return ctx.Metadata.Set(NodeMetadata{Team: team})
}

func (c *CreateIssue) Execute(ctx core.ExecutionContext) error {
	spec := CreateIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	input := IssueCreateInput{
		TeamID: spec.Team,
		Title:  spec.Title,
	}
	if spec.Description != "" {
		input.Description = &spec.Description
	}
	input.AssigneeID = spec.AssigneeID
	input.LabelIDs = spec.LabelIDs
	if p := parsePriority(spec.Priority); p != nil {
		input.Priority = p
	}
	input.StateID = spec.StateID
	issue, err := client.IssueCreate(input)
	if err != nil {
		return fmt.Errorf("create issue: %w", err)
	}
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		createIssuePayloadType,
		[]any{issue},
	)
}

func (c *CreateIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateIssue) Actions() []core.Action {
	return nil
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

func parsePriority(v any) *int {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case float64:
		i := int(x)
		return &i
	case int:
		return &x
	case string:
		var i int
		if _, err := fmt.Sscanf(x, "%d", &i); err != nil {
			return nil
		}
		return &i
	default:
		return nil
	}
}
