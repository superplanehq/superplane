package grafana

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeclareIncident struct{}

type DeclareIncidentSpec struct {
	Title       string   `json:"title" mapstructure:"title"`
	Severity    string   `json:"severity" mapstructure:"severity"`
	Description string   `json:"description" mapstructure:"description"`
	Labels      []string `json:"labels" mapstructure:"labels"`
	IsDrill     bool     `json:"isDrill" mapstructure:"isDrill"`
}

func (d *DeclareIncident) Name() string {
	return "grafana.declareIncident"
}

func (d *DeclareIncident) Label() string {
	return "Declare Incident"
}

func (d *DeclareIncident) Description() string {
	return "Declare a Grafana IRM incident"
}

func (d *DeclareIncident) Documentation() string {
	return `The Declare Incident component creates a new incident in Grafana IRM.

## Use Cases

- **Automated incident declaration**: Open an incident when a deployment, alert, or workflow detects a production issue
- **Drill automation**: Create drill incidents for operational exercises

## Configuration

- **Title**: Incident title (required)
- **Severity**: Pending, Critical, Major, or Minor (required)
- **Description**: Optional initial status update added to the incident
- **Labels**: Optional incident labels
- **Is Drill**: Mark the incident as a drill

## Output

Returns the created Grafana IRM incident.`
}

func (d *DeclareIncident) Icon() string {
	return "alert-triangle"
}

func (d *DeclareIncident) Color() string {
	return "blue"
}

func (d *DeclareIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeclareIncident) ExampleOutput() map[string]any {
	return exampleIncidentOutput("grafana.incident.declared")
}

func (d *DeclareIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Incident title",
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Grafana IRM severity",
			Placeholder: "minor",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeIncidentSeverity,
				},
			},
		},
		{
			Name:        "description",
			Label:       "Initial Status Update",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Initial status update added to the incident",
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Labels to attach to the incident",
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
			Description: "Create the incident as a drill",
		},
	}
}

func (d *DeclareIncident) Setup(ctx core.SetupContext) error {
	spec, err := decodeIncidentSpec[DeclareIncidentSpec](ctx.Configuration)
	if err != nil {
		return err
	}
	return validateDeclareIncidentSpec(spec)
}

func (d *DeclareIncident) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeIncidentSpec[DeclareIncidentSpec](ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateDeclareIncidentSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	incident, err := client.DeclareIncident(spec.Title, spec.Severity, spec.Description, spec.Labels, spec.IsDrill)
	if err != nil {
		return fmt.Errorf("error declaring incident: %w", err)
	}

	if incident != nil && strings.TrimSpace(incident.IncidentURL) == "" {
		incident.IncidentURL, _ = buildIncidentWebURL(ctx.Integration, incident.IncidentID)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "grafana.incident.declared", []any{incident})
}

func (d *DeclareIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeclareIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeclareIncident) Actions() []core.Action {
	return []core.Action{}
}

func (d *DeclareIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (d *DeclareIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeclareIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
