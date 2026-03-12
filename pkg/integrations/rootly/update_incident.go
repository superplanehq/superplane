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
	IncidentID string       `json:"incidentId"`
	Title      string       `json:"title"`
	Summary    string       `json:"summary"`
	Status     string       `json:"status"`
	SubStatus  string       `json:"subStatus"`
	Severity   string       `json:"severity"`
	Services   []string     `json:"services"`
	Teams      []string     `json:"teams"`
	Labels     []LabelEntry `json:"labels"`
}

type LabelEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (c *UpdateIncident) Name() string {
	return "rootly.updateIncident"
}

func (c *UpdateIncident) Label() string {
	return "Update Incident"
}

func (c *UpdateIncident) Description() string {
	return "Update an existing incident in Rootly"
}

func (c *UpdateIncident) Documentation() string {
	return `The Update Incident component updates an existing incident in Rootly.

## Use Cases

- **Status updates**: Update incident status when new information arrives
- **Severity changes**: Adjust severity based on impact assessment
- **Service association**: Attach affected services to an incident
- **Team assignment**: Assign teams to respond to an incident
- **Metadata updates**: Add labels to categorize incidents

## Configuration

- **Incident ID**: The UUID of the incident to update (required, supports expressions)
- **Title**: Update the incident title (optional, supports expressions)
- **Summary**: Update the incident summary (optional, supports expressions)
- **Status**: Update the incident status (optional)
- **Sub-Status**: Update the incident sub-status (optional, required by some Rootly accounts when changing status)
- **Severity**: Update the incident severity level (optional)
- **Services**: Services to attach to the incident (optional)
- **Teams**: Teams to attach to the incident (optional)
- **Labels**: Key-value labels for the incident (optional)

## Output

Returns the updated incident object including:
- **id**: Incident UUID
- **sequential_id**: Sequential incident number
- **title**: Incident title
- **slug**: URL-friendly slug
- **status**: Current incident status
- **updated_at**: Last update timestamp`
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
			Placeholder: "e.g., abc123-def456",
			Description: "The UUID of the incident to update",
		},
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Update the incident title",
		},
		{
			Name:        "summary",
			Label:       "Summary",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Update the incident summary",
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
			Name:        "subStatus",
			Label:       "Sub-Status",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Update the incident sub-status (required by some accounts when changing status)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "sub_status",
				},
			},
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Update the incident severity",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "severity",
				},
			},
		},
		{
			Name:        "services",
			Label:       "Services",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Services to attach to the incident",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "service",
					Multi: true,
				},
			},
		},
		{
			Name:        "teams",
			Label:       "Teams",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Teams to attach to the incident",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "team",
					Multi: true,
				},
			},
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Key-value labels for the incident",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:               "key",
								Label:              "Key",
								Type:               configuration.FieldTypeString,
								Required:           true,
								DisallowExpression: true,
								Description:        "Label key",
							},
							{
								Name:        "value",
								Label:       "Value",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "Label value",
							},
						},
					},
				},
			},
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

	hasUpdate := spec.Title != "" || spec.Summary != "" ||
		spec.Status != "" || spec.SubStatus != "" || spec.Severity != "" ||
		len(spec.Services) > 0 || len(spec.Teams) > 0 ||
		len(spec.Labels) > 0

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

	attrs := UpdateIncidentAttributes{
		Title:       spec.Title,
		Summary:     spec.Summary,
		Status:      spec.Status,
		SubStatusID: spec.SubStatus,
		SeverityID:  spec.Severity,
	}

	// Only set list/map fields when non-empty so that empty arrays from the frontend
	// do not clear existing services, teams, or labels in Rootly (omitempty omits nil
	// but not empty slices in JSON).
	if len(spec.Services) > 0 {
		attrs.ServiceIDs = spec.Services
	}
	if len(spec.Teams) > 0 {
		attrs.GroupIDs = spec.Teams
	}
	if len(spec.Labels) > 0 {
		labels := make(map[string]string, len(spec.Labels))
		for _, l := range spec.Labels {
			labels[l.Key] = l.Value
		}
		attrs.Labels = labels
	}

	incident, err := client.UpdateIncident(spec.IncidentID, attrs)
	if err != nil {
		return fmt.Errorf("failed to update incident: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"rootly.incident",
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
