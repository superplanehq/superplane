package statuspage

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateIncident struct{}

// UpdateIncidentSpec is the strongly typed configuration for the Update Incident component.
type UpdateIncidentSpec struct {
	Page                 string   `json:"page"`
	Incident             string   `json:"incident"`
	Status               string   `json:"status"`
	Body                 string   `json:"body"`
	ImpactOverride       string   `json:"impactOverride"`
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
	return "Update the status and message of an existing incident on your Statuspage."
}

func (c *UpdateIncident) Documentation() string {
	return `The Update Incident component updates an existing incident on your Atlassian Statuspage.

## Use Cases

- **Status transitions**: Update incident status (e.g. investigating → identified → resolved)
- **Maintenance updates**: Transition scheduled maintenance to in progress or completed
- **Integration workflows**: Update incidents from monitoring systems or approval workflows

## Configuration

- **Page** (required): The Statuspage containing the incident. Select from the dropdown, or switch to expression mode for workflow chaining (e.g. {{ $['Create Incident'].data.page_id }}).
- **Incident** (required): Incident ID to update. Supports expressions for workflow chaining (e.g. {{ $['Create Incident'].data.id }}).
- **Status** (optional): New status (investigating, identified, monitoring, resolved, scheduled, in_progress, completed)
- **Body** (optional): Update message shown as the latest incident update
- **Impact override** (optional): Override displayed severity (none, maintenance, minor, major, critical)
- **Components** (optional): Components to associate with this update
- **Component status** (optional): Status to set for selected components
- **Deliver notifications** (optional): Whether to send notifications for this update (default: true)

At least one of Status, Body, Impact override, or Components must be provided.

## Output

Returns the full Statuspage Incident object from the API. The payload has structure { type, timestamp, data } where data is the incident. Common expression paths (use $['Node Name'].data. as prefix):
- data.id, data.name, data.status, data.impact
- data.shortlink — link to the incident
- data.created_at, data.updated_at
- data.components — array of affected components
- data.incident_updates — array of update messages`
}

func (c *UpdateIncident) Icon() string {
	return "activity"
}

func (c *UpdateIncident) Color() string {
	return "gray"
}

func (c *UpdateIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIncident) ExampleOutput() map[string]any {
	return map[string]any{
		"id":         "p31zjtct2jer",
		"name":       "Database Connection Issues",
		"status":     "resolved",
		"impact":     "major",
		"shortlink":  "http://stspg.io/p31zjtct2jer",
		"created_at": "2026-02-12T10:30:00.000Z",
		"updated_at": "2026-02-12T11:00:00.000Z",
		"page_id":    "kctbh9vrtdwd",
		"component_ids": []string{"8kbf7d35c070"},
		"incident_updates": []map[string]any{
			{
				"id":         "upd1",
				"status":     "resolved",
				"body":       "All systems operational. Issue resolved.",
				"created_at": "2026-02-12T11:00:00.000Z",
			},
		},
	}
}

func (c *UpdateIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "page",
			Label:       "Page",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Statuspage containing the incident",
			Placeholder: "Select a page",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypePage,
				},
			},
		},
		{
			Name:        "incident",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Incident ID to update (supports expressions)",
			Placeholder: "e.g., p31zjtct2jer or {{ $['Create Incident'].data.id }}",
		},
		{
			Name:        "status",
			Label:       "Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "New incident status",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Investigating", Value: "investigating"},
						{Label: "Identified", Value: "identified"},
						{Label: "Monitoring", Value: "monitoring"},
						{Label: "Resolved", Value: "resolved"},
						{Label: "Scheduled", Value: "scheduled"},
						{Label: "In progress", Value: "in_progress"},
						{Label: "Completed", Value: "completed"},
					},
				},
			},
		},
		{
			Name:        "body",
			Label:       "Message",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Update message shown as the latest incident update",
		},
		{
			Name:        "impactOverride",
			Label:       "Impact override",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Override the displayed severity for this incident",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Don't override", Value: "__none__"},
						{Label: "None", Value: "none"},
						{Label: "Maintenance", Value: "maintenance"},
						{Label: "Minor", Value: "minor"},
						{Label: "Major", Value: "major"},
						{Label: "Critical", Value: "critical"},
					},
				},
			},
		},
		{
			Name:        "componentIds",
			Label:       "Components",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Components to update in this incident",
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
			Name:        "componentStatus",
			Label:       "Component status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
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
			Name:        "deliverNotifications",
			Label:       "Deliver notifications",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Send notifications for this update (default: true)",
		},
	}
}

func (c *UpdateIncident) Setup(ctx core.SetupContext) error {
	spec := UpdateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	if spec.Page == "" {
		return errors.New("page is required")
	}

	if spec.Incident == "" {
		return errors.New("incident is required")
	}

	effectiveImpact := spec.ImpactOverride
	if effectiveImpact == "__none__" {
		effectiveImpact = ""
	}
	hasUpdate := spec.Status != "" || spec.Body != "" || effectiveImpact != "" || len(spec.ComponentIDs) > 0
	if !hasUpdate {
		return errors.New("at least one of status, body, impact override, or components must be provided")
	}

	// Resolve page name for metadata when Page is a static ID (no expression).
	// Skip API call if HTTP context is not available (e.g. in tests without HTTP mock).
	metadata := NodeMetadata{}
	if spec.Page != "" && !strings.Contains(spec.Page, "{{") && ctx.HTTP != nil {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err == nil {
			pages, err := client.ListPages()
			if err == nil {
				for _, p := range pages {
					if p.ID == spec.Page {
						metadata.PageName = p.Name
						break
					}
				}
			}
		}
	}
	return ctx.Metadata.Set(metadata)
}

func (c *UpdateIncident) Execute(ctx core.ExecutionContext) error {
	spec := UpdateIncidentSpec{}
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

	impactOverride := spec.ImpactOverride
	if impactOverride == "__none__" {
		impactOverride = ""
	}

	req := UpdateIncidentRequest{
		Status:               spec.Status,
		Body:                 spec.Body,
		ImpactOverride:       impactOverride,
		ComponentIDs:         spec.ComponentIDs,
		Components:           components,
		DeliverNotifications: spec.DeliverNotifications,
	}

	incident, err := client.UpdateIncident(spec.Page, spec.Incident, req)
	if err != nil {
		return fmt.Errorf("failed to update incident: %w", err)
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
