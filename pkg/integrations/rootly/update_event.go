package rootly

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const UpdateEventPayloadType = "rootly.incident.event.updated"
const UpdateEventSuccessChannel = "success"

type UpdateEvent struct{}

type UpdateEventSpec struct {
	IncidentID string `json:"incidentId" mapstructure:"incidentId"`
	EventID    string `json:"eventId" mapstructure:"eventId"`
	Event      string `json:"event" mapstructure:"event"`
	Visibility string `json:"visibility" mapstructure:"visibility"`
}

type UpdateEventOutput struct {
	ID         string `json:"id"`
	Event      string `json:"event"`
	Visibility string `json:"visibility"`
	OccurredAt string `json:"occurredAt"`
	UpdatedAt  string `json:"updatedAt"`
}

func (u *UpdateEvent) Name() string {
	return "rootly.updateEvent"
}

func (u *UpdateEvent) Label() string {
	return "Update Event"
}

func (u *UpdateEvent) Description() string {
	return "Update an existing incident timeline event in Rootly"
}

func (u *UpdateEvent) Documentation() string {
	return `The Update Event component updates an existing incident timeline event in Rootly.

## Use Cases

- **Correction**: Correct a typo or update investigation notes from a workflow step
- **Visibility changes**: Change event visibility (internal to external) for customer-facing updates
- **Cross-system sync**: Sync edits from Jira or Slack back to the Rootly timeline

## How It Works

1. Takes an incident ID, event ID, and updated content
2. Updates the existing timeline event via the Rootly API
3. Returns the updated event with new timestamps
4. Emits the data on the success channel

## Configuration

- **Incident ID** (required): The Rootly incident UUID the event belongs to. Accepts expressions.
- **Event ID** (required): The Rootly incident event UUID to update. Accepts expressions.
- **Event** (required): The new note/annotation text. Supports Markdown formatting.
- **Visibility** (optional): Set to "internal" or "external" to change visibility.

## Output

Single output channel that emits:
- ` + "`id`" + `: The event ID
- ` + "`event`" + `: The updated event text
- ` + "`visibility`" + `: Event visibility (internal/external)
- ` + "`occurredAt`" + `: When the event occurred
- ` + "`updatedAt`" + `: When the event was last updated

## Notes

- Only the provided fields are updated; omit visibility to keep existing value
- If the incident or event doesn't exist, an error is returned
- Update history is preserved in Rootly`
}

func (u *UpdateEvent) Icon() string {
	return "edit"
}

func (u *UpdateEvent) Color() string {
	return "orange"
}

func (u *UpdateEvent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  UpdateEventSuccessChannel,
			Label: "Success",
		},
	}
}

func (u *UpdateEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. abc123",
			Description: "The Rootly incident UUID the event belongs to",
		},
		{
			Name:        "eventId",
			Label:       "Event ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. evt456",
			Description: "The Rootly incident event UUID to update",
		},
		{
			Name:        "event",
			Label:       "Event",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "Updated investigation note...",
			Description: "The new note/annotation text (supports Markdown)",
			TypeOptions: &configuration.TypeOptions{
				String: &configuration.StringTypeOptions{
					Multiline: true,
				},
			},
		},
		{
			Name:        "visibility",
			Label:       "Visibility",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Event visibility (internal or external)",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.SelectOption{
						{Value: "", Label: "Keep existing"},
						{Value: "internal", Label: "Internal"},
						{Value: "external", Label: "External"},
					},
				},
			},
		},
	}
}

func (u *UpdateEvent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (u *UpdateEvent) Setup(ctx core.SetupContext) error {
	spec := UpdateEventSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return nil
}

func (u *UpdateEvent) Execute(ctx core.ExecutionContext) error {
	spec := UpdateEventSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.IncidentID == "" {
		return fmt.Errorf("incident ID is required")
	}

	if spec.EventID == "" {
		return fmt.Errorf("event ID is required")
	}

	if spec.Event == "" {
		return fmt.Errorf("event text is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	ctx.Logger.Infof("Updating event=%s for incident=%s", spec.EventID, spec.IncidentID)

	event, err := client.UpdateIncidentEvent(spec.IncidentID, spec.EventID, spec.Event, spec.Visibility)
	if err != nil {
		return fmt.Errorf("error updating event: %w", err)
	}

	ctx.Logger.Infof("Updated event=%s for incident=%s", event.ID, spec.IncidentID)

	output := UpdateEventOutput{
		ID:         event.ID,
		Event:      event.Event,
		Visibility: event.Visibility,
		OccurredAt: event.OccurredAt,
		UpdatedAt:  event.UpdatedAt,
	}

	// Store metadata for reference
	ctx.Metadata.Set(map[string]any{
		"eventId":    event.ID,
		"incidentId": spec.IncidentID,
	})

	return ctx.Requests.Emit(UpdateEventSuccessChannel, UpdateEventPayloadType, []any{output})
}

func (u *UpdateEvent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (u *UpdateEvent) Actions() []core.Action {
	return []core.Action{}
}

func (u *UpdateEvent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions available for UpdateEvent")
}

func (u *UpdateEvent) Cleanup(ctx core.SetupContext) error {
	return nil
}
