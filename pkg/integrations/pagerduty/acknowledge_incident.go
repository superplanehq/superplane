package pagerduty

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type AcknowledgeIncident struct{}

type AcknowledgeIncidentSpec struct {
	IncidentID string `json:"incidentId"`
	FromEmail  string `json:"fromEmail"`
}

func (c *AcknowledgeIncident) Name() string {
	return "pagerduty.acknowledgeIncident"
}

func (c *AcknowledgeIncident) Label() string {
	return "Acknowledge Incident"
}

func (c *AcknowledgeIncident) Description() string {
	return "Acknowledge an incident in PagerDuty"
}

func (c *AcknowledgeIncident) Documentation() string {
	return `The Acknowledge Incident component acknowledges an existing PagerDuty incident.

## Use Cases

- **Incident response**: Acknowledge an incident to stop escalations and indicate someone is working on it
- **Automated acknowledgement**: Automatically acknowledge incidents based on workflow conditions
- **Integration workflows**: Acknowledge incidents when related events occur in other systems

## Configuration

- **Incident ID**: The ID of the incident to acknowledge (e.g., A12BC34567...)
- **From Email**: Email address of a valid PagerDuty user (required for App OAuth, optional for API tokens)

## Behavior

When an incident is acknowledged, escalations are paused and the incident status changes to "acknowledged". The incident will remain acknowledged until it is resolved or re-triggered.

## Output

Returns the acknowledged incident object with all current information.`
}

func (c *AcknowledgeIncident) Icon() string {
	return "check"
}

func (c *AcknowledgeIncident) Color() string {
	return "gray"
}

func (c *AcknowledgeIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AcknowledgeIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the incident to acknowledge (e.g., A12BC34567...)",
			Placeholder: "e.g., A12BC34567...",
		},
		{
			Name:        "fromEmail",
			Label:       "From Email",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Email address of a valid PagerDuty user. Required for App OAuth and account-level API tokens, optional for user-level API tokens.",
			Placeholder: "user@example.com",
		},
	}
}

func (c *AcknowledgeIncident) Setup(ctx core.SetupContext) error {
	spec := AcknowledgeIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incidentId is required")
	}

	return ctx.Metadata.Set(NodeMetadata{})
}

func (c *AcknowledgeIncident) Execute(ctx core.ExecutionContext) error {
	spec := AcknowledgeIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	incident, err := client.UpdateIncident(
		spec.IncidentID,
		spec.FromEmail,
		"acknowledged",
		"",
		"",
		"",
		"",
		0,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to acknowledge incident: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"pagerduty.incident",
		[]any{incident},
	)
}

func (c *AcknowledgeIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AcknowledgeIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AcknowledgeIncident) Actions() []core.Action {
	return []core.Action{}
}

func (c *AcknowledgeIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *AcknowledgeIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *AcknowledgeIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
