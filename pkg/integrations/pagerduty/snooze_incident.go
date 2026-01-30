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

type SnoozeIncident struct{}

type SnoozeIncidentSpec struct {
	IncidentID string `json:"incidentId"`
	Duration   string `json:"duration"`
	FromEmail  string `json:"fromEmail"`
}

func (c *SnoozeIncident) Name() string {
	return "pagerduty.snoozeIncident"
}

func (c *SnoozeIncident) Label() string {
	return "Snooze Incident"
}

func (c *SnoozeIncident) Description() string {
	return "Snooze an acknowledged incident in PagerDuty"
}

func (c *SnoozeIncident) Documentation() string {
	return `The Snooze Incident component temporarily pauses notifications for an acknowledged PagerDuty incident.

## Use Cases

- **Temporary acknowledgement**: Snooze an incident while investigating
- **Scheduled follow-up**: Re-trigger the incident after a specified time
- **Avoid escalation**: Prevent escalation while work is in progress

## Configuration

- **Incident ID**: The ID of the incident to snooze (must be in acknowledged state)
- **Duration**: How long to snooze the incident (1 hour, 4 hours, 8 hours, or 24 hours)
- **From Email**: Email address of a valid PagerDuty user (required for App OAuth, optional for API tokens)

## Behavior

When an incident is snoozed, it will remain in the acknowledged state and no further notifications will be sent. After the snooze duration expires, the incident will return to a triggered state and notifications will resume.

Note: Reassigning a snoozed incident will cancel the snooze timer.

## Output

Returns the snoozed incident object with all current information.`
}

func (c *SnoozeIncident) Icon() string {
	return "clock"
}

func (c *SnoozeIncident) Color() string {
	return "gray"
}

func (c *SnoozeIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *SnoozeIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the incident to snooze (must be in acknowledged state)",
			Placeholder: "e.g., A12BC34567...",
		},
		{
			Name:        "duration",
			Label:       "Duration",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "How long to snooze the incident",
			Default:     "3600",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "1 hour", Value: "3600"},
						{Label: "4 hours", Value: "14400"},
						{Label: "8 hours", Value: "28800"},
						{Label: "24 hours", Value: "86400"},
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

func (c *SnoozeIncident) Setup(ctx core.SetupContext) error {
	spec := SnoozeIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incidentId is required")
	}

	if spec.Duration == "" {
		return errors.New("duration is required")
	}

	_, err = strconv.Atoi(spec.Duration)
	if err != nil {
		return fmt.Errorf("invalid duration: %v", err)
	}

	return ctx.Metadata.Set(NodeMetadata{})
}

func (c *SnoozeIncident) Execute(ctx core.ExecutionContext) error {
	spec := SnoozeIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	duration, err := strconv.Atoi(spec.Duration)
	if err != nil {
		return fmt.Errorf("invalid duration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	incident, err := client.SnoozeIncident(spec.IncidentID, spec.FromEmail, duration)
	if err != nil {
		return fmt.Errorf("failed to snooze incident: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"pagerduty.incident",
		[]any{incident},
	)
}

func (c *SnoozeIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SnoozeIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SnoozeIncident) Actions() []core.Action {
	return []core.Action{}
}

func (c *SnoozeIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *SnoozeIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
