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

type ReassignEscalationPolicy struct{}

type ReassignEscalationPolicySpec struct {
	IncidentID       string `json:"incidentId"`
	FromEmail        string `json:"fromEmail"`
	EscalationPolicy string `json:"escalationPolicy"`
}

func (c *ReassignEscalationPolicy) Name() string {
	return "pagerduty.reassignEscalationPolicy"
}

func (c *ReassignEscalationPolicy) Label() string {
	return "Reassign Escalation Policy"
}

func (c *ReassignEscalationPolicy) Description() string {
	return "Reassign an incident to a different escalation policy in PagerDuty"
}

func (c *ReassignEscalationPolicy) Documentation() string {
	return `The Reassign Escalation Policy component changes the escalation policy of an existing PagerDuty incident.

## Use Cases

- **Team handoff**: Reassign an incident to a different team's on-call rotation
- **Escalation path change**: Move an incident to a more appropriate escalation policy
- **Cross-team collaboration**: Route incidents to specialized teams based on workflow conditions

## Configuration

- **Incident ID**: The ID of the incident to reassign (e.g., A12BC34567...)
- **Escalation Policy**: The escalation policy to assign the incident to (select from dropdown)
- **From Email**: Email address of a valid PagerDuty user (required for App OAuth, optional for API tokens)

## Behavior

When an incident's escalation policy is changed, it will be reassigned to the on-call responders defined in the new escalation policy. The incident will follow the new policy's escalation rules going forward.

This action works for both high-urgency and low-urgency incidents.

## Output

Returns the updated incident object with all current information.`
}

func (c *ReassignEscalationPolicy) Icon() string {
	return "shuffle"
}

func (c *ReassignEscalationPolicy) Color() string {
	return "gray"
}

func (c *ReassignEscalationPolicy) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ReassignEscalationPolicy) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the incident to reassign (e.g., A12BC34567...)",
			Placeholder: "e.g., A12BC34567...",
		},
		{
			Name:        "escalationPolicy",
			Label:       "Escalation Policy",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The escalation policy to assign the incident to",
			Placeholder: "Select an escalation policy",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "escalation_policy",
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

func (c *ReassignEscalationPolicy) Setup(ctx core.SetupContext) error {
	spec := ReassignEscalationPolicySpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incidentId is required")
	}

	if spec.EscalationPolicy == "" {
		return errors.New("escalationPolicy is required")
	}

	return ctx.Metadata.Set(NodeMetadata{})
}

func (c *ReassignEscalationPolicy) Execute(ctx core.ExecutionContext) error {
	spec := ReassignEscalationPolicySpec{}
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
		"",                   // status - not changing
		"",                   // priority - not changing
		"",                   // title - not changing
		"",                   // description - not changing
		spec.EscalationPolicy, // escalation policy - changing this
		nil,                  // assignees - not changing
	)
	if err != nil {
		return fmt.Errorf("failed to reassign escalation policy: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"pagerduty.incident",
		[]any{incident},
	)
}

func (c *ReassignEscalationPolicy) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ReassignEscalationPolicy) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ReassignEscalationPolicy) Actions() []core.Action {
	return []core.Action{}
}

func (c *ReassignEscalationPolicy) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *ReassignEscalationPolicy) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *ReassignEscalationPolicy) Cleanup(ctx core.SetupContext) error {
	return nil
}
