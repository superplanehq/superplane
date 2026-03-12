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

type CreateEvent struct{}

type CreateEventSpec struct {
	IncidentID string `json:"incidentId"`
	Event      string `json:"event"`
	Visibility string `json:"visibility"`
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

- **Investigation notes**: Add detailed investigation notes to the incident timeline
- **Status updates**: Post automated status updates as workflows progress
- **Cross-system sync**: Sync comments from external tools into the incident timeline

## Configuration

- **Incident ID**: The Rootly incident UUID to add the event to (required, supports expressions)
- **Event**: The note/annotation text (required, supports expressions)
- **Visibility**: Internal or external visibility (optional, default per Rootly)

## Output

Returns the created incident event with:
- **id**: Event ID
- **event**: Event content
- **visibility**: Event visibility
- **occurred_at**: Event timestamp
- **created_at**: Creation timestamp`
}

func (c *CreateEvent) Icon() string {
	return "message-square"
}

func (c *CreateEvent) Color() string {
	return "gray"
}

func (c *CreateEvent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The Rootly incident UUID to add the event to",
			Placeholder: "e.g., abc123-def456",
		},
		{
			Name:        "event",
			Label:       "Event",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The note/annotation text to add to the incident timeline",
		},
		{
			Name:     "visibility",
			Label:    "Visibility",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Internal", Value: "internal"},
						{Label: "External", Value: "external"},
					},
				},
			},
			Description: "Set event visibility (optional, defaults to Rootly settings)",
		},
	}
}

func (c *CreateEvent) Setup(ctx core.SetupContext) error {
	spec := CreateEventSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incidentId is required")
	}

	if spec.Event == "" {
		return errors.New("event is required")
	}

	if spec.Visibility != "" && spec.Visibility != "internal" && spec.Visibility != "external" {
		return errors.New("visibility must be internal or external")
	}

	return nil
}

func (c *CreateEvent) Execute(ctx core.ExecutionContext) error {
	spec := CreateEventSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	incidentEvent, err := client.CreateIncidentEvent(spec.IncidentID, spec.Event, spec.Visibility)
	if err != nil {
		return fmt.Errorf("failed to create incident event: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"rootly.incident.event",
		[]any{incidentEvent},
	)
}

func (c *CreateEvent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateEvent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateEvent) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateEvent) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateEvent) Cleanup(ctx core.SetupContext) error {
	return nil
}
