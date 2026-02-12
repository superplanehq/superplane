package linear

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateIssue struct{}

type CreateIssueConfiguration struct {
	TeamID      string   `json:"teamId" mapstructure:"teamId"`
	Title       string   `json:"title" mapstructure:"title"`
	Description string   `json:"description" mapstructure:"description"`
	AssigneeID  string   `json:"assigneeId" mapstructure:"assigneeId"`
	Priority    int      `json:"priority" mapstructure:"priority"`
	StateID     string   `json:"stateId" mapstructure:"stateId"`
	LabelIDs    []string `json:"labelIds" mapstructure:"labelIds"`
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

- **Automated issue creation**: Create Linear issues from other tool events
- **Cross-tool syncing**: Mirror issues from other systems into Linear
- **Workflow automation**: Generate tasks based on triggers
- **Incident tracking**: Automatically create issues for alerts or incidents

## Configuration

- **Team**: The Linear team where the issue will be created (required)
- **Title**: Issue title (required)
- **Description**: Issue description in markdown (optional)
- **Assignee**: User to assign the issue to (optional)
- **Priority**: Issue priority (0=None, 1=Urgent, 2=High, 3=Medium, 4=Low)
- **State**: Initial workflow state (optional)
- **Labels**: List of label IDs to apply (optional)

## Output

Returns the created issue including:
- Issue ID and identifier (e.g., ENG-123)
- Title and description
- URL to view the issue in Linear`
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
			Name:     "teamId",
			Label:    "Team",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "team",
				},
			},
			Description: "Linear team where the issue will be created",
		},
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Issue title",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Issue description (supports markdown)",
		},
		{
			Name:        "assigneeId",
			Label:       "Assignee",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "User ID to assign the issue to",
		},
		{
			Name:        "priority",
			Label:       "Priority",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Issue priority (0=None, 1=Urgent, 2=High, 3=Medium, 4=Low)",
		},
		{
			Name:        "stateId",
			Label:       "State",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Workflow state ID",
		},
		{
			Name:        "labelIds",
			Label:       "Labels",
			Type:        configuration.FieldTypeArray,
			Required:    false,
			Description: "List of label IDs",
		},
	}
}

func (c *CreateIssue) Execute(ctx core.ExecutionContext) ([]core.OutputChannel, error) {
	var config CreateIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration(), &config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	if config.TeamID == "" {
		return nil, fmt.Errorf("teamId is required")
	}

	if config.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	client := NewClient(ctx.SyncContext().HTTP, ctx.SyncContext().Integration)

	input := IssueInput{
		TeamID:      config.TeamID,
		Title:       config.Title,
		Description: config.Description,
		AssigneeID:  config.AssigneeID,
		Priority:    config.Priority,
		StateID:     config.StateID,
		LabelIDs:    config.LabelIDs,
	}

	issue, err := client.CreateIssue(input)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	return []core.OutputChannel{
		{
			Name: core.DefaultOutputChannel.Name,
			Output: map[string]any{
				"id":          issue.ID,
				"identifier":  issue.Identifier,
				"title":       issue.Title,
				"description": issue.Description,
				"url":         issue.URL,
			},
		},
	}, nil
}
