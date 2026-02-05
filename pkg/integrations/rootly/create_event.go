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
	return "Add a timeline event (note/annotation) to a Rootly incident"
}

func (c *CreateEvent) Documentation() string {
	return `The Create Event component adds a timeline event (note or annotation) to an existing incident in Rootly.

## Use Cases

- **Post investigation notes**: Add investigation notes from SuperPlane when a workflow step completes
- **Automated status updates**: Add status updates to the incident timeline from CI or monitoring
- **Sync external comments**: Sync comments from Jira or Slack into the Rootly incident timeline
- **Audit trail**: Add automated annotations for workflow actions taken

## Configuration

- **Incident ID**: Rootly incident UUID to add the event to (required, supports expressions)
- **Event**: The note/annotation text (required, supports expressions, Markdown supported in Rootly)
- **Visibility**: Visibility of the event - internal or external (optional, supports expressions)

## Output

Returns the created incident event object including:
- **id**: Event ID
- **event**: The note/annotation text
- **visibility**: Event visibility (internal/external)
- **occurred_at**: When the event occurred
- **created_at**: Event creation timestamp
- **updated_at**: Event update timestamp`
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
			Description: "Rootly incident UUID to add the event to",
		},
		{
			Name:        "event",
			Label:       "Event",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The note/annotation text (supports Markdown in Rootly)",
		},
		{
			Name:        "visibility",
			Label:       "Visibility",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Visibility of the event (internal or external)",
			Placeholder: "Select visibility",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.SelectOption{
						{Label: "Internal", Value: "internal"},
						{Label: "External", Value: "external"},
					},
				},
			},
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
		return errors.New("incident ID is required")
	}

	if spec.Event == "" {
		return errors.New("event is required")
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

	event, err := client.CreateIncidentEvent(spec.IncidentID, spec.Event, spec.Visibility)
	if err != nil {
		return fmt.Errorf("failed to create incident event: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"rootly.incidentEvent",
		[]any{event},
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
