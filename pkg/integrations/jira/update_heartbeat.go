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

const UpdateJiraHeartbeatPayloadType = "jira.heartbeat.updated"

type UpdateHeartbeat struct{}

type UpdateHeartbeatSpec struct {
	Team          string  `json:"team" mapstructure:"team"`
	Heartbeat     string  `json:"heartbeat" mapstructure:"heartbeat"`
	Description   *string `json:"description,omitempty" mapstructure:"description"`
	Interval      *int    `json:"interval,omitempty" mapstructure:"interval"`
	IntervalUnit  *string `json:"intervalUnit,omitempty" mapstructure:"intervalUnit"`
	Enabled       *bool   `json:"enabled,omitempty" mapstructure:"enabled"`
	AlertMessage  *string `json:"alertMessage,omitempty" mapstructure:"alertMessage"`
	AlertTags     []any   `json:"alertTags,omitempty" mapstructure:"alertTags"`
	AlertPriority *string `json:"alertPriority,omitempty" mapstructure:"alertPriority"`
}

type UpdateHeartbeatNodeMetadata struct {
	TeamName string `json:"teamName,omitempty"`
}

func (c *UpdateHeartbeat) Name() string {
	return "jira.updateHeartbeat"
}

func (c *UpdateHeartbeat) Label() string {
	return "Update Heartbeat"
}

func (c *UpdateHeartbeat) Description() string {
	return "Update a Jira Service Management Operations heartbeat"
}

func (c *UpdateHeartbeat) Documentation() string {
	return `The Update Heartbeat component changes settings on an existing JSM Operations heartbeat.

## Use Cases

- **Tune intervals**: Adjust expected ping frequency after operational changes
- **Enable/disable monitoring**: Turn heartbeats on or off from automation
- **Alert tuning**: Update expiration alert message, tags, or priority

## Configuration

- **Team** and **Heartbeat** (required): Identify the heartbeat to update
- **Optional fields** use toggles: only enabled fields are sent to the Jira API
- Heartbeat **name** cannot be changed (create a new heartbeat instead)

## Output

Returns the updated heartbeat object from the JSM Operations API.

## Notes

- Enable at least one optional field toggle before saving or running the node.
- Requires Jira Service Management Operations with heartbeats enabled.`
}

func (c *UpdateHeartbeat) Icon() string {
	return "jira"
}

func (c *UpdateHeartbeat) Color() string {
	return "blue"
}

func (c *UpdateHeartbeat) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateHeartbeat) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "team",
			Label:       "Team",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "JSM Operations team that owns the heartbeat",
			Placeholder: "Select a team",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "team",
				},
			},
		},
		{
			Name:        "heartbeat",
			Label:       "Heartbeat",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Heartbeat to update (listed from the selected team)",
			Placeholder: "Select a heartbeat",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "heartbeat",
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
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Description: "New description for the heartbeat",
		},
		{
			Name:        "interval",
			Label:       "Interval",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Default:     5,
			Description: "New expected ping interval (minimum 1)",
		},
		{
			Name:        "intervalUnit",
			Label:       "Interval unit",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "minutes",
			Description: "Unit for the expected ping interval",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "interval", Values: []string{"*"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Minutes", Value: "minutes"},
						{Label: "Hours", Value: "hours"},
						{Label: "Days", Value: "days"},
					},
				},
			},
		},
		{
			Name:        "enabled",
			Label:       "Enabled",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Togglable:   true,
			Default:     true,
			Description: "Enable or disable heartbeat monitoring",
		},
		{
			Name:        "alertMessage",
			Label:       "Alert message",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "New expiration alert message",
		},
		{
			Name:        "alertTags",
			Label:       "Alert tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Replacement tags for the expiration alert",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "alertPriority",
			Label:       "Alert priority",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Togglable:   true,
			Default:     "P3",
			Description: "Priority for the expiration alert",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "P1", Value: "P1"},
						{Label: "P2", Value: "P2"},
						{Label: "P3", Value: "P3"},
						{Label: "P4", Value: "P4"},
						{Label: "P5", Value: "P5"},
					},
				},
			},
		},
	}
}

func (c *UpdateHeartbeat) Setup(ctx core.SetupContext) error {
	spec := UpdateHeartbeatSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	cloudID, err := resolveCloudID(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	if strings.TrimSpace(spec.Team) == "" {
		return fmt.Errorf("team is required")
	}
	if strings.TrimSpace(spec.Heartbeat) == "" {
		return fmt.Errorf("heartbeat is required")
	}
	if spec.Interval != nil && *spec.Interval < 1 {
		return fmt.Errorf("interval must be at least 1")
	}
	if !hasEffectiveHeartbeatUpdate(spec) {
		return fmt.Errorf("at least one update field must be enabled")
	}

	return ctx.Metadata.Set(UpdateHeartbeatNodeMetadata{TeamName: resolveOpsTeamName(ctx, cloudID, spec.Team)})
}

func (c *UpdateHeartbeat) Execute(ctx core.ExecutionContext) error {
	spec := UpdateHeartbeatSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	cloudID, err := resolveCloudID(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	teamID := strings.TrimSpace(spec.Team)
	name := strings.TrimSpace(spec.Heartbeat)
	if teamID == "" {
		return fmt.Errorf("team is required")
	}
	if name == "" {
		return fmt.Errorf("heartbeat is required")
	}

	req := updateHeartbeatRequestFromSpec(spec)
	if updateHeartbeatRequestEmpty(req) {
		return fmt.Errorf("at least one update field must be enabled")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	resp, err := client.UpdateHeartbeat(cloudID, teamID, name, req)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		UpdateJiraHeartbeatPayloadType,
		[]any{resp},
	)
}

func hasEffectiveHeartbeatUpdate(spec UpdateHeartbeatSpec) bool {
	return !updateHeartbeatRequestEmpty(updateHeartbeatRequestFromSpec(spec))
}

func updateHeartbeatRequestFromSpec(spec UpdateHeartbeatSpec) *UpdateHeartbeatRequest {
	req := &UpdateHeartbeatRequest{}

	if spec.Description != nil {
		req.Description = strings.TrimSpace(*spec.Description)
	}
	if spec.Interval != nil && *spec.Interval >= 1 {
		interval := *spec.Interval
		req.Interval = &interval
		if spec.IntervalUnit != nil {
			if unit := strings.TrimSpace(*spec.IntervalUnit); unit != "" {
				req.IntervalUnit = unit
			}
		}
	}
	if spec.Enabled != nil {
		enabled := *spec.Enabled
		req.Enabled = &enabled
	}
	if spec.AlertMessage != nil {
		req.AlertMessage = strings.TrimSpace(*spec.AlertMessage)
	}
	if spec.AlertTags != nil {
		req.AlertTags = heartbeatAlertTagsFromList(spec.AlertTags)
	}
	if spec.AlertPriority != nil {
		if p := heartbeatAlertPriorityForAPI(*spec.AlertPriority); p != "" {
			req.AlertPriority = p
		}
	}

	return req
}

func updateHeartbeatRequestEmpty(req *UpdateHeartbeatRequest) bool {
	if req == nil {
		return true
	}
	if req.Description != "" {
		return false
	}
	if req.Interval != nil {
		return false
	}
	if req.Enabled != nil {
		return false
	}
	if req.AlertMessage != "" {
		return false
	}
	if len(req.AlertTags) > 0 {
		return false
	}
	if strings.TrimSpace(req.AlertPriority) != "" {
		return false
	}
	return true
}

func (c *UpdateHeartbeat) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateHeartbeat) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateHeartbeat) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateHeartbeat) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateHeartbeat) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateHeartbeat) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
