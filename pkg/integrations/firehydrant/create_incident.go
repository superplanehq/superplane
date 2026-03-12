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
	Name        string `json:"name"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Priority    string `json:"priority"`
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

- **Alert escalation**: Create incidents from monitoring alerts or error tracking
- **Cross-tool sync**: Open a FireHydrant incident from other SuperPlane triggers (e.g., PagerDuty, Rootly, GitHub)
- **Manual incident creation**: Create incidents from workflow events
- **Automated response**: Automatically declare incidents when thresholds are breached

## Configuration

- **Name**: Incident name/title (required, supports expressions)
- **Summary**: Short summary of the incident (optional, supports expressions)
- **Description**: Detailed description of the incident (optional, supports expressions)
- **Severity**: Severity level, e.g., SEV1, SEV2 (optional, populated from FireHydrant)
- **Priority**: Priority level, e.g., P1, P2 (optional, populated from FireHydrant)

## Output

Returns the created incident object including:
- **id**: Incident ID
- **name**: Incident name
- **description**: Incident description
- **summary**: Incident summary
- **customer_impact_summary**: Summary of customer impact
- **current_milestone**: Current milestone (e.g., started, acknowledged)
- **number**: Incident number
- **incident_url**: URL to the incident in FireHydrant
- **severity**: Severity level
- **priority**: Priority level
- **tag_list**: List of tags associated with the incident
- **impacts**: List of impacts associated with the incident
- **milestones**: List of milestones associated with the incident`
}

func (c *CreateIncident) Icon() string {
	return "flame"
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
			Name:        "name",
			Label:       "Incident Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "A succinct name for the incident",
		},
		{
			Name:        "summary",
			Label:       "Summary",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "A short summary of the incident",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "A detailed description of the incident",
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "The severity level of the incident (e.g., SEV1, SEV2)",
			Placeholder: "Select a severity",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "severity",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "priority",
			Label:       "Priority",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "The priority level of the incident (e.g., P1, P2)",
			Placeholder: "Select a priority",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "priority",
					UseNameAsValue: true,
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

	incident, err := client.CreateIncident(CreateIncidentRequest{
		Name:        spec.Name,
		Summary:     spec.Summary,
		Description: spec.Description,
		Severity:    spec.Severity,
		Priority:    spec.Priority,
	})
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
