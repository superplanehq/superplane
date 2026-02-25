package getdata

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

const ComponentName = "getData"
const PayloadType = "data.get"

func init() {
	registry.RegisterComponent(ComponentName, &GetData{})
}

type GetData struct{}

type Spec struct {
	Key string `json:"key"`
}

func (c *GetData) Name() string {
	return ComponentName
}

func (c *GetData) Label() string {
	return "Get Data"
}

func (c *GetData) Description() string {
	return "Get a key/value pair from canvas storage"
}

func (c *GetData) Documentation() string {
	return `The Get Data component reads a value from canvas-level storage by key.

## Use Cases

- **Reuse shared state**: Read values previously stored with Set Data
- **Cross-path lookups**: Access values set in another branch
- **Conditional logic**: Fetch stored flags and IDs for downstream decisions

## How It Works

1. Reads ` + "`key`" + ` from configuration
2. Looks up the value in canvas-level storage
3. Emits a ` + "`data.get`" + ` event with ` + "`key`" + `, ` + "`value`" + `, and ` + "`exists`" + ``
}

func (c *GetData) Icon() string {
	return "database-zap"
}

func (c *GetData) Color() string {
	return "blue"
}

func (c *GetData) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetData) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "key",
			Label:       "Key",
			Type:        configuration.FieldTypeString,
			Description: "Canvas data key to fetch",
			Required:    true,
		},
	}
}

func (c *GetData) Execute(ctx core.ExecutionContext) error {
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

	value, exists, err := ctx.CanvasData.Get(spec.Key)
	if err != nil {
		return fmt.Errorf("failed to get canvas data: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PayloadType,
		[]any{
			map[string]any{
				"key":    spec.Key,
				"value":  value,
				"exists": exists,
			},
		},
	)
}

func (c *GetData) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetData) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetData) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("getData does not support actions")
}

func (c *GetData) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *GetData) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetData) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetData) Cleanup(ctx core.SetupContext) error {
	return nil
}
