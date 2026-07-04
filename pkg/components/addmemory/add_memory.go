package addmemory

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components/memorywrite"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "addMemory"
const PayloadType = "memory.added"

func init() {
	registry.RegisterAction(ComponentName, &AddMemory{})
}

type AddMemory struct{}

type Spec struct {
	Namespace    string      `json:"namespace"`
	Values       any         `json:"values,omitempty"`
	ValueList    []ValuePair `json:"valueList,omitempty"`
	IterateList  bool        `json:"iterateList,omitempty"`
	ListSource   string      `json:"listSource,omitempty"`
	ItemVariable string      `json:"itemVariable,omitempty"`
}

func (s Spec) listMode() memorywrite.ListMode {
	return memorywrite.ListMode{
		IterateList:  s.IterateList,
		ListSource:   s.ListSource,
		ItemVariable: s.ItemVariable,
	}.Normalize()
}

type ValuePair = memorywrite.NameValuePair

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
	return `The Add Memory component appends one or more items to canvas-level memory storage.

## Use Cases

- Persist identifiers for later cleanup paths
- Store cross-run mappings (for example pull request to resource ID)
- Keep structured operational context per canvas

## How It Works

1. Reads ` + "`namespace`" + ` and value fields from configuration
2. Appends a new memory row for the current canvas
3. Emits ` + "`memory.added`" + ` with the saved payload

## List Mode

Enable "Input is a list" to add one memory row per element of a list expression.
Configure ` + "`listSource`" + ` (the expression that yields the list) and ` + "`itemVariable`" + `
(the name to reference each element in field-value expressions, defaults to ` + "`item`" + `).

A single ` + "`memory.added`" + ` event is emitted with all written rows and the total ` + "`count`" + `.`
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
			Name:        "iterateList",
			Label:       "Input is a list",
			Type:        configuration.FieldTypeBool,
			Description: "When enabled, iterate over a list expression and add one memory row per element",
		},
		{
			Name:        "listSource",
			Label:       "List Source",
			Type:        configuration.FieldTypeExpression,
			Description: "Expression that evaluates to a list, e.g. $[\"Runner\"].data.result.services",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "iterateList", Values: []string{"true"}},
			},
		},
		{
			Name:        "itemVariable",
			Label:       "Item Variable",
			Type:        configuration.FieldTypeString,
			Description: "Variable name bound to each list element when evaluating field values (default: item)",
			Default:     memorywrite.DefaultItemVariable,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "iterateList", Values: []string{"true"}},
			},
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
								Description: "Object field value (can be expression). In list mode, reference each element with the iteration variable, e.g. item.name",
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

	mode := spec.listMode()
	if err := mode.Validate(); err != nil {
		return err
	}

	if mode.IterateList {
		return c.executeListMode(ctx, spec, mode)
	}

	values := buildValues(spec)
	metadata := map[string]any{
		"namespace": spec.Namespace,
		"fields":    buildFieldNames(spec, values),
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
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

func (c *AddMemory) executeListMode(ctx core.ExecutionContext, spec Spec, mode memorywrite.ListMode) error {
	items, err := mode.EvaluateList(ctx.Expressions)
	if err != nil {
		return err
	}

	resolved, err := memorywrite.ResolveAllItemValues(items, mode, spec.ValueList, ctx.Expressions)
	if err != nil {
		return err
	}

	writtenValues := make([]any, 0, len(resolved))
	for i, values := range resolved {
		if err := ctx.CanvasMemory.Add(spec.Namespace, values); err != nil {
			return fmt.Errorf("failed to add canvas memory for list item %d: %w", i, err)
		}
		writtenValues = append(writtenValues, values)
	}

	metadata := map[string]any{
		"namespace":    spec.Namespace,
		"fields":       memorywrite.FieldNames(spec.ValueList),
		"iterateList":  true,
		"itemVariable": mode.ItemVariable,
		"count":        len(writtenValues),
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PayloadType,
		[]any{
			map[string]any{
				"data": map[string]any{
					"namespace": spec.Namespace,
					"values":    writtenValues,
					"count":     len(writtenValues),
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
		return memorywrite.FieldNames(spec.ValueList)
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

func (c *AddMemory) Setup(ctx core.SetupContext) error {
	var spec Spec
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	return spec.listMode().Validate()
}

func (c *AddMemory) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddMemory) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *AddMemory) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AddMemory) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AddMemory) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
