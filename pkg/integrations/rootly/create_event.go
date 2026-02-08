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

const CreateEventPayloadType = "rootly.timeline_event"
const CreateEventOutputChannel = "default"

type CreateEvent struct{}

type CreateEventSpec struct {
	IncidentID string `json:"incidentId"`
	Body       string `json:"body"`
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

- **Investigation notes**: Post investigation notes from SuperPlane when a workflow step completes
- **Automated status updates**: Add status updates to the incident timeline from CI or monitoring
- **Cross-system sync**: Sync comments from Jira or Slack into the Rootly incident timeline
- **Audit trail**: Record workflow actions in the incident timeline

## Configuration

- **Incident ID** (required): The Rootly incident UUID to add the event to. Accepts expressions (e.g., ` + "`{{ event.incident.id }}`" + `).
- **Event** (required): The note/annotation text. Supports Markdown formatting in Rootly.
- **Visibility** (optional): Event visibility - "internal" (default) or "external".

## Output

Emits the created timeline event to the default channel:
- **id**: Timeline event ID
- **body**: Event content
- **visibility**: Event visibility
- **occurred_at**: When the event occurred
- **created_at**: When the event was created

## Notes

- This is a synchronous component - it creates the event and returns immediately
- Use expressions to pass incident IDs from upstream components (e.g., On Incident trigger)
- Markdown is supported in the event body`
}

func (c *CreateEvent) Icon() string {
	return "message-square"
}

func (c *CreateEvent) Color() string {
	return "gray"
}

func (c *CreateEvent) ExampleOutput() map[string]any {
	return map[string]any{
		"id":          "te_12345678",
		"body":        "Investigation started by automation workflow",
		"visibility":  "internal",
		"occurred_at": "2026-02-08T10:00:00Z",
		"created_at":  "2026-02-08T10:00:00Z",
	}
}

func (c *CreateEvent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:        CreateEventOutputChannel,
			Label:       "Default",
			Description: "Emits the created timeline event",
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
			Description: "The Rootly incident UUID to add the event to",
			Placeholder: "e.g. {{ event.incident.id }}",
		},
		{
			Name:        "body",
			Label:       "Event",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The note/annotation text (supports Markdown)",
			Placeholder: "Enter your timeline event content...",
		},
		{
			Name:        "visibility",
			Label:       "Visibility",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Event visibility (internal or external)",
			Default:     "internal",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Internal", Value: "internal"},
						{Label: "External", Value: "external"},
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

	// Body is required but may be an expression, so we only validate if it's a literal
	return nil
}

func (c *CreateEvent) Execute(ctx core.ExecutionContext) error {
	spec := CreateEventSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.IncidentID == "" {
		return ctx.ExecutionState.Fail("validation_error", "incident ID is required")
	}

	if spec.Body == "" {
		return ctx.ExecutionState.Fail("validation_error", "event body is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Rootly client: %w", err)
	}

	visibility := spec.Visibility
	if visibility == "" {
		visibility = "internal"
	}

	event, err := client.CreateTimelineEvent(spec.IncidentID, spec.Body, visibility)
	if err != nil {
		return ctx.ExecutionState.Fail("api_error", fmt.Sprintf("failed to create timeline event: %v", err))
	}

	if event == nil {
		return ctx.ExecutionState.Fail("api_error", "timeline event creation returned empty response")
	}

	payload := map[string]any{
		"id":          event.ID,
		"body":        event.Body,
		"visibility":  event.Visibility,
		"occurred_at": event.OccurredAt,
		"created_at":  event.CreatedAt,
	}

	return ctx.ExecutionState.Emit(CreateEventOutputChannel, CreateEventPayloadType, []any{payload})
}

func (c *CreateEvent) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateEvent) HandleAction(ctx core.ActionContext) error {
	return errors.New("no actions available")
}

func (c *CreateEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *CreateEvent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateEvent) Cleanup(ctx core.SetupContext) error {
	return nil
}
