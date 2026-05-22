package upsertmemory

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

const ComponentName = "upsertMemory"
const PayloadType = "memory.upserted"
const OperationUpdated = "updated"
const OperationCreated = "created"

func init() {
	registry.RegisterAction(ComponentName, &UpsertMemory{})
}

type UpsertMemory struct{}

type Spec struct {
	Namespace    string      `json:"namespace"`
	MatchList    []FieldPair `json:"matchList"`
	ValueList    []FieldPair `json:"valueList"`
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

type FieldPair = memorywrite.NameValuePair

type canvasMemoryUpdateContext interface {
	Update(namespace string, matches map[string]any, values map[string]any) ([]any, error)
}

func (c *UpsertMemory) Name() string {
	return ComponentName
}

func (c *UpsertMemory) Label() string {
	return "Upsert Memory"
}

func (c *UpsertMemory) Description() string {
	return "Update matching memory rows, or create one when no match exists"
}

func (c *UpsertMemory) Documentation() string {
	return `The Upsert Memory component updates matching rows in canvas-level memory storage, and creates a new row when no matches are found.

## Use Cases

- Keep one record per identifier (for example environment or pull request)
- Replace ad-hoc update-then-add branching with one component
- Persist latest status snapshots with stable matching keys

## How It Works

1. Reads ` + "`namespace`" + `, ` + "`matchList`" + `, and ` + "`valueList`" + ` from configuration
2. Attempts to update all matching memory rows
3. If no rows were updated, inserts a new memory row with the values
4. Emits ` + "`memory.upserted`" + ` to the default channel with ` + "`operation`" + ` set to ` + "`updated`" + ` or ` + "`created`" + `

## Simplified Matching

If ` + "`matchList`" + ` is empty, the component treats the namespace as a singleton record and upserts at namespace level.
This lets you store just one field (for example ` + "`value`" + `) without extra marker fields.

## Output

Always emits to the default channel. Check ` + "`data.operation`" + ` to know whether the component updated existing rows or created a new row.

## List Mode

Enable "Input is a list" to iterate over a list expression and run one upsert per element.
The ` + "`matchList`" + ` is evaluated once (same matches for every iteration), while each
` + "`valueList`" + ` field value is evaluated per element with the iteration variable in scope
(defaults to ` + "`item`" + `). All upserts are reported in a single ` + "`memory.upserted`" + ` event with per-item operation results.

Note: when a non-empty ` + "`matchList`" + ` is configured in list mode, each iteration attempts to
update the same matched rows. List mode is most useful for inserting multiple rows
(empty ` + "`matchList`" + `) or when each item's values are meant to overwrite the matched rows.`
}

func (c *UpsertMemory) Icon() string {
	return "database"
}

func (c *UpsertMemory) Color() string {
	return "blue"
}

func (c *UpsertMemory) ExampleOutput() map[string]any {
	return exampleOutput()
}

func (c *UpsertMemory) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpsertMemory) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeString,
			Description: "Memory namespace to upsert in",
			Required:    true,
		},
		{
			Name:        "iterateList",
			Label:       "Input is a list",
			Type:        configuration.FieldTypeBool,
			Description: "When enabled, iterate over a list expression and upsert once per element (matchList stays global)",
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
			Description: "List of fields to set on matching rows or on the created row",
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
								Description: "Field name to set",
								Required:    true,
							},
							{
								Name:        "value",
								Label:       "Field Value",
								Type:        configuration.FieldTypeExpression,
								Description: "Field value (can be expression)",
								Required:    true,
							},
						},
					},
				},
			},
		},
		{
			Name:        "matchList",
			Label:       "Matches",
			Type:        configuration.FieldTypeList,
			Description: "Optional exact field/value matches used to find rows",
			Required:    false,
			Togglable:   true,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Match",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "name",
								Label:       "Field Name",
								Type:        configuration.FieldTypeString,
								Description: "Field name to match",
								Required:    true,
							},
							{
								Name:        "value",
								Label:       "Field Value",
								Type:        configuration.FieldTypeExpression,
								Description: "Expected field value (can be expression)",
								Required:    true,
							},
						},
					},
				},
			},
		},
	}
}

func (c *UpsertMemory) Setup(ctx core.SetupContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	spec = normalizeSpec(spec)
	if err := validateSpec(spec); err != nil {
		return err
	}
	return spec.listMode().Validate()
}

func (c *UpsertMemory) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	spec = normalizeSpec(spec)
	if err := validateSpec(spec); err != nil {
		return err
	}

	updateCtx, ok := ctx.CanvasMemory.(canvasMemoryUpdateContext)
	if !ok {
		return fmt.Errorf("canvas memory update operations are not supported")
	}

	mode := spec.listMode()
	if err := mode.Validate(); err != nil {
		return err
	}

	if mode.IterateList {
		return executeListMode(ctx, spec, mode, updateCtx)
	}

	matches := buildPairs(spec.MatchList)
	values := buildPairs(spec.ValueList)
	updatedValues, updateErr := updateCtx.Update(spec.Namespace, matches, values)
	if updateErr != nil {
		return fmt.Errorf("failed to upsert canvas memory: %w", updateErr)
	}

	operation := OperationUpdated
	affectedValues := updatedValues

	if len(updatedValues) == 0 {
		if err := ctx.CanvasMemory.Add(spec.Namespace, values); err != nil {
			return fmt.Errorf("failed to upsert canvas memory: %w", err)
		}
		operation = OperationCreated
		affectedValues = []any{values}
	}

	metadata := map[string]any{
		"namespace":    spec.Namespace,
		"matchFields":  extractFieldNames(spec.MatchList),
		"valueFields":  extractFieldNames(spec.ValueList),
		"matches":      matches,
		"operation":    operation,
		"updatedCount": len(updatedValues),
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
					"matches":   matches,
					"values":    values,
					"operation": operation,
					"records":   affectedValues,
					"count":     len(affectedValues),
				},
			},
		},
	)
}

func executeListMode(ctx core.ExecutionContext, spec Spec, mode memorywrite.ListMode, updateCtx canvasMemoryUpdateContext) error {
	items, err := mode.EvaluateList(ctx.Expressions)
	if err != nil {
		return err
	}

	matches := buildPairs(spec.MatchList)
	records := make([]any, 0)
	perItemValues := make([]any, 0, len(items))
	itemResults := make([]any, 0, len(items))
	updatedTotal := 0
	createdTotal := 0

	for i, item := range items {
		scope := mode.Scope(item)
		values, resolveErr := memorywrite.ResolvePairs(spec.ValueList, scope, ctx.Expressions)
		if resolveErr != nil {
			return fmt.Errorf("failed to resolve values for list item %d: %w", i, resolveErr)
		}
		perItemValues = append(perItemValues, values)

		updatedValues, updateErr := updateCtx.Update(spec.Namespace, matches, values)
		if updateErr != nil {
			return fmt.Errorf("failed to upsert canvas memory for list item %d: %w", i, updateErr)
		}

		operation := OperationUpdated
		affected := updatedValues
		if len(updatedValues) == 0 {
			if err := ctx.CanvasMemory.Add(spec.Namespace, values); err != nil {
				return fmt.Errorf("failed to upsert canvas memory for list item %d: %w", i, err)
			}
			operation = OperationCreated
			affected = []any{values}
			createdTotal++
		} else {
			updatedTotal += len(updatedValues)
		}

		records = append(records, affected...)
		itemResults = append(itemResults, map[string]any{
			"operation": operation,
			"values":    values,
			"records":   affected,
		})
	}

	metadata := map[string]any{
		"namespace":    spec.Namespace,
		"matchFields":  extractFieldNames(spec.MatchList),
		"valueFields":  extractFieldNames(spec.ValueList),
		"matches":      matches,
		"updatedCount": updatedTotal,
		"createdCount": createdTotal,
		"iterateList":  true,
		"itemVariable": mode.ItemVariable,
		"count":        len(items),
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
					"matches":   matches,
					"values":    perItemValues,
					"items":     itemResults,
					"records":   records,
					"count":     len(records),
				},
			},
		},
	)
}

func decodeSpec(raw any) (Spec, error) {
	var spec Spec
	if err := mapstructure.Decode(raw, &spec); err != nil {
		return Spec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	return spec, nil
}

func normalizeSpec(spec Spec) Spec {
	spec.Namespace = strings.TrimSpace(spec.Namespace)
	return spec
}

func validateSpec(spec Spec) error {
	if spec.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if len(buildPairs(spec.ValueList)) == 0 {
		return fmt.Errorf("at least one memory value update is required")
	}
	return nil
}

func buildPairs(pairs []FieldPair) map[string]any {
	values := make(map[string]any, len(pairs))
	for _, pair := range pairs {
		name := strings.TrimSpace(pair.Name)
		if name == "" {
			continue
		}
		values[name] = pair.Value
	}
	return values
}

func extractFieldNames(pairs []FieldPair) []string {
	fields := make([]string, 0, len(pairs))
	seen := map[string]struct{}{}
	for _, pair := range pairs {
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

func (c *UpsertMemory) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpsertMemory) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpsertMemory) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpsertMemory) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpsertMemory) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpsertMemory) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
