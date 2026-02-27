package addmemory

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

const ComponentName = "addMemory"
const PayloadType = "memory.added"

func init() {
	registry.RegisterComponent(ComponentName, &AddMemory{})
}

type AddMemory struct{}

type Spec struct {
	Namespace string      `json:"namespace"`
	Values    any         `json:"values,omitempty"`
	ValueList []ValuePair `json:"valueList,omitempty"`
}

type ValuePair struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

func (c *AddMemory) Name() string {
	return ComponentName
}

func (c *AddMemory) Label() string {
	return "Add Memory"
}

func (c *AddMemory) Description() string {
	return "Add a namespaced JSON value to canvas memory"
}

func (c *AddMemory) Documentation() string {
	return `The Add Memory component appends a new item to canvas-level memory storage.

## Use Cases

- Persist identifiers for later cleanup paths
- Store cross-run mappings (for example pull request to resource ID)
- Keep structured operational context per canvas

## How It Works

1. Reads ` + "`namespace`" + ` and value fields from configuration
2. Appends a new memory row for the current canvas
3. Emits ` + "`memory.added`" + ` with the saved payload`
}

func (c *AddMemory) Icon() string {
	return "database"
}

func (c *AddMemory) Color() string {
	return "blue"
}

func (c *AddMemory) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddMemory) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeString,
			Description: "Memory namespace for this record",
			Required:    true,
		},
		{
			Name:        "valueList",
			Label:       "Values",
			Type:        configuration.FieldTypeList,
			Description: "Fill object fields without writing raw JSON",
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
								Description: "Object field value (can be expression)",
								Required:    true,
							},
						},
					},
				},
			},
		},
	}
}

func (c *AddMemory) Execute(ctx core.ExecutionContext) error {
	var spec Spec
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Namespace = strings.TrimSpace(spec.Namespace)
	if spec.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	values := buildValues(spec)
	metadata := map[string]any{
		"namespace": spec.Namespace,
		"fields":    buildFieldNames(spec, values),
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	if err := ctx.NodeMetadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set node metadata: %w", err)
	}

	if err := ctx.CanvasMemory.Add(spec.Namespace, values); err != nil {
		return fmt.Errorf("failed to add canvas memory: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PayloadType,
		[]any{
			map[string]any{
				"data": map[string]any{
					"namespace": spec.Namespace,
					"values":    values,
				},
			},
		},
	)
}

func buildValues(spec Spec) any {
	if len(spec.ValueList) == 0 {
		return spec.Values
	}

	values := make(map[string]any, len(spec.ValueList))
	for _, pair := range spec.ValueList {
		name := strings.TrimSpace(pair.Name)
		if name == "" {
			continue
		}
		values[name] = pair.Value
	}

	return values
}

func buildFieldNames(spec Spec, values any) []string {
	if len(spec.ValueList) > 0 {
		fields := make([]string, 0, len(spec.ValueList))
		seen := map[string]struct{}{}
		for _, pair := range spec.ValueList {
			name := strings.TrimSpace(pair.Name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			fields = append(fields, name)
		}
		return fields
	}

	valueMap, ok := values.(map[string]any)
	if !ok {
		return []string{}
	}

	fields := make([]string, 0, len(valueMap))
	for name := range valueMap {
		fields = append(fields, name)
	}

	return fields
}

func (c *AddMemory) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AddMemory) Actions() []core.Action {
	return []core.Action{}
}

func (c *AddMemory) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("addMemory does not support actions")
}

func (c *AddMemory) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *AddMemory) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddMemory) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *AddMemory) Cleanup(ctx core.SetupContext) error {
	return nil
}
