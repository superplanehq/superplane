package pagerduty

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type EscalateIncident struct{}

type EscalateIncidentSpec struct {
	IncidentID      string `json:"incidentId"`
	FromEmail       string `json:"fromEmail"`
	EscalationLevel string `json:"escalationLevel"`
}

func (c *EscalateIncident) Name() string {
	return "pagerduty.escalateIncident"
}

func (c *EscalateIncident) Label() string {
	return "Escalate Incident"
}

func (c *EscalateIncident) Description() string {
	return "Escalate an incident to a specific level (high-urgency only)"
}

func (c *EscalateIncident) Documentation() string {
	return `The Escalate Incident component escalates an existing PagerDuty incident to a specific escalation level within its current escalation policy.

## Important: High-Urgency Incidents Only

**This action only works on high-urgency incidents.** Low-urgency incidents cannot be escalated in PagerDuty. If you need to reassign a low-urgency incident, use the "Reassign Escalation Policy" action instead.

## What is Escalation?

In PagerDuty, an escalation policy defines a chain of responders:
- **Level 1**: Primary on-call (e.g., the assigned engineer)
- **Level 2**: Secondary responder (e.g., team lead)
- **Level 3**: Tertiary responder (e.g., manager)
- And so on...

Escalating an incident moves it to a higher level, notifying the responders at that level immediately instead of waiting for the automatic escalation timeout.

## Use Cases

- **Manual escalation**: Escalate when the current responder cannot resolve the issue
- **Automated escalation**: Automatically escalate based on workflow conditions (e.g., incident age)
- **Skip levels**: Jump directly to a higher level for critical situations

## Configuration

- **Incident ID**: The ID of the incident to escalate (e.g., A12BC34567...)
- **Escalation Level**: The level to escalate to (1-10). This is the level number within the incident's current escalation policy.
- **From Email**: Email address of a valid PagerDuty user (required for App OAuth, optional for API tokens)

## Output

Returns the escalated incident object with all current information.`
}

func (c *EscalateIncident) Icon() string {
	return "arrow-up"
}

func (c *EscalateIncident) Color() string {
	return "gray"
}

func (c *EscalateIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *EscalateIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the incident to escalate (e.g., A12BC34567...)",
			Placeholder: "e.g., A12BC34567...",
		},
		{
			Name:        "escalationLevel",
			Label:       "Escalation Level",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "The level to escalate to within the incident's escalation policy. Only works on high-urgency incidents.",
			Default:     "2",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Level 1", Value: "1"},
						{Label: "Level 2", Value: "2"},
						{Label: "Level 3", Value: "3"},
						{Label: "Level 4", Value: "4"},
						{Label: "Level 5", Value: "5"},
						{Label: "Level 6", Value: "6"},
						{Label: "Level 7", Value: "7"},
						{Label: "Level 8", Value: "8"},
						{Label: "Level 9", Value: "9"},
						{Label: "Level 10", Value: "10"},
					},
				},
			},
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

func (c *EscalateIncident) Setup(ctx core.SetupContext) error {
	spec := EscalateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incidentId is required")
	}

	if spec.EscalationLevel == "" {
		return errors.New("escalationLevel is required")
	}

	level, err := strconv.Atoi(spec.EscalationLevel)
	if err != nil {
		return fmt.Errorf("invalid escalationLevel: %v", err)
	}

	if level < 1 || level > 10 {
		return errors.New("escalationLevel must be between 1 and 10")
	}

	return ctx.Metadata.Set(NodeMetadata{})
}

func (c *EscalateIncident) Execute(ctx core.ExecutionContext) error {
	spec := EscalateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	level, err := strconv.Atoi(spec.EscalationLevel)
	if err != nil {
		return fmt.Errorf("invalid escalationLevel: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	incident, err := client.UpdateIncident(
		spec.IncidentID,
		spec.FromEmail,
		"", // status - not changing
		"", // priority - not changing
		"", // title - not changing
		"", // description - not changing
		"", // escalation policy - not changing
		level,
		nil, // assignees - not changing
	)
	if err != nil {
		return fmt.Errorf("failed to escalate incident: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"pagerduty.incident",
		[]any{incident},
	)
}

func (c *EscalateIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *EscalateIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *EscalateIncident) Actions() []core.Action {
	return []core.Action{}
}

func (c *EscalateIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *EscalateIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *EscalateIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
