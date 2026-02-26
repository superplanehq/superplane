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
	Key       string      `json:"key"`
	Value     any         `json:"value"`
	ValueList []ValuePair `json:"valueList,omitempty"`
	Operation string `json:"operation"`
	UniqueBy *string `json:"uniqueBy,omitempty"`
}

type ValuePair struct {
	Name  string `json:"name"`
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

1. Reads ` + "`key`" + ` and ` + "`fields`" + ` from configuration
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
			Name:        "valueList",
			Label:       "Fields",
			Type:        configuration.FieldTypeList,
			Description: "Fill multiple fields without writing a JSON object",
			Required:    true,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Field",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "name",
								Label:       "Field Name",
								Type:        configuration.FieldTypeString,
								Description: "Object field name",
								Required:    true,
							},
							{
								Name:        "value",
								Label:       "Field Value",
								Type:        configuration.FieldTypeExpression,
								Description: "Object field value (can be an expression)",
								Required:    true,
							},
						},
					},
				},
			},
		},
		{
			Name:        "operation",
			Label:       "Operation",
			Type:        configuration.FieldTypeSelect,
			Description: "How to persist value for this key",
			Required:    true,
			Default:     "set",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Set value", Value: "set"},
						{Label: "Append to list", Value: "append"},
					},
				},
			},
		},
		{
			Name:        "uniqueBy",
			Label:       "Unique By (optional)",
			Type:        configuration.FieldTypeString,
			Description: "When appending objects, replace existing list item that matches this field",
			Required:    false,
			Togglable:   true,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "operation", Values: []string{"append"}},
			},
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

	if spec.Operation == "" {
		spec.Operation = "set"
	}

	storedValue := buildValue(spec)

	switch spec.Operation {
	case "set":
	case "append":
		var err error
		storedValue, err = appendToCanvasList(ctx, spec, storedValue)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported operation: %s", spec.Operation)
	}

	if err := ctx.CanvasData.Set(spec.Key, storedValue); err != nil {
		return fmt.Errorf("failed to set canvas data: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PayloadType,
		[]any{
			map[string]any{
				"key":       spec.Key,
				"value":     storedValue,
				"operation": spec.Operation,
			},
		},
	)
}

func buildValue(spec Spec) any {
	if len(spec.ValueList) == 0 {
		return spec.Value
	}

	objectValue := make(map[string]any, len(spec.ValueList))
	for _, pair := range spec.ValueList {
		name := strings.TrimSpace(pair.Name)
		if name == "" {
			continue
		}
		objectValue[name] = pair.Value
	}

	return objectValue
}

func appendToCanvasList(ctx core.ExecutionContext, spec Spec, nextValue any) (any, error) {
	existingValue, exists, err := ctx.CanvasData.Get(spec.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing canvas data: %w", err)
	}

	if !exists || existingValue == nil {
		return []any{nextValue}, nil
	}

	list, ok := existingValue.([]any)
	if !ok {
		return nil, fmt.Errorf("key %s already exists but is not a list", spec.Key)
	}

	uniqueBy := ""
	if spec.UniqueBy != nil {
		uniqueBy = strings.TrimSpace(*spec.UniqueBy)
	}
	if uniqueBy == "" {
		return append(list, nextValue), nil
	}

	newItem, ok := nextValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("uniqueBy requires value to be an object")
	}

	newUniqueValue, ok := newItem[uniqueBy]
	if !ok {
		return nil, fmt.Errorf("value object does not contain uniqueBy field %s", uniqueBy)
	}

	for i, item := range list {
		existingObj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		existingUniqueValue, ok := existingObj[uniqueBy]
		if !ok {
			continue
		}
		if fmt.Sprintf("%v", existingUniqueValue) == fmt.Sprintf("%v", newUniqueValue) {
			list[i] = nextValue
			return list, nil
		}
	}

	return append(list, nextValue), nil
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
