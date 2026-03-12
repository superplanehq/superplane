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
const ResultModeAll = "all"
const ResultModeLatest = "latest"
const EmitModeAllAtOnce = "allAtOnce"
const EmitModeOneByOne = "oneByOne"
const ChannelNameFound = "found"
const ChannelNameNotFound = "notFound"

func init() {
	registry.RegisterComponent(ComponentName, &ReadMemory{})
}

type ReadMemory struct{}

type Spec struct {
	Namespace  string      `json:"namespace"`
	ResultMode string      `json:"resultMode,omitempty"`
	EmitMode   string      `json:"emitMode,omitempty"`
	MatchList  []MatchPair `json:"matchList"`
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

1. Reads ` + "`namespace`" + `, ` + "`resultMode`" + `, ` + "`emitMode`" + `, and ` + "`matchList`" + ` from configuration
2. Finds memory rows matching all configured key/value pairs
3. Emits ` + "`memory.read`" + ` to the ` + "`found`" + ` or ` + "`notFound`" + ` channel

## Output Channels

- **Found**: At least one matching memory row was found
- **Not Found**: No matching memory rows were found`

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
	return []core.OutputChannel{
		{Name: ChannelNameFound, Label: "Found"},
		{Name: ChannelNameNotFound, Label: "Not Found"},
	}
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
			Name:        "resultMode",
			Label:       "Result Mode",
			Type:        configuration.FieldTypeSelect,
			Description: "Choose whether to return all matches or only the latest match",
			Required:    true,
			Default:     ResultModeAll,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "All Matches", Value: ResultModeAll},
						{Label: "Latest Match", Value: ResultModeLatest},
					},
				},
			},
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
		{
			Name:        "emitMode",
			Label:       "Emit Mode",
			Type:        configuration.FieldTypeSelect,
			Description: "Choose whether list results are emitted as a single event or one event per record",
			Required:    false,
			Default:     EmitModeAllAtOnce,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "All At Once", Value: EmitModeAllAtOnce},
						{Label: "One By One", Value: EmitModeOneByOne},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "resultMode", Values: []string{ResultModeAll}},
			},
		},
	}
}

func (c *ReadMemory) Setup(ctx core.SetupContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	spec = normalizeSpec(spec)

	return validateSpec(spec)
}

func (c *ReadMemory) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	spec = normalizeSpec(spec)
	if err := validateSpec(spec); err != nil {
		return err
	}

	matches := buildMatches(spec.MatchList)
	values := []any{}
	if spec.ResultMode == ResultModeLatest {
		value, findErr := ctx.CanvasMemory.FindFirst(spec.Namespace, matches)
		if findErr != nil {
			return fmt.Errorf("failed to read canvas memory: %w", findErr)
		}
		if value != nil {
			values = append(values, value)
		}
	} else {
		var findErr error
		values, findErr = ctx.CanvasMemory.Find(spec.Namespace, matches)
		if findErr != nil {
			return fmt.Errorf("failed to read canvas memory: %w", findErr)
		}
	}

	metadata := map[string]any{
		"namespace":  spec.Namespace,
		"fields":     extractFieldNames(spec.MatchList),
		"matches":    matches,
		"resultMode": spec.ResultMode,
		"emitMode":   spec.EmitMode,
		"count":      len(values),
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	if err := ctx.NodeMetadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set node metadata: %w", err)
	}

	channel := ChannelNameNotFound
	if len(values) > 0 {
		channel = ChannelNameFound
	}

	payloads := buildPayloads(spec, matches, values)

	return ctx.ExecutionState.Emit(
		channel,
		PayloadType,
		payloads,
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
	spec.ResultMode = strings.TrimSpace(spec.ResultMode)
	if spec.ResultMode == "" {
		spec.ResultMode = ResultModeAll
	}
	spec.EmitMode = strings.TrimSpace(spec.EmitMode)
	if spec.EmitMode == "" || spec.ResultMode == ResultModeLatest {
		spec.EmitMode = EmitModeAllAtOnce
	}
	return spec
}

func validateSpec(spec Spec) error {
	if spec.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if spec.ResultMode != ResultModeAll && spec.ResultMode != ResultModeLatest {
		return fmt.Errorf("resultMode must be either %q or %q", ResultModeAll, ResultModeLatest)
	}
	if spec.EmitMode != EmitModeAllAtOnce && spec.EmitMode != EmitModeOneByOne {
		return fmt.Errorf("emitMode must be either %q or %q", EmitModeAllAtOnce, EmitModeOneByOne)
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

func buildPayloads(spec Spec, matches map[string]any, values []any) []any {
	if spec.ResultMode == ResultModeAll && spec.EmitMode == EmitModeOneByOne && len(values) > 0 {
		payloads := make([]any, 0, len(values))
		for i, value := range values {
			payloads = append(payloads, map[string]any{
				"data": map[string]any{
					"namespace":  spec.Namespace,
					"matches":    matches,
					"resultMode": spec.ResultMode,
					"emitMode":   spec.EmitMode,
					"values":     []any{value},
					"count":      1,
					"index":      i,
					"totalCount": len(values),
				},
			})
		}
		return payloads
	}

	return []any{
		map[string]any{
			"data": map[string]any{
				"namespace":  spec.Namespace,
				"matches":    matches,
				"resultMode": spec.ResultMode,
				"emitMode":   spec.EmitMode,
				"values":     values,
				"count":      len(values),
			},
		},
	}
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
