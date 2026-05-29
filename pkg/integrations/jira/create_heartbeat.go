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

const CreateJiraHeartbeatPayloadType = "jira.heartbeat.created"

type CreateHeartbeat struct{}

type CreateHeartbeatSpec struct {
	Team          string `json:"team" mapstructure:"team"`
	Name          string `json:"name" mapstructure:"name"`
	Description   string `json:"description,omitempty" mapstructure:"description"`
	Interval      int    `json:"interval" mapstructure:"interval"`
	IntervalUnit  string `json:"intervalUnit" mapstructure:"intervalUnit"`
	Enabled       *bool  `json:"enabled,omitempty" mapstructure:"enabled"`
	AlertMessage  string `json:"alertMessage,omitempty" mapstructure:"alertMessage"`
	AlertTags     []any  `json:"alertTags,omitempty" mapstructure:"alertTags"`
	AlertPriority string `json:"alertPriority,omitempty" mapstructure:"alertPriority"`
}

type CreateHeartbeatNodeMetadata struct {
	TeamName string `json:"teamName,omitempty"`
}

func (c *CreateHeartbeat) Name() string {
	return "jira.createHeartbeat"
}

func (c *CreateHeartbeat) Label() string {
	return "Create Heartbeat"
}

func (c *CreateHeartbeat) Description() string {
	return "Create a Jira Service Management Operations heartbeat monitor"
}

func (c *CreateHeartbeat) Documentation() string {
	return `The Create Heartbeat component registers a new heartbeat in Jira Service Management Operations.

## Use Cases

- **Cron monitoring**: Register an expected ping interval for scheduled jobs or agents
- **Infrastructure checks**: Create heartbeats when provisioning monitored systems
- **Automation setup**: Pair with Ping Heartbeat in workflows that report liveness

## Configuration

- **Team**: JSM Operations team that owns the heartbeat
- **Name** (required): Unique heartbeat name within the team
- **Interval** (required): How often a ping is expected (minimum 1)
- **Interval unit**: minutes, hours, or days
- **Enabled**: Whether monitoring is active (defaults to enabled)
- **Description**, **alert message**, **alert tags**, **alert priority**: Optional expiration alert settings

## Output

Returns the created heartbeat object from the JSM Operations API, including **name**, **interval**, **intervalUnit**, **enabled**, and **status**.

## Notes

- Requires Jira Service Management Operations (Premium) with heartbeats enabled on your site.
- Re-sync the Jira integration so SuperPlane has your Atlassian cloud id.`
}

func (c *CreateHeartbeat) Icon() string {
	return "jira"
}

func (c *CreateHeartbeat) Color() string {
	return "orange"
}

func (c *CreateHeartbeat) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateHeartbeat) Configuration() []configuration.Field {
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
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Unique heartbeat name within the team",
			Placeholder: "DNS Server Checker",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Optional description of what this heartbeat monitors",
		},
		{
			Name:        "interval",
			Label:       "Interval",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Description: "How often a ping is expected (minimum 1)",
			Default:     5,
		},
		{
			Name:        "intervalUnit",
			Label:       "Interval unit",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "minutes",
			Description: "Unit for the expected ping interval",
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
			Description: "Enable heartbeat monitoring",
			Default:     true,
		},
		{
			Name:        "alertMessage",
			Label:       "Alert message",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Message used when the heartbeat expires",
		},
		{
			Name:        "alertTags",
			Label:       "Alert tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Tags applied to the expiration alert",
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
			Default:     "__none__",
			Description: "Priority for the expiration alert (defaults to P3 when not set)",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Default (P3)", Value: "__none__"},
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

func (c *CreateHeartbeat) Setup(ctx core.SetupContext) error {
	spec := CreateHeartbeatSpec{}
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
	if strings.TrimSpace(spec.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if spec.Interval < 1 {
		return fmt.Errorf("interval must be at least 1")
	}
	if strings.TrimSpace(spec.IntervalUnit) == "" {
		return fmt.Errorf("intervalUnit is required")
	}

	return ctx.Metadata.Set(CreateHeartbeatNodeMetadata{TeamName: resolveOpsTeamName(ctx, cloudID, spec.Team)})
}

func (c *CreateHeartbeat) Execute(ctx core.ExecutionContext) error {
	spec := CreateHeartbeatSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	cloudID, err := resolveCloudID(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	teamID := strings.TrimSpace(spec.Team)
	name := strings.TrimSpace(spec.Name)
	if teamID == "" {
		return fmt.Errorf("team is required")
	}
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if spec.Interval < 1 {
		return fmt.Errorf("interval must be at least 1")
	}

	enabled := createHeartbeatEnabledFromSpec(spec)
	req := &CreateHeartbeatRequest{
		Name:         name,
		Description:  strings.TrimSpace(spec.Description),
		Interval:     spec.Interval,
		IntervalUnit: strings.TrimSpace(spec.IntervalUnit),
		Enabled:      &enabled,
		AlertMessage: strings.TrimSpace(spec.AlertMessage),
		AlertTags:    heartbeatAlertTagsFromList(spec.AlertTags),
	}
	if p := heartbeatAlertPriorityForAPI(spec.AlertPriority); p != "" {
		req.AlertPriority = p
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	resp, err := client.CreateHeartbeat(cloudID, teamID, req)
	if err != nil {
		return fmt.Errorf("failed to create heartbeat: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		CreateJiraHeartbeatPayloadType,
		[]any{resp},
	)
}

func (c *CreateHeartbeat) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateHeartbeat) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateHeartbeat) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateHeartbeat) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateHeartbeat) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateHeartbeat) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func createHeartbeatEnabledFromSpec(spec CreateHeartbeatSpec) bool {
	if spec.Enabled == nil {
		return true
	}
	return *spec.Enabled
}
