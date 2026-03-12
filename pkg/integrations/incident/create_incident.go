package incident

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
	Name       string `json:"name"`
	Summary    string `json:"summary"`
	SeverityID string `json:"severityId"`
	Visibility string `json:"visibility"`
}

func (c *CreateIncident) Name() string {
	return "incident.createIncident"
}

func (c *CreateIncident) Label() string {
	return "Create Incident"
}

func (c *CreateIncident) Description() string {
	return "Create a new incident in incident.io"
}

func (c *CreateIncident) Documentation() string {
	return `The Create Incident component creates a new incident in incident.io.

## Use Cases

- **Alert escalation**: Create incidents from monitoring alerts
- **Error tracking**: Automatically create incidents when errors are detected
- **Manual incident creation**: Create incidents from workflow events
- **Integration workflows**: Create incidents from external system events

## Configuration

- **Name**: The incident name or title (required, supports expressions)
- **Summary**: Additional details about the incident (optional, supports expressions)
- **Severity**: Select a severity from your incident.io organization (required)
- **Visibility**: Public (anyone can access) or Private (only invited users)

## Output

Returns the created incident object including:
- **id**: Incident ID
- **name**: Incident name
- **reference**: Human-readable reference (e.g. INC-123)
- **permalink**: Link to the incident in incident.io
- **severity**: Severity details if set
- **visibility**: public or private
- **created_at**, **updated_at**: Timestamps`
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
			Name:        "name",
			Label:       "Incident Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "A succinct name or title for the incident",
		},
		{
			Name:        "severityId",
			Label:       "Severity",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The severity level of the incident",
			Placeholder: "Select a severity",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "severity",
				},
			},
		},
		{
			Name:        "summary",
			Label:       "Summary",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Additional details about the incident",
		},
		{
			Name:     "visibility",
			Label:    "Visibility",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  "public",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Public", Value: "public"},
						{Label: "Private", Value: "private"},
					},
				},
			},
		},
	}
}

func (c *CreateIncident) Setup(ctx core.SetupContext) error {
	spec := CreateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}
	if spec.SeverityID == "" {
		return errors.New("severity is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}
	severities, err := client.ListSeverities()
	if err != nil {
		return fmt.Errorf("error listing severities: %w", err)
	}
	var found bool
	for _, s := range severities {
		if s.ID == spec.SeverityID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("severity with id %q not found or no longer available; select a severity from the list", spec.SeverityID)
	}

	return nil
}

func (c *CreateIncident) Execute(ctx core.ExecutionContext) error {
	spec := CreateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	// Use execution ID as idempotency key so the same workflow run does not create duplicates on retry
	idempotencyKey := ctx.ID.String()

	incident, err := client.CreateIncident(spec.Name, idempotencyKey, spec.SeverityID, spec.Visibility, spec.Summary)
	if err != nil {
		return fmt.Errorf("failed to create incident: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"incident.incident",
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
	return nil
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
