package linear

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIssueCreated struct{}

type OnIssueCreatedConfiguration struct {
	TeamID   string   `json:"teamId" mapstructure:"teamId"`
	LabelIDs []string `json:"labelIds" mapstructure:"labelIds"`
}

func (t *OnIssueCreated) Name() string {
	return "linear.onIssueCreated"
}

func (t *OnIssueCreated) Label() string {
	return "On Issue Created"
}

func (t *OnIssueCreated) Description() string {
	return "Trigger when a new issue is created in Linear"
}

func (t *OnIssueCreated) Documentation() string {
	return `The On Issue Created trigger starts a workflow when a new issue is created in Linear.

## Use Cases

- **Automated notifications**: Send alerts when issues are created
- **Cross-tool syncing**: Create corresponding issues in other systems
- **Workflow kickoff**: Start approval or review processes for new issues
- **Team coordination**: Notify team members or channels about new work

## Configuration

- **Team**: Filter by specific Linear team (optional, empty = all teams)
- **Labels**: Filter by label IDs (optional, empty = any labels)

## Output

Provides the created issue data including:
- Issue ID, identifier (e.g., ENG-123)
- Title and description
- Team, assignee, priority, state
- Labels and timestamps
- URL to view in Linear`
}

func (t *OnIssueCreated) Icon() string {
	return "linear"
}

func (t *OnIssueCreated) Color() string {
	return "blue"
}

func (t *OnIssueCreated) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "teamId",
			Label:    "Team",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "team",
				},
			},
			Description: "Filter by team (leave empty for all teams)",
		},
		{
			Name:        "labelIds",
			Label:       "Labels",
			Type:        configuration.FieldTypeArray,
			Required:    false,
			Description: "Filter by label IDs (leave empty for any labels)",
		},
	}
}

func (t *OnIssueCreated) OutputPayload() map[string]any {
	return map[string]any{
		"id":          "",
		"identifier":  "",
		"title":       "",
		"description": "",
		"teamId":      "",
		"assigneeId":  "",
		"priority":    0,
		"stateId":     "",
		"labelIds":    []string{},
		"url":         "",
		"createdAt":   "",
	}
}

func (t *OnIssueCreated) Match(ctx core.TriggerContext) (bool, map[string]any, error) {
	webhookData, ok := ctx.Metadata["webhook"].(map[string]interface{})
	if !ok {
		return false, nil, nil
	}

	action, _ := webhookData["action"].(string)
	if action != "create" {
		return false, nil, nil
	}

	resourceType, _ := webhookData["type"].(string)
	if resourceType != "Issue" {
		return false, nil, nil
	}

	data, ok := webhookData["data"].(map[string]interface{})
	if !ok {
		return false, nil, nil
	}

	var config OnIssueCreatedConfiguration
	if ctx.Configuration != nil {
		if teamID, ok := ctx.Configuration["teamId"].(string); ok {
			config.TeamID = teamID
		}
		if labelIDs, ok := ctx.Configuration["labelIds"].([]interface{}); ok {
			for _, id := range labelIDs {
				if strID, ok := id.(string); ok {
					config.LabelIDs = append(config.LabelIDs, strID)
				}
			}
		}
	}

	// Filter by team if specified
	if config.TeamID != "" {
		issueTeamID, _ := data["teamId"].(string)
		if issueTeamID != config.TeamID {
			return false, nil, nil
		}
	}

	// Filter by labels if specified
	if len(config.LabelIDs) > 0 {
		issueLabels, ok := data["labelIds"].([]interface{})
		if !ok {
			return false, nil, nil
		}

		hasMatchingLabel := false
		for _, configLabel := range config.LabelIDs {
			for _, issueLabel := range issueLabels {
				if strLabel, ok := issueLabel.(string); ok && strLabel == configLabel {
					hasMatchingLabel = true
					break
				}
			}
			if hasMatchingLabel {
				break
			}
		}

		if !hasMatchingLabel {
			return false, nil, nil
		}
	}

	output := map[string]any{
		"id":          data["id"],
		"identifier":  data["identifier"],
		"title":       data["title"],
		"description": data["description"],
		"teamId":      data["teamId"],
		"assigneeId":  data["assigneeId"],
		"priority":    data["priority"],
		"stateId":     data["stateId"],
		"labelIds":    data["labelIds"],
		"url":         data["url"],
		"createdAt":   data["createdAt"],
	}

	return true, output, nil
}
