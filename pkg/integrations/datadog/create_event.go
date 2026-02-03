package datadog

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateEvent struct{}

type CreateEventSpec struct {
	Title     string `json:"title"`
	Text      string `json:"text"`
	AlertType string `json:"alertType"`
	Priority  string `json:"priority"`
	Tags      string `json:"tags"`
}

func (c *CreateEvent) Name() string {
	return "datadog.createEvent"
}

func (c *CreateEvent) Label() string {
	return "Create Event"
}

func (c *CreateEvent) Description() string {
	return "Create a new event in DataDog"
}

func (c *CreateEvent) Icon() string {
	return "chart-bar"
}

func (c *CreateEvent) Color() string {
	return "gray"
}

func (c *CreateEvent) Documentation() string {
	return `The Create Event component creates a new event in DataDog.

## Use Cases

- **Deployment tracking**: Log deployment events to correlate with metrics
- **Incident annotation**: Add context to incidents with custom events
- **Workflow notifications**: Create events to track workflow execution milestones

## Outputs

The component emits an event containing:
- ` + "`id`" + `: The unique identifier of the created event
- ` + "`title`" + `: The event title
- ` + "`text`" + `: The event body
- ` + "`date_happened`" + `: Unix timestamp when the event occurred
- ` + "`alert_type`" + `: The severity level (info, warning, error, success)
- ` + "`priority`" + `: Event priority (normal, low)
- ` + "`tags`" + `: Array of tags attached to the event
- ` + "`url`" + `: Link to view the event in DataDog
`
}

func (c *CreateEvent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "title",
			Label:       "Event Title",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The title of the event (max 100 characters)",
		},
		{
			Name:        "text",
			Label:       "Event Text",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The body of the event (supports markdown)",
		},
		{
			Name:     "alertType",
			Label:    "Alert Type",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  "info",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Info", Value: "info"},
						{Label: "Warning", Value: "warning"},
						{Label: "Error", Value: "error"},
						{Label: "Success", Value: "success"},
					},
				},
			},
		},
		{
			Name:     "priority",
			Label:    "Priority",
			Type:     configuration.FieldTypeSelect,
			Required: false,
			Default:  "normal",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Normal", Value: "normal"},
						{Label: "Low", Value: "low"},
					},
				},
			},
		},
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Comma-separated list of tags (e.g., env:prod,service:web)",
			Placeholder: "env:prod,service:web",
		},
	}
}

func (c *CreateEvent) Setup(ctx core.SetupContext) error {
	spec := CreateEventSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Title == "" {
		return errors.New("title is required")
	}

	if spec.Text == "" {
		return errors.New("text is required")
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

	req := CreateEventRequest{
		Title:     spec.Title,
		Text:      spec.Text,
		AlertType: spec.AlertType,
		Priority:  spec.Priority,
	}

	if spec.Tags != "" {
		req.Tags = parseTags(spec.Tags)
	}

	event, err := client.CreateEvent(req)
	if err != nil {
		return fmt.Errorf("failed to create event: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"datadog.event",
		[]any{eventToMap(event)},
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

func eventToMap(event *Event) map[string]any {
	return map[string]any{
		"id":            event.ID,
		"title":         event.Title,
		"text":          event.Text,
		"date_happened": event.DateHappened,
		"alert_type":    event.AlertType,
		"priority":      event.Priority,
		"tags":          event.Tags,
		"url":           event.URL,
	}
}

func parseTags(tags string) []string {
	var result []string
	for _, tag := range strings.Split(tags, ",") {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func (c *CreateEvent) Cleanup(ctx core.SetupContext) error {
	return nil
}
