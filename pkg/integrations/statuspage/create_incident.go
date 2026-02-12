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

// CreateIncidentSpec is the strongly typed configuration for the Create Incident component.
// Conditionally visible fields use pointers so they can be nil when not shown.
type CreateIncidentSpec struct {
	Page                    string   `mapstructure:"page"`
	IncidentType            string   `mapstructure:"incidentType"`
	Name                    string   `mapstructure:"name"`
	Body                    string   `mapstructure:"body"`
	Status                  string   `mapstructure:"status"`
	ImpactOverride          string   `mapstructure:"impactOverride"`
	ComponentIDs            []string `mapstructure:"componentIds"`
	ComponentStatus         string   `mapstructure:"componentStatus"`
	ScheduledFor            string   `mapstructure:"scheduledFor"`
	ScheduledUntil          string   `mapstructure:"scheduledUntil"`
	ScheduledRemindPrior    bool     `mapstructure:"scheduledRemindPrior"`
	ScheduledAutoInProgress bool     `mapstructure:"scheduledAutoInProgress"`
	ScheduledAutoCompleted  bool     `mapstructure:"scheduledAutoCompleted"`
	DeliverNotifications    *bool    `mapstructure:"deliverNotifications"`
}

func (c *CreateIncident) Name() string {
	return "statuspage.createIncident"
}

func (c *CreateIncident) Label() string {
	return "Create Incident"
}

func (c *CreateIncident) Description() string {
	return "Create a new incident or scheduled maintenance on your Statuspage."
}

func (c *CreateIncident) Documentation() string {
	return `The Create Incident component creates a new realtime or scheduled incident on your Atlassian Statuspage.

## Use Cases

- **Realtime incidents**: Create and notify subscribers when an unexpected outage occurs
- **Scheduled maintenance**: Schedule maintenance windows with optional reminders and auto-transitions
- **Integration workflows**: Create incidents from monitoring alerts or other workflow events

## Configuration

- **Page** (required): The Statuspage to create the incident on
- **Incident type**: Realtime (active incident) or Scheduled (planned maintenance)
- **Name** (required): Short title for the incident
- **Body** (optional): Initial message shown as the first incident update
- **Status** (realtime): investigating, identified, monitoring, or resolved
- **Impact override** (realtime): none, minor, major, or critical
- **Components** (optional): Select one or more components to associate with the incident
- **Component status** (optional): Status to set for all selected components (e.g. degraded_performance, under_maintenance)
- **Scheduled For / Until** (scheduled): Start and end time for scheduled maintenance
- **Scheduled options** (scheduled): Remind prior, auto in-progress, auto completed
- **Deliver notifications** (optional): Whether to send notifications for the initial update (default: true)

## Output

Returns the full Statuspage Incident object from the API. Common expression paths:
- incident.id, incident.name, incident.status, incident.impact
- incident.shortlink — link to the incident
- incident.created_at, incident.updated_at
- incident.components — array of affected components
- incident.incident_updates — array of update messages`
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

func (c *CreateIncident) ExampleOutput() map[string]any {
	return map[string]any{
		"id":                  "p31zjtct2jer",
		"name":                "Database Connection Issues",
		"status":              "investigating",
		"impact":              "major",
		"shortlink":           "http://stspg.io/p31zjtct2jer",
		"created_at":          "2026-02-12T10:30:00.000Z",
		"page_id":             "kctbh9vrtdwd",
		"affected_components": []string{"API"},
		"component_count":     1,
		"latest_update":       "We are investigating reports of slow database queries.",
	}
}

func (c *CreateIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "page",
			Label:       "Page",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Statuspage to create the incident on",
			Placeholder: "Select a page",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypePage,
				},
			},
		},
		{
			Name:     "incidentType",
			Label:    "Incident type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "realtime",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Realtime", Value: "realtime"},
						{Label: "Scheduled", Value: "scheduled"},
					},
				},
			},
		},
		{
			Name:        "name",
			Label:       "Incident name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Short title for the incident",
		},
		{
			Name:        "body",
			Label:       "Initial message",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "First incident update message (optional)",
		},
		{
			Name:     "status",
			Label:    "Status",
			Type:     configuration.FieldTypeSelect,
			Required: false,
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
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"realtime"}},
			},
		},
		{
			Name:     "impactOverride",
			Label:    "Impact override",
			Type:     configuration.FieldTypeSelect,
			Required: false,
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
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"realtime"}},
			},
		},
		{
			Name:        "componentIds",
			Label:       "Components",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Components to associate with this incident",
			Placeholder: "Select components",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeComponent,
					Multi: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "page_id",
							ValueFrom: &configuration.ParameterValueFrom{Field: "page"},
						},
					},
				},
			},
		},
		{
			Name:     "componentStatus",
			Label:    "Component status",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Description: "Status to set for all selected components",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Operational", Value: "operational"},
						{Label: "Degraded performance", Value: "degraded_performance"},
						{Label: "Partial outage", Value: "partial_outage"},
						{Label: "Major outage", Value: "major_outage"},
						{Label: "Under maintenance", Value: "under_maintenance"},
					},
				},
			},
		},
		{
			Name:        "scheduledFor",
			Label:       "Scheduled for",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "When the scheduled maintenance starts (ISO 8601)",
			TypeOptions: &configuration.TypeOptions{
				DateTime: &configuration.DateTimeTypeOptions{
					Format: "2006-01-02T15:04",
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"scheduled"}},
			},
		},
		{
			Name:        "scheduledUntil",
			Label:       "Scheduled until",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "When the scheduled maintenance ends (ISO 8601)",
			TypeOptions: &configuration.TypeOptions{
				DateTime: &configuration.DateTimeTypeOptions{
					Format: "2006-01-02T15:04",
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"scheduled"}},
			},
		},
		{
			Name:        "scheduledRemindPrior",
			Label:       "Remind subscribers 60 minutes before",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"scheduled"}},
			},
		},
		{
			Name:        "scheduledAutoInProgress",
			Label:       "Auto transition to In Progress at start",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"scheduled"}},
			},
		},
		{
			Name:        "scheduledAutoCompleted",
			Label:       "Auto transition to Completed at end",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"scheduled"}},
			},
		},
		{
			Name:        "deliverNotifications",
			Label:       "Deliver notifications",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Send notifications for the initial incident update (default: true)",
		},
	}
}

func (c *CreateIncident) Setup(ctx core.SetupContext) error {
	spec := CreateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	if spec.Page == "" {
		return errors.New("page is required")
	}

	if spec.IncidentType == "" {
		return errors.New("incidentType is required")
	}

	if spec.IncidentType != "realtime" && spec.IncidentType != "scheduled" {
		return fmt.Errorf("incidentType must be realtime or scheduled, got %q", spec.IncidentType)
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}

	return nil
}

func (c *CreateIncident) Execute(ctx core.ExecutionContext) error {
	spec := CreateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	components := make(map[string]string)
	for _, id := range spec.ComponentIDs {
		if spec.ComponentStatus != "" {
			components[id] = spec.ComponentStatus
		} else {
			components[id] = "operational"
		}
	}

	req := CreateIncidentRequest{
		Name:             spec.Name,
		Body:             spec.Body,
		ComponentIDs:     spec.ComponentIDs,
		Components:       components,
		Realtime:         spec.IncidentType == "realtime",
		DeliverNotifications: spec.DeliverNotifications,
	}

	if spec.IncidentType == "realtime" {
		req.Status = spec.Status
		if req.Status == "" {
			req.Status = "investigating"
		}
		req.ImpactOverride = spec.ImpactOverride
	} else {
		req.ScheduledFor = spec.ScheduledFor
		req.ScheduledUntil = spec.ScheduledUntil
		req.ScheduledRemindPrior = spec.ScheduledRemindPrior
		req.ScheduledAutoInProgress = spec.ScheduledAutoInProgress
		req.ScheduledAutoCompleted = spec.ScheduledAutoCompleted
	}

	incident, err := client.CreateIncident(spec.Page, req)
	if err != nil {
		return fmt.Errorf("failed to create incident: %w", err)
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
