package rootly

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
	IncidentID string   `json:"incidentId"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	Status     string   `json:"status"`
	Severity   string   `json:"severity"`
	Services   []string `json:"services"`
	Teams      []string `json:"teams"`
	Labels     []Label  `json:"labels"`
}

type Label struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (u *UpdateIncident) Name() string {
	return "rootly.updateIncident"
}

func (u *UpdateIncident) Label() string {
	return "Update Incident"
}

func (u *UpdateIncident) Description() string {
	return "Update an existing incident in Rootly"
}

func (u *UpdateIncident) Documentation() string {
	return `The Update Incident component updates an existing incident in Rootly.

## Use Cases

- **Status sync**: Update incident status when new information arrives from external systems
- **Severity escalation**: Change incident severity based on workflow conditions
- **Service attachment**: Attach services or teams to an incident from workflow steps
- **Summary updates**: Add or modify incident summary with new details

## Configuration

- **Incident ID**: The Rootly incident UUID to update (required, supports expressions)
- **Title**: New incident title (optional, supports expressions)
- **Summary**: New summary/description (optional, supports expressions)
- **Status**: New status: in_triage, started, detected, acknowledged, mitigated, resolved, closed, cancelled (optional)
- **Severity**: Rootly severity slug (optional)
- **Services**: Service names to attach (optional)
- **Teams**: Team/group names to attach (optional)
- **Labels**: Key-value labels to apply (optional)

## Output

Returns the updated incident object including:
- **id**: Incident ID
- **sequential_id**: Sequential incident number
- **title**: Incident title
- **slug**: Incident slug
- **status**: Current incident status
- **updated_at**: Last update timestamp`
}

func (u *UpdateIncident) Icon() string {
	return "alert-triangle"
}

func (u *UpdateIncident) Color() string {
	return "gray"
}

func (u *UpdateIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (u *UpdateIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The Rootly incident UUID to update",
		},
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "New incident title",
		},
		{
			Name:        "summary",
			Label:       "Summary",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "New summary/description",
		},
		{
			Name:        "status",
			Label:       "Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "New incident status",
			Placeholder: "Select a status",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "In Triage", Value: "in_triage"},
						{Label: "Started", Value: "started"},
						{Label: "Detected", Value: "detected"},
						{Label: "Acknowledged", Value: "acknowledged"},
						{Label: "Mitigated", Value: "mitigated"},
						{Label: "Resolved", Value: "resolved"},
						{Label: "Closed", Value: "closed"},
						{Label: "Cancelled", Value: "cancelled"},
					},
				},
			},
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "The severity level of the incident",
			Placeholder: "Select a severity",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "severity",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "services",
			Label:       "Services",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Services to attach to the incident",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Service",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "teams",
			Label:       "Teams",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Teams to attach to the incident",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Team",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Key-value labels to apply (e.g., platform: backend-api)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:     "key",
								Label:    "Key",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
							{
								Name:     "value",
								Label:    "Value",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
	}
}

func (u *UpdateIncident) Setup(ctx core.SetupContext) error {
	spec := UpdateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incident ID is required")
	}

	return nil
}

func (u *UpdateIncident) Execute(ctx core.ExecutionContext) error {
	spec := UpdateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Convert labels to map
	labels := make(map[string]string)
	for _, label := range spec.Labels {
		if label.Key != "" {
			labels[label.Key] = label.Value
		}
	}

	req := UpdateIncidentRequest{
		Title:    spec.Title,
		Summary:  spec.Summary,
		Status:   spec.Status,
		Severity: spec.Severity,
		Services: spec.Services,
		Teams:    spec.Teams,
		Labels:   labels,
	}

	incident, err := client.UpdateIncident(spec.IncidentID, req)
	if err != nil {
		return fmt.Errorf("failed to update incident: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"rootly.incident",
		[]any{incident},
	)
}

func (u *UpdateIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (u *UpdateIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (u *UpdateIncident) Actions() []core.Action {
	return []core.Action{}
}

func (u *UpdateIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (u *UpdateIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (u *UpdateIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
