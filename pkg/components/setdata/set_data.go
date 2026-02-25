package setdata

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "setData"
const PayloadType = "data.set"

func init() {
	registry.RegisterComponent(ComponentName, &SetData{})
}

type SetData struct{}

type Spec struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

func (c *SetData) Name() string {
	return ComponentName
}

func (c *SetData) Label() string {
	return "Set Data"
}

func (c *SetData) Description() string {
	return "Set a key/value pair in canvas storage"
}

func (c *SetData) Documentation() string {
	return `The Set Data component stores a key/value pair in canvas-level storage.

## Use Cases

- **Shared values**: Persist computed values for later nodes
- **Cross-path communication**: Share data between different branches
- **Reusable state**: Save intermediate values to read with Get Data

## How It Works

1. Reads ` + "`key`" + ` and ` + "`value`" + ` from configuration
2. Persists the pair in canvas-level storage
3. Emits a ` + "`data.set`" + ` event with the saved key/value`
}

func (c *SetData) Icon() string {
	return "database"
}

func (c *SetData) Color() string {
	return "blue"
}

func (c *SetData) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *SetData) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "key",
			Label:       "Key",
			Type:        configuration.FieldTypeString,
			Description: "Canvas data key to set",
			Required:    true,
		},
		{
			Name:        "value",
			Label:       "Value",
			Type:        configuration.FieldTypeExpression,
			Description: "Value to store (can be an expression)",
			Required:    true,
		},
	}
}

func (c *SetData) Execute(ctx core.ExecutionContext) error {
	if ctx.CanvasData == nil {
		return fmt.Errorf("canvas data context is not available")
	}

	var spec Spec
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Key = strings.TrimSpace(spec.Key)
	if spec.Key == "" {
		return fmt.Errorf("key is required")
	}

	if err := ctx.CanvasData.Set(spec.Key, spec.Value); err != nil {
		return fmt.Errorf("failed to set canvas data: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PayloadType,
		[]any{
			map[string]any{
				"key":   spec.Key,
				"value": spec.Value,
			},
		},
	)
}

func (c *SetData) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SetData) Actions() []core.Action {
	return []core.Action{}
}

func (c *SetData) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("setData does not support actions")
}

func (c *SetData) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *SetData) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SetData) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *SetData) Cleanup(ctx core.SetupContext) error {
	return nil
}
