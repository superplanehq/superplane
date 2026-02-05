package rootly

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const CreateEventPayloadType = "rootly.incident.event.created"
const CreateEventSuccessChannel = "success"

type CreateEvent struct{}

type CreateEventSpec struct {
	IncidentID string `json:"incidentId" mapstructure:"incidentId"`
	Event      string `json:"event" mapstructure:"event"`
	Visibility string `json:"visibility" mapstructure:"visibility"`
}

type CreateEventOutput struct {
	ID         string `json:"id"`
	Event      string `json:"event"`
	Visibility string `json:"visibility"`
	OccurredAt string `json:"occurredAt"`
	CreatedAt  string `json:"createdAt"`
}

func (c *CreateEvent) Name() string {
	return "rootly.createEvent"
}

func (c *CreateEvent) Label() string {
	return "Create Event"
}

func (c *CreateEvent) Description() string {
	return "Add a timeline event to a Rootly incident"
}

func (c *CreateEvent) Documentation() string {
	return `The Create Event component adds a timeline event (note/annotation) to a Rootly incident.

## Use Cases

- **Investigation notes**: Post investigation notes from SuperPlane when a step completes
- **Automated status updates**: Add automated status updates to the incident timeline from CI or monitoring
- **Cross-system sync**: Sync comments from Jira or Slack into the Rootly incident timeline

## How It Works

1. Takes an incident ID and event text
2. Creates a new timeline event via the Rootly API
3. Returns the created event with its ID and timestamps
4. Emits the data on the success channel

## Configuration

- **Incident ID** (required): The Rootly incident UUID to add the event to. Accepts expressions.
- **Event** (required): The note/annotation text. Supports Markdown formatting in Rootly.
- **Visibility** (optional): Set to "internal" or "external". Defaults to Rootly's default if not specified.

## Output

Single output channel that emits:
- ` + "`id`" + `: The event ID
- ` + "`event`" + `: The event text
- ` + "`visibility`" + `: Event visibility (internal/external)
- ` + "`occurredAt`" + `: When the event occurred
- ` + "`createdAt`" + `: When the event was created

## Notes

- Markdown is supported in the event text
- Events appear in the incident timeline in chronological order
- If the incident doesn't exist, an error is returned`
}

func (c *CreateEvent) Icon() string {
	return "plus-circle"
}

func (c *CreateEvent) Color() string {
	return "orange"
}

func (c *CreateEvent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  CreateEventSuccessChannel,
			Label: "Success",
		},
	}
}

func (c *CreateEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. abc123",
			Description: "The Rootly incident UUID to add the event to",
		},
		{
			Name:        "event",
			Label:       "Event",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "Investigation note or status update...",
			Description: "The note/annotation text (supports Markdown)",
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
						{Value: "", Label: "Default"},
						{Value: "internal", Label: "Internal"},
						{Value: "external", Label: "External"},
					},
				},
			},
		},
	}
}

func (c *CreateEvent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateEvent) Setup(ctx core.SetupContext) error {
	spec := CreateEventSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return nil
}

func (c *CreateEvent) Execute(ctx core.ExecutionContext) error {
	spec := CreateEventSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.IncidentID == "" {
		return fmt.Errorf("incident ID is required")
	}

	if spec.Event == "" {
		return fmt.Errorf("event text is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	ctx.Logger.Infof("Creating event for incident=%s", spec.IncidentID)

	event, err := client.CreateIncidentEvent(spec.IncidentID, spec.Event, spec.Visibility)
	if err != nil {
		return fmt.Errorf("error creating event: %w", err)
	}

	ctx.Logger.Infof("Created event=%s for incident=%s", event.ID, spec.IncidentID)

	output := CreateEventOutput{
		ID:         event.ID,
		Event:      event.Event,
		Visibility: event.Visibility,
		OccurredAt: event.OccurredAt,
		CreatedAt:  event.CreatedAt,
	}

	// Store metadata for reference
	ctx.Metadata.Set(map[string]any{
		"eventId":    event.ID,
		"incidentId": spec.IncidentID,
	})

	return ctx.Requests.Emit(CreateEventSuccessChannel, CreateEventPayloadType, []any{output})
}

func (c *CreateEvent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateEvent) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateEvent) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions available for CreateEvent")
}

func (c *CreateEvent) Cleanup(ctx core.SetupContext) error {
	return nil
}
