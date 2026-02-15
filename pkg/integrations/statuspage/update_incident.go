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

type UpdateIncident struct{}

type UpdateIncidentSpec struct {
	IncidentID           string   `json:"incidentId"`
	Name                 string   `json:"name"`
	Status               string   `json:"status"`
	ImpactOverride       string   `json:"impactOverride"`
	Body                 string   `json:"body"`
	ComponentIDs         []string `json:"componentIds"`
	ComponentStatus      string   `json:"componentStatus"`
	DeliverNotifications *bool    `json:"deliverNotifications,omitempty"`
}

func (c *UpdateIncident) Name() string {
	return "statuspage.updateIncident"
}

func (c *UpdateIncident) Label() string {
	return "Update Incident"
}

func (c *UpdateIncident) Description() string {
	return "Update an existing incident on Statuspage"
}

func (c *UpdateIncident) Documentation() string {
	return `The Update Incident component modifies an existing incident on an Atlassian Statuspage page.

## Use Cases

- **Status progression**: Move incidents through investigating → identified → monitoring → resolved
- **Impact updates**: Adjust incident impact as more information becomes available
- **Component updates**: Update the status of affected components
- **Resolution**: Resolve incidents when issues are fixed

## Configuration

- **Incident ID**: The ID of the incident to update (required, supports expressions)
- **Name**: Update the incident title (optional, supports expressions)
- **Status**: Update incident status — investigating, identified, monitoring, or resolved
- **Impact Override**: Override the calculated impact — none, minor, major, or critical
- **Message**: A status update message (optional, supports expressions)
- **Component IDs**: Update the affected components (optional)
- **Component Status**: Update the status of affected components (optional)
- **Deliver Notifications**: Whether to send notifications to subscribers

## Output

Returns the updated incident object with all current information.`
}

func (c *UpdateIncident) Icon() string {
	return "edit"
}

func (c *UpdateIncident) Color() string {
	return "gray"
}

func (c *UpdateIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the incident to update",
			Placeholder: "e.g., abc123def456",
		},
		{
			Name:        "name",
			Label:       "Incident Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Update the incident title",
		},
		{
			Name:        "status",
			Label:       "Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Update the incident status",
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
			Description: "A status update message",
		},
		{
			Name:        "componentIds",
			Label:       "Affected Components",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Update the affected components",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Component ID",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeIntegrationResource,
						TypeOptions: &configuration.TypeOptions{
							Resource: &configuration.ResourceTypeOptions{
								Type: "component",
							},
						},
					},
				},
			},
		},
		{
			Name:        "componentStatus",
			Label:       "Component Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Update the status of affected components",
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

func (c *UpdateIncident) Setup(ctx core.SetupContext) error {
	spec := UpdateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incidentId is required")
	}

	// Validate that at least one field to update is provided
	hasUpdate := spec.Name != "" || spec.Status != "" ||
		spec.ImpactOverride != "" || spec.Body != "" ||
		len(spec.ComponentIDs) > 0 || spec.ComponentStatus != ""

	if !hasUpdate {
		return errors.New("at least one field to update must be provided")
	}

	return ctx.Metadata.Set(NodeMetadata{})
}

func (c *UpdateIncident) Execute(ctx core.ExecutionContext) error {
	spec := UpdateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	payload := UpdateIncidentPayload{
		Name:                 spec.Name,
		Status:               spec.Status,
		ImpactOverride:       spec.ImpactOverride,
		Body:                 spec.Body,
		ComponentIDs:         spec.ComponentIDs,
		ComponentStatus:      spec.ComponentStatus,
		DeliverNotifications: spec.DeliverNotifications,
	}

	incident, err := client.UpdateIncident(spec.IncidentID, payload)
	if err != nil {
		return fmt.Errorf("failed to update incident: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"statuspage.incident",
		[]any{incident},
	)
}

func (c *UpdateIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIncident) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *UpdateIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
