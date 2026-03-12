package upsertmemory

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

const ComponentName = "upsertMemory"
const PayloadType = "memory.upserted"
const OperationUpdated = "updated"
const OperationCreated = "created"

func init() {
	registry.RegisterComponent(ComponentName, &UpsertMemory{})
}

type UpsertMemory struct{}

type Spec struct {
	Namespace string      `json:"namespace"`
	MatchList []FieldPair `json:"matchList"`
	ValueList []FieldPair `json:"valueList"`
}

type FieldPair struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

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

Always emits to the default channel. Check ` + "`data.operation`" + ` to know whether the component updated existing rows or created a new row.`
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
	return validateSpec(spec)
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
	if err := ctx.NodeMetadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set node metadata: %w", err)
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

func (c *UpsertMemory) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpsertMemory) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("upsertMemory does not support actions")
}

func (c *UpsertMemory) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpsertMemory) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *UpsertMemory) Cleanup(ctx core.SetupContext) error {
	return nil
}
