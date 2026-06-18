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

type canvasMemoryUpsertContext interface {
	Update(namespace string, matches map[string]any, values map[string]any) ([]any, error)
	UpdateNamespace(namespace string, values map[string]any) ([]any, error)
}

type canvasMemoryUpsertRecordsContext interface {
	AddRecord(namespace string, values any) (core.CanvasMemoryRecord, error)
	UpdateRecords(namespace string, matches map[string]any, values map[string]any) ([]core.CanvasMemoryRecord, error)
	UpdateNamespaceRecords(namespace string, values map[string]any) ([]core.CanvasMemoryRecord, error)
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
Both ` + "`matchList`" + ` and ` + "`valueList`" + ` field values are evaluated per element with the iteration
variable in scope (defaults to ` + "`item`" + `), so expressions like ` + "`item.uuid`" + ` resolve to the current
element's value on every iteration. This means each item upserts against its own matched row.
All upserts are reported in a single ` + "`memory.upserted`" + ` event with per-item operation results.

To use a single global match across iterations, write a ` + "`{{ ... }}`" + ` template (resolved before
iteration starts) instead of a bare expression.`
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
			Description: "When enabled, iterate over a list expression and upsert once per element. Both matchList and valueList values are evaluated per element with the iteration variable in scope (e.g. item.uuid)",
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

	upsertCtx, ok := ctx.CanvasMemory.(canvasMemoryUpsertContext)
	if !ok {
		return fmt.Errorf("canvas memory update operations are not supported")
	}

	mode := spec.listMode()
	if err := mode.Validate(); err != nil {
		return err
	}

	if mode.IterateList {
		return executeListMode(ctx, spec, mode, upsertCtx)
	}

	matches := buildPairs(spec.MatchList)
	values := buildPairs(spec.ValueList)
	operation, affectedValues, upsertErr := upsertMemoryRow(ctx, spec.Namespace, matches, values, upsertCtx, false)
	if upsertErr != nil {
		return fmt.Errorf("failed to upsert canvas memory: %w", upsertErr)
	}

	updatedCount := 0
	if operation == OperationUpdated {
		updatedCount = len(affectedValues)
	}

	metadata := map[string]any{
		"namespace":    spec.Namespace,
		"matchFields":  memorywrite.FieldNames(spec.MatchList),
		"valueFields":  memorywrite.FieldNames(spec.ValueList),
		"matches":      matches,
		"operation":    operation,
		"updatedCount": updatedCount,
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

func executeListMode(ctx core.ExecutionContext, spec Spec, mode memorywrite.ListMode, upsertCtx canvasMemoryUpsertContext) error {
	items, err := mode.EvaluateList(ctx.Expressions)
	if err != nil {
		return err
	}

	insertOnly := len(memorywrite.FieldNames(spec.MatchList)) == 0

	resolvedMatches, err := memorywrite.ResolveAllItemMatches(items, mode, spec.MatchList, ctx.Expressions)
	if err != nil {
		return err
	}

	resolved, err := memorywrite.ResolveAllItemValues(items, mode, spec.ValueList, ctx.Expressions)
	if err != nil {
		return err
	}

	recordCtx, ok := upsertCtx.(canvasMemoryUpsertRecordsContext)
	if !ok {
		return fmt.Errorf("canvas memory record upsert operations are not supported")
	}

	return executeListModeWithRecords(ctx, spec, mode, recordCtx, resolvedMatches, insertOnly, resolved)
}

func executeListModeWithRecords(
	ctx core.ExecutionContext,
	spec Spec,
	mode memorywrite.ListMode,
	upsertCtx canvasMemoryUpsertRecordsContext,
	resolvedMatches []map[string]any,
	insertOnly bool,
	resolved []map[string]any,
) error {
	records := make([]core.CanvasMemoryRecord, 0)
	recordPositions := map[uuid.UUID]int{}
	updatedRecords := make([]core.CanvasMemoryRecord, 0)
	updatedPositions := map[uuid.UUID]int{}
	createdRecords := make([]core.CanvasMemoryRecord, 0)
	createdPositions := map[uuid.UUID]int{}
	perItemValues := make([]any, 0, len(resolved))
	itemResults := make([]any, 0, len(resolved))

	for i, values := range resolved {
		perItemValues = append(perItemValues, values)

		var matches map[string]any
		if i < len(resolvedMatches) {
			matches = resolvedMatches[i]
		}

		operation, affected, upsertErr := upsertMemoryRecord(spec.Namespace, matches, values, upsertCtx, insertOnly)
		if upsertErr != nil {
			return fmt.Errorf("failed to upsert canvas memory for list item %d: %w", i, upsertErr)
		}

		records = memorywrite.AppendUniqueRecords(records, recordPositions, affected)
		if operation == OperationCreated {
			createdRecords = memorywrite.AppendUniqueRecords(createdRecords, createdPositions, affected)
		} else {
			updatedRecords = memorywrite.AppendUniqueRecords(updatedRecords, updatedPositions, affected)
		}

		itemResults = append(itemResults, map[string]any{
			"operation": operation,
			"matches":   matches,
			"values":    values,
			"records":   memorywrite.RecordValues(affected),
		})
	}

	recordValues := memorywrite.RecordValues(records)
	metadata := map[string]any{
		"namespace":    spec.Namespace,
		"matchFields":  memorywrite.FieldNames(spec.MatchList),
		"valueFields":  memorywrite.FieldNames(spec.ValueList),
		"updatedCount": len(updatedRecords),
		"createdCount": len(createdRecords),
		"iterateList":  true,
		"itemVariable": mode.ItemVariable,
		"count":        len(resolved),
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
					"values":    perItemValues,
					"items":     itemResults,
					"records":   recordValues,
					"count":     len(recordValues),
				},
			},
		},
	)
}

func upsertMemoryRow(
	ctx core.ExecutionContext,
	namespace string,
	matches map[string]any,
	values map[string]any,
	upsertCtx canvasMemoryUpsertContext,
	insertOnly bool,
) (operation string, affected []any, err error) {
	if insertOnly {
		if err := ctx.CanvasMemory.Add(namespace, values); err != nil {
			return "", nil, err
		}
		return OperationCreated, []any{values}, nil
	}

	var updated []any
	if len(matches) == 0 {
		updated, err = upsertCtx.UpdateNamespace(namespace, values)
	} else {
		updated, err = upsertCtx.Update(namespace, matches, values)
	}
	if err != nil {
		return "", nil, err
	}
	if len(updated) > 0 {
		return OperationUpdated, updated, nil
	}

	if err := ctx.CanvasMemory.Add(namespace, values); err != nil {
		return "", nil, err
	}
	return OperationCreated, []any{values}, nil
}

func upsertMemoryRecord(
	namespace string,
	matches map[string]any,
	values map[string]any,
	upsertCtx canvasMemoryUpsertRecordsContext,
	insertOnly bool,
) (operation string, affected []core.CanvasMemoryRecord, err error) {
	if insertOnly {
		record, err := upsertCtx.AddRecord(namespace, values)
		if err != nil {
			return "", nil, err
		}
		return OperationCreated, []core.CanvasMemoryRecord{record}, nil
	}

	var updated []core.CanvasMemoryRecord
	if len(matches) == 0 {
		updated, err = upsertCtx.UpdateNamespaceRecords(namespace, values)
	} else {
		updated, err = upsertCtx.UpdateRecords(namespace, matches, values)
	}
	if err != nil {
		return "", nil, err
	}
	if len(updated) > 0 {
		return OperationUpdated, updated, nil
	}

	record, err := upsertCtx.AddRecord(namespace, values)
	if err != nil {
		return "", nil, err
	}
	return OperationCreated, []core.CanvasMemoryRecord{record}, nil
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
