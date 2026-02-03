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

type CreateIncident struct{}

type CreateIncidentSpec struct {
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Severity string `json:"severity"`
}

func (c *CreateIncident) Name() string {
	return "rootly.createIncident"
}

func (c *CreateIncident) Label() string {
	return "Create Incident"
}

func (c *CreateIncident) Description() string {
	return "Create a new incident in Rootly"
}

func (c *CreateIncident) Documentation() string {
	return `The Create Incident component creates a new incident in Rootly.

## Use Cases

- **Alert escalation**: Create incidents from monitoring alerts
- **Error tracking**: Automatically create incidents when errors are detected
- **Manual incident creation**: Create incidents from workflow events
- **Integration workflows**: Create incidents from external system events

## Configuration

- **Title**: A succinct description of the incident (required, supports expressions)
- **Summary**: Additional details about the incident (optional, supports expressions)
- **Severity**: Incident severity level (optional, supports expressions)

## Output

Returns the created incident object including:
- **id**: Incident ID
- **title**: Incident title
- **status**: Current incident status
- **severity**: Incident severity
- **started_at**: Incident creation timestamp
- **url**: Link to the incident in Rootly`
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
			Name:        "summary",
			Label:       "Summary",
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

	incident, err := client.CreateIncident(spec.Title, spec.Summary, spec.Severity)
	if err != nil {
		return fmt.Errorf("failed to create incident: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"rootly.incident",
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
