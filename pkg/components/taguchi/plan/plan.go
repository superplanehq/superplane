package plan

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	taguchi "github.com/marijaaleksic/taguchi"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	ComponentName = "taguchiPlan"
	PayloadType   = "taguchi.plan"
	MemoryKindArm = "arm"
)

func init() {
	registry.RegisterComponent(ComponentName, &Plan{})
}

type Plan struct{}

type Spec struct {
	ExperimentID string   `json:"experimentId"`
	ArrayName    string   `json:"arrayName,omitempty"`
	Factors      []Factor `json:"factors"`
}

type Factor struct {
	Name   string   `json:"name"`
	Levels []string `json:"levels"`
}

func (c *Plan) Name() string  { return ComponentName }
func (c *Plan) Label() string { return "Taguchi Plan" }
func (c *Plan) Description() string {
	return "Pick an orthogonal array for the given factors and emit the arm list"
}
func (c *Plan) Documentation() string {
	return `Defines a Taguchi multi-variate experiment. Given a list of factors and their string-valued levels, this component selects the smallest orthogonal array that fits (L4/L8/L9/L16/L18), generates one arm per row, and emits them all on the ` + "`default`" + ` channel.

Arms are also written to canvas memory under the ` + "`taguchi:{experimentId}`" + ` namespace with ` + "`kind=arm`" + ` so downstream analysis can recover them.

## Output

` + "`default`" + `: ` + "`{experimentId, arrayName, arms: [{arm_id, params}]}`" + `
`
}
func (c *Plan) Icon() string                     { return "flask" }
func (c *Plan) Color() string                    { return "amber" }
func (c *Plan) ExampleOutput() map[string]any    { return exampleOutput() }
func (c *Plan) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *Plan) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "experimentId",
			Label:       "Experiment ID",
			Type:        configuration.FieldTypeExpression,
			Description: "Unique identifier for this experiment (namespaces canvas memory)",
			Required:    true,
		},
		{
			Name:        "arrayName",
			Label:       "Orthogonal Array",
			Type:        configuration.FieldTypeSelect,
			Description: "Optional. Auto-picked from factor counts if omitted.",
			Required:    false,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Auto", Value: ""},
						{Label: "L4", Value: "L4"},
						{Label: "L8", Value: "L8"},
						{Label: "L9", Value: "L9"},
						{Label: "L16", Value: "L16"},
						{Label: "L18", Value: "L18"},
					},
				},
			},
		},
		{
			Name:        "factors",
			Label:       "Factors",
			Type:        configuration.FieldTypeList,
			Description: "Control factors and their levels",
			Required:    true,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Factor",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{Name: "name", Label: "Name", Type: configuration.FieldTypeString, Required: true},
							{Name: "levels", Label: "Levels", Type: configuration.FieldTypeList, Required: true,
								TypeOptions: &configuration.TypeOptions{
									List: &configuration.ListTypeOptions{
										ItemLabel:      "Level",
										ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (c *Plan) Setup(ctx core.SetupContext) error {
	_, err := decodeSpec(ctx.Configuration)
	return err
}

func (c *Plan) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}

	arrayName, arms, err := buildArms(spec)
	if err != nil {
		return err
	}

	namespace := "taguchi:" + spec.ExperimentID
	now := time.Now().UTC().Format(time.RFC3339)
	for _, arm := range arms {
		row := map[string]any{
			"kind":        MemoryKindArm,
			"arm_id":      arm["arm_id"],
			"params":      arm["params"],
			"deployed_at": now,
		}
		if err := ctx.CanvasMemory.Add(namespace, row); err != nil {
			return fmt.Errorf("failed to persist arm to canvas memory: %w", err)
		}
	}

	if err := ctx.Metadata.Set(map[string]any{
		"experimentId": spec.ExperimentID,
		"arrayName":    arrayName,
		"armCount":     len(arms),
	}); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PayloadType,
		[]any{
			map[string]any{
				"experimentId": spec.ExperimentID,
				"arrayName":    arrayName,
				"arms":         arms,
			},
		},
	)
}

func decodeSpec(raw any) (Spec, error) {
	var spec Spec
	if err := mapstructure.Decode(raw, &spec); err != nil {
		return Spec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	spec.ExperimentID = strings.TrimSpace(spec.ExperimentID)
	spec.ArrayName = strings.TrimSpace(spec.ArrayName)
	if spec.ExperimentID == "" {
		return Spec{}, fmt.Errorf("experimentId is required")
	}
	if len(spec.Factors) == 0 {
		return Spec{}, fmt.Errorf("at least one factor is required")
	}
	for i, f := range spec.Factors {
		if strings.TrimSpace(f.Name) == "" {
			return Spec{}, fmt.Errorf("factor %d: name is required", i)
		}
		if len(f.Levels) < 2 {
			return Spec{}, fmt.Errorf("factor %q: at least 2 levels are required", f.Name)
		}
	}
	return spec, nil
}

// buildArms constructs the taguchi experiment, generates trials, and maps
// numeric trial levels back to the original string labels.
func buildArms(spec Spec) (string, []map[string]any, error) {
	controls := make([]taguchi.ControlFactor, len(spec.Factors))
	for i, f := range spec.Factors {
		levels := make([]float64, len(f.Levels))
		for j := range f.Levels {
			levels[j] = float64(j + 1) // library uses 1-indexed levels
		}
		controls[i] = taguchi.ControlFactor{Name: f.Name, Levels: levels}
	}

	arrayName := spec.ArrayName
	if arrayName == "" {
		arrayName = pickArray(spec.Factors)
	}

	exp, err := taguchi.NewExperimentFromFactors(taguchi.LargerTheBetter{}, controls, taguchi.ArrayType(arrayName), nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build experiment: %w", err)
	}

	trials := exp.GenerateTrials()
	arms := make([]map[string]any, 0, len(trials))
	for _, trial := range trials {
		params := map[string]any{}
		for _, f := range spec.Factors {
			raw, ok := trial.Control[f.Name]
			if !ok {
				continue
			}
			idx := int(raw) - 1
			if idx < 0 || idx >= len(f.Levels) {
				// library wraps via modulo for dummy levels; use last level as fallback.
				idx = len(f.Levels) - 1
			}
			params[f.Name] = f.Levels[idx]
		}
		arms = append(arms, map[string]any{
			"arm_id": fmt.Sprintf("arm-%d", trial.ID),
			"params": params,
		})
	}

	return arrayName, arms, nil
}

// pickArray selects the smallest orthogonal array that fits the factor/level counts.
// Very lightweight — prefers L9/L18 for 3-level factors, L4/L8/L16 for 2-level.
func pickArray(factors []Factor) string {
	maxLevels := 0
	numFactors := len(factors)
	for _, f := range factors {
		if len(f.Levels) > maxLevels {
			maxLevels = len(f.Levels)
		}
	}
	if maxLevels <= 2 {
		switch {
		case numFactors <= 3:
			return "L4"
		case numFactors <= 7:
			return "L8"
		default:
			return "L16"
		}
	}
	// 3-level (or mixed with one 2-level factor) → L9 or L18
	if numFactors <= 4 {
		return "L9"
	}
	return "L18"
}

func (c *Plan) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *Plan) Actions() []core.Action                         { return nil }
func (c *Plan) HandleAction(_ core.ActionContext) error        { return fmt.Errorf("%s does not support actions", ComponentName) }
func (c *Plan) Cancel(_ core.ExecutionContext) error           { return nil }
func (c *Plan) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *Plan) Cleanup(_ core.SetupContext) error { return nil }
