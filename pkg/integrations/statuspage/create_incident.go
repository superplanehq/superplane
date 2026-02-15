package statuspage

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateIncident struct{}

type CreateIncidentSpec struct {
	Name                 string   `json:"name"`
	Status               string   `json:"status"`
	ImpactOverride       string   `json:"impactOverride"`
	Body                 string   `json:"body"`
	ComponentIDs         []string `json:"componentIds"`
	ComponentStatus      string   `json:"componentStatus"`
	DeliverNotifications *bool    `json:"deliverNotifications,omitempty"`
}

func (c *CreateIncident) Name() string {
	return "statuspage.createIncident"
}

func (c *CreateIncident) Label() string {
	return "Create Incident"
}

func (c *CreateIncident) Description() string {
	return "Create a new incident on Statuspage"
}

func (c *CreateIncident) Documentation() string {
	return `The Create Incident component creates a new incident on an Atlassian Statuspage page.

## Use Cases

- **Status communication**: Create incidents to communicate service disruptions
- **Automated incident creation**: Trigger incident creation from monitoring alerts
- **Maintenance windows**: Create scheduled maintenance incidents

## Configuration

- **Name**: The title of the incident (required, supports expressions)
- **Status**: Incident status — investigating, identified, monitoring, or resolved
- **Impact Override**: Override the calculated impact — none, minor, major, or critical
- **Body**: A message describing the incident (optional, supports expressions)
- **Component IDs**: Statuspage components affected by the incident (optional)
- **Component Status**: Status of the affected components (optional)
- **Deliver Notifications**: Whether to send notifications to subscribers (default: true)

## Output

Returns the created incident object including:
- **id**: Incident ID
- **name**: Incident name
- **status**: Current incident status
- **impact**: Calculated impact level
- **created_at**: Incident creation timestamp
- **shortlink**: Public URL for the incident`
}

func (c *CreateIncident) Icon() string {
	return "activity"
}

func (c *CreateIncident) Color() string {
	return "gray"
}

func (c *CreateIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "name",
			Label:       "Incident Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The title of the incident",
		},
		{
			Name:     "status",
			Label:    "Status",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "investigating",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Investigating", Value: "investigating"},
						{Label: "Identified", Value: "identified"},
						{Label: "Monitoring", Value: "monitoring"},
						{Label: "Resolved", Value: "resolved"},
					},
				},
			},
		},
		{
			Name:        "impactOverride",
			Label:       "Impact Override",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Override the calculated impact level",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "None", Value: "none"},
						{Label: "Minor", Value: "minor"},
						{Label: "Major", Value: "major"},
						{Label: "Critical", Value: "critical"},
					},
				},
			},
		},
		{
			Name:        "body",
			Label:       "Message",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "A message describing the incident",
		},
		{
			Name:        "componentIds",
			Label:       "Affected Components",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Statuspage components affected by this incident",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Component ID",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "componentStatus",
			Label:       "Component Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Status of the affected components",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Operational", Value: "operational"},
						{Label: "Degraded Performance", Value: "degraded_performance"},
						{Label: "Partial Outage", Value: "partial_outage"},
						{Label: "Major Outage", Value: "major_outage"},
						{Label: "Under Maintenance", Value: "under_maintenance"},
					},
				},
			},
		},
		{
			Name:        "deliverNotifications",
			Label:       "Deliver Notifications",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Whether to send notifications to subscribers",
		},
	}
}

func (c *CreateIncident) Setup(ctx core.SetupContext) error {
	spec := CreateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}

	if spec.Status == "" {
		return errors.New("status is required")
	}

	return ctx.Metadata.Set(NodeMetadata{})
}

func (c *CreateIncident) Execute(ctx core.ExecutionContext) error {
	spec := CreateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	payload := IncidentPayload{
		Name:                 spec.Name,
		Status:               spec.Status,
		ImpactOverride:       spec.ImpactOverride,
		Body:                 spec.Body,
		ComponentIDs:         spec.ComponentIDs,
		ComponentStatus:      spec.ComponentStatus,
		DeliverNotifications: spec.DeliverNotifications,
	}

	incident, err := client.CreateIncident(payload)
	if err != nil {
		return fmt.Errorf("failed to create incident: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"statuspage.incident",
		[]any{incident},
	)
}

func (c *CreateIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateIncident) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
