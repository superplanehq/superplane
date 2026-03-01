package deletememory

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

const ComponentName = "deleteMemory"
const PayloadType = "memory.deleted"
const ChannelNameDeleted = "deleted"
const ChannelNameNotFound = "notFound"

func init() {
	registry.RegisterComponent(ComponentName, &DeleteMemory{})
}

type DeleteMemory struct{}

type Spec struct {
	Namespace string      `json:"namespace"`
	MatchList []MatchPair `json:"matchList"`
}

type MatchPair struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

type canvasMemoryDeleteContext interface {
	Delete(namespace string, matches map[string]any) ([]any, error)
}

func (c *DeleteMemory) Name() string {
	return ComponentName
}

func (c *DeleteMemory) Label() string {
	return "Delete Memory"
}

func (c *DeleteMemory) Description() string {
	return "Delete values from canvas memory by namespace and field matches"
}

func (c *DeleteMemory) Documentation() string {
	return `The Delete Memory component removes memory rows from canvas-level memory storage.

## Use Cases

- Remove stale IDs after cleanup is complete
- Keep memory stores bounded over time

## How It Works

1. Reads ` + "`namespace`" + ` and ` + "`matchList`" + ` from configuration
2. Deletes memory rows matching all configured key/value pairs
3. Emits ` + "`memory.deleted`" + ` to the ` + "`deleted`" + ` or ` + "`notFound`" + ` channel

## Output Channels

- **Deleted**: At least one matching memory row was removed
- **Not Found**: No matching memory rows were removed`
}

func (c *DeleteMemory) Icon() string {
	return "database"
}

func (c *DeleteMemory) Color() string {
	return "red"
}

func (c *DeleteMemory) ExampleOutput() map[string]any {
	return exampleOutput()
}

func (c *DeleteMemory) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ChannelNameDeleted, Label: "Deleted"},
		{Name: ChannelNameNotFound, Label: "Not Found"},
	}
}

func (c *DeleteMemory) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeString,
			Description: "Memory namespace to delete from",
			Required:    true,
		},
		{
			Name:        "matchList",
			Label:       "Matches",
			Type:        configuration.FieldTypeList,
			Description: "List of exact field/value matches used for deletion",
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
	}
}

func (c *DeleteMemory) Setup(ctx core.SetupContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	spec = normalizeSpec(spec)
	return validateSpec(spec)
}

func (c *DeleteMemory) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	spec = normalizeSpec(spec)
	if err := validateSpec(spec); err != nil {
		return err
	}

	deleteCtx, ok := ctx.CanvasMemory.(canvasMemoryDeleteContext)
	if !ok {
		return fmt.Errorf("canvas memory delete operations are not supported")
	}

	matches := buildMatches(spec.MatchList)
	deletedValues, deleteErr := deleteCtx.Delete(spec.Namespace, matches)
	if deleteErr != nil {
		return fmt.Errorf("failed to delete canvas memory: %w", deleteErr)
	}

	metadata := map[string]any{
		"namespace": spec.Namespace,
		"fields":    extractFieldNames(spec.MatchList),
		"matches":   matches,
		"count":     len(deletedValues),
	}
	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}
	if err := ctx.NodeMetadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set node metadata: %w", err)
	}

	channel := ChannelNameNotFound
	if len(deletedValues) > 0 {
		channel = ChannelNameDeleted
	}

	return ctx.ExecutionState.Emit(
		channel,
		PayloadType,
		[]any{
			map[string]any{
				"data": map[string]any{
					"namespace": spec.Namespace,
					"matches":   matches,
					"deleted":   deletedValues,
					"count":     len(deletedValues),
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

	matches := buildMatches(spec.MatchList)
	if len(matches) == 0 {
		return fmt.Errorf("at least one memory match is required")
	}

	return nil
}

func buildMatches(matchList []MatchPair) map[string]any {
	matches := make(map[string]any, len(matchList))
	for _, pair := range matchList {
		name := strings.TrimSpace(pair.Name)
		if name == "" {
			continue
		}
		matches[name] = pair.Value
	}
	return matches
}

func extractFieldNames(matchList []MatchPair) []string {
	fields := make([]string, 0, len(matchList))
	seen := map[string]struct{}{}
	for _, pair := range matchList {
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

func (c *DeleteMemory) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteMemory) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteMemory) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("deleteMemory does not support actions")
}

func (c *DeleteMemory) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteMemory) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DeleteMemory) Cleanup(ctx core.SetupContext) error {
	return nil
}
