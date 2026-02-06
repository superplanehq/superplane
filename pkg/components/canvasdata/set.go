package canvasdata

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_set.json
var setExampleOutputBytes []byte

var setExampleOutputOnce sync.Once
var setExampleOutput map[string]any

const SetComponentName = "canvasdata.set"
const SetPayloadType = "canvasdata.set.finished"

func init() {
	registry.RegisterComponent(SetComponentName, &SetCanvasData{})
}

type SetSpec struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type SetCanvasData struct{}

func (c *SetCanvasData) Name() string {
	return SetComponentName
}

func (c *SetCanvasData) Label() string {
	return "Set Canvas Data"
}

func (c *SetCanvasData) Description() string {
	return "Store a value under a key on the canvas. Each write creates a new history entry."
}

func (c *SetCanvasData) Documentation() string {
	return `Set Canvas Data writes a key-value pair to the canvas data store. Every write is versioned so you can later read the current or previous values.

## Use Cases

- **Last deployed version**: Save the version or SHA after a deploy step.
- **Ephemeral resources**: Record machine IDs or resource names for later teardown.
- **Canvas state**: Store small state (timestamps, flags) shared across workflow runs.

## Behavior

- **Key**: A string key (e.g. app/backend/last_version, ephemeral/machines). Use namespaced keys to avoid collisions.
- **Value**: The value to store. Can be a string or expression that evaluates to a string, number, or object (stored as JSON).
- Output includes the key, value, and created_at timestamp for use downstream.`
}

func (c *SetCanvasData) Icon() string {
	return "database"
}

func (c *SetCanvasData) Color() string {
	return "green"
}

func (c *SetCanvasData) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *SetCanvasData) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&setExampleOutputOnce, setExampleOutputBytes, &setExampleOutput)
}

func (c *SetCanvasData) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "key",
			Label:       "Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "app/my-service/last_version",
			Description: "Canvas-scoped key. Use namespaced keys (e.g. app/..., ephemeral/...) to avoid collisions.",
		},
		{
			Name:        "value",
			Label:       "Value",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "Value to store. Can be a string, number, or object (stored as JSON).",
		},
	}
}

func (c *SetCanvasData) Execute(ctx core.ExecutionContext) error {
	spec := SetSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Key == "" {
		return fmt.Errorf("key is required")
	}

	valueStr, err := valueToString(spec.Value)
	if err != nil {
		return fmt.Errorf("value: %w", err)
	}

	canvasID, err := uuid.Parse(ctx.WorkflowID)
	if err != nil {
		return fmt.Errorf("invalid workflow id: %w", err)
	}

	rec, err := models.SetCanvasData(canvasID, spec.Key, valueStr)
	if err != nil {
		return fmt.Errorf("set canvas data: %w", err)
	}

	payload := map[string]any{
		"key":   rec.Key,
		"value": rec.Value,
	}
	if rec.CreatedAt != nil {
		payload["created_at"] = rec.CreatedAt.Format("2006-01-02T15:04:05.000Z07:00")
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		SetPayloadType,
		[]any{payload},
	)
}

func valueToString(v any) (string, error) {
	if v == nil {
		return "", nil
	}
	switch t := v.(type) {
	case string:
		return t, nil
	case float64, int, int64, bool:
		return fmt.Sprintf("%v", t), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

func (c *SetCanvasData) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SetCanvasData) Actions() []core.Action {
	return []core.Action{}
}

func (c *SetCanvasData) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("set canvas data does not support actions")
}

func (c *SetCanvasData) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *SetCanvasData) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SetCanvasData) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *SetCanvasData) Cleanup(ctx core.SetupContext) error {
	return nil
}
