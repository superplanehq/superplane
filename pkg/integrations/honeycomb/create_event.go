package honeycomb

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

type CreateEventConfiguration struct {
	Dataset string         `json:"dataset" mapstructure:"dataset"`
	Fields  map[string]any `json:"fields" mapstructure:"fields"`
}

func (c *CreateEvent) Name() string {
	return "honeycomb.createEvent"
}

func (c *CreateEvent) Label() string {
	return "Create Event"
}

func (c *CreateEvent) Description() string {
	return "Send an event to Honeycomb dataset"
}

func (c *CreateEvent) Icon() string {
	return "honeycomb"
}

func (c *CreateEvent) Color() string {
	return "gray"
}

func (c *CreateEvent) Documentation() string {
	return `
Sends a JSON event to a Honeycomb dataset.

Each key in the JSON object becomes a Honeycomb field.

Notes:
• Dataset must exist
• Fields must be valid JSON object
• Timestamp is auto-added if missing
`
}

func (c *CreateEvent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "dataset",
			Label:    "Dataset",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "dataset",
					UseNameAsValue: false,
				},
			},
		},
		{
			Name:     "fields",
			Label:    "Fields JSON",
			Type:     configuration.FieldTypeObject,
			Required: true,
			Default:  "{\"message\":\"deploy\",\"status\":\"ok\"}",
			Description: `JSON object to send as event.
							Example:
							{"message":"deploy","status":"ok"}`,
		},
	}
}

func (c *CreateEvent) Setup(ctx core.SetupContext) error {
	var cfg CreateEventConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	cfg.Dataset = strings.TrimSpace(cfg.Dataset)
	if cfg.Dataset == "" {
		return errors.New("dataset is required")
	}

	if len(cfg.Fields) == 0 {
		return errors.New("fields json is required")
	}

	return nil
}

func (c *CreateEvent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateEvent) Execute(ctx core.ExecutionContext) error {
	var cfg CreateEventConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.CreateEvent(cfg.Dataset, cfg.Fields); err != nil {
		return err
	}

	output := map[string]any{
		"status":  "sent",
		"dataset": cfg.Dataset,
		"fields":  cfg.Fields,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"honeycomb.event.created",
		[]any{output},
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
