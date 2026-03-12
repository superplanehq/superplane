package updatememory

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

const ComponentName = "updateMemory"
const PayloadType = "memory.updated"
const ChannelNameFound = "found"
const ChannelNameNotFound = "notFound"

func init() {
	registry.RegisterComponent(ComponentName, &UpdateMemory{})
}

type UpdateMemory struct{}

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

func (c *UpdateMemory) Name() string {
	return ComponentName
}

func (c *UpdateMemory) Label() string {
	return "Update Memory"
}

func (c *UpdateMemory) Description() string {
	return "Update values in canvas memory by namespace and field matches"
}

func (c *UpdateMemory) Documentation() string {
	return `The Update Memory component updates matching rows in canvas-level memory storage.

## Use Cases

- Patch stored records after external state changes
- Enrich existing memory rows with additional fields
- Keep identifiers and status data in sync

## How It Works

1. Reads ` + "`namespace`" + `, ` + "`matchList`" + `, and ` + "`valueList`" + ` from configuration
2. Updates all matching memory rows in a single SQL operation
3. Emits ` + "`memory.updated`" + ` to the ` + "`found`" + ` or ` + "`notFound`" + ` channel

## Output Channels

- **Found**: At least one matching memory row was updated
- **Not Found**: No matching memory rows were updated`
}

func (c *UpdateMemory) Icon() string {
	return "database"
}

func (c *UpdateMemory) Color() string {
	return "blue"
}

func (c *UpdateMemory) ExampleOutput() map[string]any {
	return exampleOutput()
}

func (c *UpdateMemory) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ChannelNameFound, Label: "Found"},
		{Name: ChannelNameNotFound, Label: "Not Found"},
	}
}

func (c *UpdateMemory) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeString,
			Description: "Memory namespace to update in",
			Required:    true,
		},
		{
			Name:        "matchList",
			Label:       "Matches",
			Type:        configuration.FieldTypeList,
			Description: "List of exact field/value matches used to find rows",
			Required:    true,
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
		{
			Name:        "valueList",
			Label:       "Values",
			Type:        configuration.FieldTypeList,
			Description: "List of fields to set on every matching row",
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
								Description: "Field name to update",
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
	}
}

func (c *UpdateMemory) Setup(ctx core.SetupContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	spec = normalizeSpec(spec)
	return validateSpec(spec)
}

func (c *UpdateMemory) Execute(ctx core.ExecutionContext) error {
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
		return fmt.Errorf("failed to update canvas memory: %w", updateErr)
	}

	metadata := map[string]any{
		"namespace":    spec.Namespace,
		"matchFields":  extractFieldNames(spec.MatchList),
		"valueFields":  extractFieldNames(spec.ValueList),
		"matches":      matches,
		"updatedCount": len(updatedValues),
	}
	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}
	if err := ctx.NodeMetadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set node metadata: %w", err)
	}

	channel := ChannelNameNotFound
	if len(updatedValues) > 0 {
		channel = ChannelNameFound
	}

	return ctx.ExecutionState.Emit(
		channel,
		PayloadType,
		[]any{
			map[string]any{
				"data": map[string]any{
					"namespace": spec.Namespace,
					"matches":   matches,
					"values":    values,
					"updated":   updatedValues,
					"count":     len(updatedValues),
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
	if len(buildPairs(spec.MatchList)) == 0 {
		return fmt.Errorf("at least one memory match is required")
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

func (c *UpdateMemory) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateMemory) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateMemory) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("updateMemory does not support actions")
}

func (c *UpdateMemory) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateMemory) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *UpdateMemory) Cleanup(ctx core.SetupContext) error {
	return nil
}
