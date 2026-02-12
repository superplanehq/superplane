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
	IncidentID string `json:"incident_id" mapstructure:"incident_id"`
	Title      string `json:"title" mapstructure:"title"`
	Summary    string `json:"summary" mapstructure:"summary"`
	Severity   string `json:"severity" mapstructure:"severity"`
	Status     string `json:"status" mapstructure:"status"`
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

- **Status updates**: Change incident status (e.g. mitigated, resolved)
- **Severity changes**: Escalate or de-escalate incident severity
- **Title/summary updates**: Refine incident details as investigation progresses
- **Automated workflows**: Update incidents based on monitoring signals

## Configuration

- **Incident ID**: The ID of the incident to update (required, supports expressions)
- **Title**: Updated incident title (optional, supports expressions)
- **Summary**: Updated incident summary (optional, supports expressions)
- **Severity**: Updated severity level (optional, supports expressions)
- **Status**: Updated incident status (optional, supports expressions)

## Output

Returns the updated incident object including:
- **id**: Incident ID
- **title**: Incident title
- **status**: Current incident status
- **severity**: Incident severity
- **started_at**: Incident creation timestamp
- **url**: Link to the incident in Rootly`
}

func (c *UpdateIncident) Icon() string {
	return "alert-triangle"
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
			Name:        "incident_id",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The Rootly incident ID to update",
		},
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Updated incident title",
		},
		{
			Name:        "summary",
			Label:       "Summary",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Updated incident summary",
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Updated severity level",
			Placeholder: "Select a severity",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "severity",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "status",
			Label:       "Status",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Updated incident status (e.g. mitigated, resolved)",
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
		return errors.New("incident_id is required")
	}

	return nil
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

	incident, err := client.UpdateIncident(spec.IncidentID, spec.Title, spec.Summary, spec.Severity, spec.Status)
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
