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
	IncidentID       string `json:"incidentId"`
	FromEmail        string `json:"fromEmail"`
	EscalationPolicy string `json:"escalationPolicy"`
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
	return `The Acknowledge Incident component acknowledges a PagerDuty incident so it is assigned and being worked on.

## Use Cases

- **Auto-acknowledge incidents**: Acknowledge incidents when an on-call engineer is assigned or when a workflow starts from SuperPlane
- **Automation remediation**: Acknowledge incidents when automation begins remediation
- **Sync incident state**: Sync incident state with Jira or Slack (e.g., acknowledge when ticket is in progress)

## Configuration

- **Incident ID**: PagerDuty incident ID (e.g., P1ABC23)
- **From Email**: Email or user ID of the user acknowledging (must be a valid PagerDuty user). Required for App OAuth and account-level API tokens, optional for user-level API tokens.
- **Escalation Policy**: Optional escalation policy override

## Output

Returns the acknowledged incident object with all current information including status, acknowledged by, and timestamp.`
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
			Description: "The ID of the incident to acknowledge (e.g., P1ABC23)",
			Placeholder: "e.g., P1ABC23",
		},
		{
			Name:        "fromEmail",
			Label:       "From Email",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Email address of a valid PagerDuty user. Required for App OAuth and account-level API tokens, optional for user-level API tokens.",
			Placeholder: "user@example.com",
		},
		{
			Name:        "escalationPolicy",
			Label:       "Escalation Policy",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional escalation policy override",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "escalation_policy",
				},
			},
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
		spec.EscalationPolicy,
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
