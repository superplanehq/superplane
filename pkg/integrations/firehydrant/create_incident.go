package firehydrant

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
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Severity    string   `json:"severity"`
	Services    []string `json:"services"`
}

func (c *CreateIncident) Name() string {
	return "firehydrant.createIncident"
}

func (c *CreateIncident) Label() string {
	return "Create Incident"
}

func (c *CreateIncident) Description() string {
	return "Create a new incident in FireHydrant"
}

func (c *CreateIncident) Documentation() string {
	return `The Create Incident component creates a new incident in FireHydrant.

## Use Cases

- **Alert escalation**: Create incidents from monitoring alerts
- **Error tracking**: Automatically create incidents when errors are detected
- **Manual incident creation**: Create incidents from workflow events
- **Integration workflows**: Create incidents from external system events

## Configuration

- **Name**: A succinct description of the incident (required, supports expressions)
- **Description**: Additional details about the incident (optional, supports expressions)
- **Severity**: Incident severity level (optional, supports expressions)
- **Services**:List of service IDs to associate with the incident (optional)

## Output

Returns the created incident object including:
- **id**: Incident ID
- **name**: Incident name
- **description**: Incident description
- **severity**: Incident severity
- **status**: Current incident status
- **created_at**: Incident creation timestamp
- **html_url**: Link to the incident in FireHydrant`
}

func (c *CreateIncident) Icon() string {
	return "flame"
}

func (c *CreateIncident) Color() string {
	return "orange"
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
			Description: "A succinct description of the incident",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Additional details about the incident",
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
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Service IDs to associate with the incident",
			Placeholder: "Select services",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "service",
					Multi: true,
				},
			},
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

	return nil
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

	incident, err := client.CreateIncident(spec.Name, spec.Description, spec.Severity, spec.Services)
	if err != nil {
		return fmt.Errorf("failed to create incident: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"firehydrant.incident",
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
