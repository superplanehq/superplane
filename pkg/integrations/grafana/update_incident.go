package grafana

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateIncident struct{}

type UpdateIncidentSpec struct {
	Incident string   `json:"incident" mapstructure:"incident"`
	Title    *string  `json:"title" mapstructure:"title"`
	Severity *string  `json:"severity" mapstructure:"severity"`
	Labels   []string `json:"labels" mapstructure:"labels"`
	IsDrill  *bool    `json:"isDrill" mapstructure:"isDrill"`
}

func (u *UpdateIncident) Name() string {
	return "grafana.updateIncident"
}

func (u *UpdateIncident) Label() string {
	return "Update Incident"
}

func (u *UpdateIncident) Description() string {
	return "Update supported Grafana IRM incident fields"
}

func (u *UpdateIncident) Documentation() string {
	return `The Update Incident component updates supported fields on an existing Grafana IRM incident.

## Configuration

- **Incident**: The incident to update (required)
- **Title**: Optional new incident title
- **Severity**: Optional new severity: Pending, Critical, Major, or Minor
- **Labels**: Optional labels to add to the incident
- **Is Drill**: Optional drill flag

## Output

Returns the updated Grafana IRM incident.`
}

func (u *UpdateIncident) Icon() string {
	return "alert-triangle"
}

func (u *UpdateIncident) Color() string {
	return "blue"
}

func (u *UpdateIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (u *UpdateIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		incidentResourceField("incident", "Incident", "The incident to update"),
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "New incident title",
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "New Grafana IRM severity",
			Placeholder: "minor",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeIncidentSeverity,
				},
			},
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Labels to add to the incident",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "isDrill",
			Label:       "Is Drill",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Update whether the incident is a drill",
		},
	}
}

func (u *UpdateIncident) Setup(ctx core.SetupContext) error {
	spec, err := decodeIncidentSpec[UpdateIncidentSpec](ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateUpdateIncidentSpec(spec); err != nil {
		return err
	}
	return resolveIncidentNodeMetadata(ctx, spec.Incident)
}

func (u *UpdateIncident) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeIncidentSpec[UpdateIncidentSpec](ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateUpdateIncidentSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	incident, err := client.UpdateIncident(spec.Incident, spec.Title, spec.Severity, spec.Labels, spec.IsDrill)
	if err != nil {
		return fmt.Errorf("error updating incident: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "grafana.incident.updated", []any{incident})
}

func (u *UpdateIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (u *UpdateIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (u *UpdateIncident) Hooks() []core.Hook {
	return []core.Hook{}
}

func (u *UpdateIncident) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (u *UpdateIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (u *UpdateIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
