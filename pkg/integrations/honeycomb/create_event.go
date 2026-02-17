package honeycomb

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateEvent struct{}

type CreateEventConfiguration struct {
	Dataset    string `json:"dataset" mapstructure:"dataset"`
	FieldsJSON string `json:"fields" mapstructure:"fields"`
}

func (c *CreateEvent) Name() string {
	return "honeycomb.createEvent"
}

func (c *CreateEvent) Label() string {
	return "Create Event"
}

func (c *CreateEvent) Description() string {
	return "Send an event to Honeycomb"
}

func (c *CreateEvent) Documentation() string {
	return `Sends a custom event to Honeycomb using the Events API.

The component sends a single event as a JSON object where each key becomes a Honeycomb field.

Notes:
- The request body is the JSON object you provide in "Fields".
- If you do not include a "time" field, the current time is automatically set via request header.`
}

func (c *CreateEvent) Icon() string {
	return "honeycomb"
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
			Name:        "dataset",
			Label:       "Dataset",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Honeycomb dataset slug.",
		},
		{
			Name:        "fields",
			Label:       "Fields (JSON)",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: `JSON object of fields to send, e.g. {"message":"hello","severity":"info"}`,
		},
	}
}

func (c *CreateEvent) Setup(ctx core.SetupContext) error {
	var cfg CreateEventConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if cfg.Dataset == "" {
		return errors.New("dataset is required")
	}
	if cfg.FieldsJSON == "" {
		return errors.New("fields is required")
	}

	var tmp map[string]any
	if err := json.Unmarshal([]byte(cfg.FieldsJSON), &tmp); err != nil {
		return fmt.Errorf("invalid fields json: %w", err)
	}

	return nil
}

func (c *CreateEvent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateEvent) Execute(ctx core.ExecutionContext) error {
	var cfg CreateEventConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if cfg.Dataset == "" {
		return errors.New("dataset is required")
	}
	if cfg.FieldsJSON == "" {
		return errors.New("fields is required")
	}

	var fields map[string]any
	if err := json.Unmarshal([]byte(cfg.FieldsJSON), &fields); err != nil {
		return fmt.Errorf("invalid fields json: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create honeycomb client: %w", err)
	}

	if err := client.CreateEvent(cfg.Dataset, fields); err != nil {
		return err
	}

	// Emit payload aligned with UI expectations + examples.
	payload := map[string]any{
		"status":  "sent",
		"dataset": cfg.Dataset,
		"fields":  fields,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"honeycomb.event.created",
		[]any{payload},
	)
}

func (c *CreateEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
func (c *CreateEvent) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateEvent) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateEvent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateEvent) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateEvent) ExampleOutput() map[string]any {
	return embeddedExampleOutputCreateEvent()
}
