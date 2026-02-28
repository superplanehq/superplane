package readmemory

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

const ComponentName = "readMemory"
const PayloadType = "memory.read"

func init() {
	registry.RegisterComponent(ComponentName, &ReadMemory{})
}

type ReadMemory struct{}

type Spec struct {
	Namespace string      `json:"namespace"`
	MatchList []MatchPair `json:"matchList"`
}

type MatchPair struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

func (c *ReadMemory) Name() string {
	return ComponentName
}

func (c *ReadMemory) Label() string {
	return "Read Memory"
}

func (c *ReadMemory) Description() string {
	return "Find values from canvas memory by namespace and field matches"
}

func (c *ReadMemory) Documentation() string {
	return `The Read Memory component looks up values from canvas-level memory storage.

## Use Cases

- Retrieve previously stored IDs before cleanup actions
- Check whether related data already exists
- Rehydrate context from prior runs

## How It Works

1. Reads ` + "`namespace`" + ` and ` + "`matchList`" + ` from configuration
2. Finds memory rows matching all configured key/value pairs
3. Emits ` + "`memory.read`" + ` with matching values`
}

func (c *ReadMemory) Icon() string {
	return "database"
}

func (c *ReadMemory) Color() string {
	return "purple"
}

func (c *ReadMemory) ExampleOutput() map[string]any {
	return exampleOutput()
}

func (c *ReadMemory) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ReadMemory) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "namespace",
			Label:       "Namespace",
			Type:        configuration.FieldTypeString,
			Description: "Memory namespace to search in",
			Required:    true,
		},
		{
			Name:        "matchList",
			Label:       "Matches",
			Type:        configuration.FieldTypeList,
			Description: "List of exact field/value matches used for lookup",
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

func (c *ReadMemory) Setup(ctx core.SetupContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	spec.Namespace = strings.TrimSpace(spec.Namespace)

	return validateSpec(spec)
}

func (c *ReadMemory) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	spec.Namespace = strings.TrimSpace(spec.Namespace)
	if err := validateSpec(spec); err != nil {
		return err
	}

	matches := buildMatches(spec.MatchList)
	values, err := ctx.CanvasMemory.Find(spec.Namespace, matches)
	if err != nil {
		return fmt.Errorf("failed to read canvas memory: %w", err)
	}

	metadata := map[string]any{
		"namespace": spec.Namespace,
		"matches":   matches,
		"count":     len(values),
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
					"count":     len(values),
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

func (c *ReadMemory) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ReadMemory) Actions() []core.Action {
	return []core.Action{}
}

func (c *ReadMemory) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("readMemory does not support actions")
}

func (c *ReadMemory) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ReadMemory) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *ReadMemory) Cleanup(ctx core.SetupContext) error {
	return nil
}
