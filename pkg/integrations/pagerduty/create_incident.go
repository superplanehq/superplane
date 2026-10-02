package pagerduty

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateIncident struct{}

type CreateIncidentSpec struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Urgency     string `json:"urgency"`
	Service     string `json:"service"`
	FromEmail   string `json:"fromEmail"`
}

func (c *CreateIncident) Name() string {
	return "pagerduty.createIncident"
}

func (c *CreateIncident) Label() string {
	return "Create Incident"
}

func (c *CreateIncident) Description() string {
	return "Create a new incident in PagerDuty"
}

func (c *CreateIncident) Documentation() string {
	return `The Create Incident component creates a new incident in PagerDuty.

## Use Cases

- **Alert escalation**: Create incidents from monitoring alerts
- **Error tracking**: Automatically create incidents when errors are detected
- **Manual incident creation**: Create incidents from workflow events
- **Integration workflows**: Create incidents from external system events

## Configuration

- **Title**: A succinct description of the incident (required, supports expressions)
- **Description**: Additional details about the incident (optional, supports expressions)
- **Urgency**: Incident urgency level (high or low)
- **Service**: Select the PagerDuty service to create the incident in
- **From Email**: Email of the PagerDuty user to act as, sent as the From header. Required with App OAuth or an account-scoped API token; optional with a user-scoped API token, which already identifies the user.

## Output

Returns the created incident object including:
- **id**: Incident ID
- **incident_number**: Human-readable incident number
- **status**: Current incident status
- **urgency**: Incident urgency
- **service**: Service information
- **created_at**: Incident creation timestamp`
}

func (c *CreateIncident) Icon() string {
	return "alert-triangle"
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
			Name:        "title",
			Label:       "Incident Title",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "A succinct description of the incident",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Additional details about the incident",
		},
		{
			Name:     "urgency",
			Label:    "Urgency",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "high",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "High", Value: "high"},
						{Label: "Low", Value: "low"},
					},
				},
			},
		},
		{
			Name:        "service",
			Label:       "Service",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The PagerDuty service to create the incident for",
			Placeholder: "Select a service",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "service",
				},
			},
		},
		{
			Name:        "fromEmail",
			Label:       "From Email",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Email of the PagerDuty user to act as, sent as the From header. Required with App OAuth or an account-scoped API token; optional with a user-scoped API token, which already identifies the user.",
			Placeholder: "user@example.com",
		},
	}
}

func (c *CreateIncident) Setup(ctx core.SetupContext) error {
	spec := CreateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Title == "" {
		return errors.New("title is required")
	}

	if spec.Urgency == "" {
		return errors.New("urgency is required")
	}

	if spec.Service == "" {
		return errors.New("service is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	service, err := client.GetService(spec.Service)
	if err != nil {
		return fmt.Errorf("error finding service: %v", err)
	}

	return ctx.Metadata.Set(NodeMetadata{Service: service})
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

	incident, err := client.CreateIncident(spec.Title, spec.Service, spec.Urgency, spec.Description, spec.FromEmail)
	if err != nil {
		return fmt.Errorf("failed to create incident: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"pagerduty.incident",
		[]any{incident},
	)
}

func (c *CreateIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateIncident) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateIncident) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
