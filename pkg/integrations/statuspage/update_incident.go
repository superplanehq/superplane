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

// isScheduledStatus returns true if the status belongs to a scheduled maintenance incident.
func isScheduledStatus(status string) bool {
	switch status {
	case "scheduled", "in_progress", "verifying", "completed":
		return true
	default:
		return false
	}
}

// UpdateIncidentSpec is the strongly typed configuration for the Update Incident component.
type UpdateIncidentSpec struct {
	Page               string `json:"page"`
	Incident           string `json:"incident"`
	IncidentExpression string `json:"incidentExpression"`
	IncidentType       string `json:"incidentType"`
	StatusRealtime     string `json:"statusRealtime"`
	StatusScheduled    string `json:"statusScheduled"`
	Body               string `json:"body"`
	ImpactOverride     string `json:"impactOverride"`
	Components         []struct {
		ComponentID string `json:"componentId"`
		Status      string `json:"status"`
	} `json:"components"`
	DeliverNotifications *bool `json:"deliverNotifications,omitempty"`
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
- **Incident type**: Realtime or Scheduled — determines which status options are shown. You cannot change an incident's type.
- **Status** (optional): New status. Options depend on incident type: realtime (investigating, identified, monitoring, resolved) or scheduled (scheduled, in progress, verifying, completed)
- **Body** (optional): Update message shown as the latest incident update
- **Impact override** (optional, realtime only): Override displayed severity (none, maintenance, minor, major, critical)
- **Components** (optional): List of components and their status. Each item has Component ID (supports expressions) and Status (operational, degraded_performance, partial_outage, major_outage, under_maintenance)
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
			Label:       "Incident",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select an incident or choose 'Use expression' when page is an expression",
			Placeholder: "Select an incident",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeIncident,
					Parameters: []configuration.ParameterRef{
						{Name: "page_id", ValueFrom: &configuration.ParameterValueFrom{Field: "page"}},
					},
				},
			},
		},
		{
			Name:        "incidentExpression",
			Label:       "Incident expression",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Expression for incident ID when using expression for page (e.g. {{ $['Create Incident'].data.id }})",
			Placeholder: "e.g. {{ $['Create Incident'].data.id }}",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incident", Values: []string{IncidentUseExpressionID}},
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
			Name:        "statusRealtime",
			Label:       "Status",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "New incident status (supports expressions)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeIncidentStatusRealtime,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"realtime"}},
			},
		},
		{
			Name:        "statusScheduled",
			Label:       "Status",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "New incident status (supports expressions)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeIncidentStatusScheduled,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"scheduled"}},
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
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Override the displayed severity for this incident (supports expressions)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeImpactUpdate,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"realtime"}},
			},
		},
		{
			Name:        "components",
			Label:       "Components",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Components to update in this incident and their status. Component ID supports expressions.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Component",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "componentId",
								Label:       "Component ID",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Placeholder: "e.g. 8kbf7d35c070 or {{ $['X'].data.id }}",
							},
							{
								Name:     "status",
								Label:    "Status",
								Type:     configuration.FieldTypeSelect,
								Required: false,
								Default:  "operational",
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
						},
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
	if spec.Incident == IncidentUseExpressionID {
		if spec.IncidentExpression == "" {
			return errors.New("incident expression is required when using expression for incident")
		}
	}

	incidentType := spec.IncidentType
	if incidentType == "" {
		incidentType = "realtime"
	}
	if incidentType != "realtime" && incidentType != "scheduled" {
		return fmt.Errorf("incidentType must be realtime or scheduled, got %q", incidentType)
	}

	effectiveImpact := spec.ImpactOverride
	if effectiveImpact == "__none__" {
		effectiveImpact = ""
	}
	if incidentType == "scheduled" {
		effectiveImpact = "" // scheduled incidents don't support impact override
	}
	hasUpdate := spec.StatusRealtime != "" || spec.StatusScheduled != "" || spec.Body != "" || effectiveImpact != "" || len(spec.Components) > 0
	if !hasUpdate {
		return errors.New("at least one of status, body, impact override, or components must be provided")
	}

	// Resolve page name and component names for metadata when IDs are static (no expressions).
	componentIDs := componentIDsForMetadataSetup(ctx.Configuration, func() []string {
		var ids []string
		for _, item := range spec.Components {
			if item.ComponentID != "" {
				ids = append(ids, item.ComponentID)
			}
		}
		return ids
	})
	metadata, err := resolveMetadataSetup(ctx, spec.Page, componentIDs)
	if err != nil {
		return err
	}
	if spec.Incident != "" && spec.Incident != IncidentUseExpressionID && !strings.Contains(spec.Incident, "{{") {
		incidentName, err := resolveIncidentName(ctx, spec.Page, spec.Incident)
		if err != nil {
			return fmt.Errorf("incident not found or inaccessible: %w", err)
		}
		metadata.IncidentName = incidentName
	}
	return ctx.Metadata.Set(metadata)
}

func (c *UpdateIncident) Execute(ctx core.ExecutionContext) error {
	spec := UpdateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	incidentID := spec.Incident
	if incidentID == IncidentUseExpressionID {
		incidentID = spec.IncidentExpression
	}
	if incidentID == "" {
		return fmt.Errorf("incident ID is required")
	}

	incidentType := spec.IncidentType
	if incidentType == "" {
		incidentType = "realtime"
	}
	if incidentType != "realtime" && incidentType != "scheduled" {
		return fmt.Errorf("incidentType must be realtime or scheduled, got %q", incidentType)
	}

	var effectiveStatus string
	if incidentType == "scheduled" {
		effectiveStatus = spec.StatusScheduled
	} else {
		effectiveStatus = spec.StatusRealtime
	}

	impactOverride := spec.ImpactOverride
	if impactOverride == "__none__" {
		impactOverride = ""
	}
	if incidentType == "scheduled" {
		impactOverride = ""
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	if effectiveStatus != "" {
		existing, err := client.GetIncident(spec.Page, incidentID)
		if err != nil {
			return fmt.Errorf("failed to fetch incident for validation: %w", err)
		}
		existingStatus, _ := existing["status"].(string)
		actualIsScheduled := isScheduledStatus(existingStatus)
		requestedIsScheduled := isScheduledStatus(effectiveStatus)
		if !actualIsScheduled && requestedIsScheduled {
			return fmt.Errorf("cannot change a realtime incident to scheduled maintenance; status must be investigating, identified, monitoring, or resolved")
		}
		if actualIsScheduled && !requestedIsScheduled {
			return fmt.Errorf("cannot change a scheduled maintenance incident to realtime; status must be scheduled, in progress, verifying, or completed")
		}
	}

	nameOrIDToStatus := make(map[string]string)
	rawIDs := make([]string, 0, len(spec.Components))
	for _, item := range spec.Components {
		if item.ComponentID == "" {
			continue
		}
		status := item.Status
		if status == "" {
			status = "operational"
		}
		nameOrIDToStatus[item.ComponentID] = status
		rawIDs = append(rawIDs, item.ComponentID)
	}

	var componentIDs []string
	var components map[string]string
	if len(nameOrIDToStatus) > 0 && !containsExpression(rawIDs) {
		var errResolve error
		componentIDs, components, errResolve = resolveComponentNameOrIDs(client, spec.Page, nameOrIDToStatus)
		if errResolve != nil {
			return fmt.Errorf("failed to resolve component IDs: %w", errResolve)
		}
	} else {
		components = nameOrIDToStatus
		componentIDs = make([]string, 0, len(components))
		for id := range components {
			componentIDs = append(componentIDs, id)
		}
	}

	req := UpdateIncidentRequest{
		Status:               effectiveStatus,
		Body:                 spec.Body,
		ImpactOverride:       impactOverride,
		ComponentIDs:         componentIDs,
		Components:           components,
		DeliverNotifications: spec.DeliverNotifications,
	}

	incident, err := client.UpdateIncident(spec.Page, incidentID, req)
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
