package status

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

const (
	ComponentName          = "taguchiStatus"
	PayloadType            = "taguchi.status"
	ChannelNameSampleMet   = "sampleSizeMet"
	ChannelNamePending     = "pending"
	MemoryKindArm          = "arm"
	MemoryKindTrial        = "trial"
)

func init() {
	registry.RegisterComponent(ComponentName, &Status{})
}

type Status struct{}

type Spec struct {
	ExperimentID string `json:"experimentId"`
	MinPerArm    int    `json:"minPerArm"`
}

func (c *Status) Name() string  { return ComponentName }
func (c *Status) Label() string { return "Taguchi Status" }
func (c *Status) Description() string {
	return "Gate a workflow until every arm has accumulated enough trial samples"
}
func (c *Status) Documentation() string {
	return `Counts trials per arm in canvas memory and emits ` + "`sampleSizeMet`" + ` once every known arm has at least ` + "`minPerArm`" + ` trials. Otherwise emits ` + "`pending`" + ` with per-arm counts so the canvas can loop back / wait.`
}
func (c *Status) Icon() string                  { return "gauge" }
func (c *Status) Color() string                 { return "amber" }
func (c *Status) ExampleOutput() map[string]any { return map[string]any{} }
func (c *Status) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ChannelNameSampleMet, Label: "Sample Size Met"},
		{Name: ChannelNamePending, Label: "Pending"},
	}
}
func (c *Status) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "experimentId",
			Label:       "Experiment ID",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "Experiment namespace",
		},
		{
			Name:        "minPerArm",
			Label:       "Min trials per arm",
			Type:        configuration.FieldTypeNumber,
			Required:    true,
			Default:     30,
			Description: "Minimum completed trials per arm before the gate opens",
		},
	}
}

func (c *Status) Setup(ctx core.SetupContext) error {
	_, err := decode(ctx.Configuration)
	return err
}

func (c *Status) Execute(ctx core.ExecutionContext) error {
	spec, err := decode(ctx.Configuration)
	if err != nil {
		return err
	}

	namespace := "taguchi:" + spec.ExperimentID

	armRows, err := ctx.CanvasMemory.Find(namespace, map[string]any{"kind": MemoryKindArm})
	if err != nil {
		return fmt.Errorf("failed to read arms: %w", err)
	}
	trialRows, err := ctx.CanvasMemory.Find(namespace, map[string]any{"kind": MemoryKindTrial})
	if err != nil {
		return fmt.Errorf("failed to read trials: %w", err)
	}

	counts := map[string]int{}
	for _, row := range armRows {
		m, _ := row.(map[string]any)
		if id, ok := m["arm_id"].(string); ok {
			counts[id] = 0
		}
	}
	for _, row := range trialRows {
		m, _ := row.(map[string]any)
		if id, ok := m["arm_id"].(string); ok {
			counts[id]++
		}
	}

	met := len(counts) > 0
	for _, n := range counts {
		if n < spec.MinPerArm {
			met = false
			break
		}
	}

	channel := ChannelNamePending
	if met {
		channel = ChannelNameSampleMet
	}

	_ = ctx.Metadata.Set(map[string]any{
		"experimentId": spec.ExperimentID,
		"minPerArm":    spec.MinPerArm,
		"counts":       counts,
		"met":          met,
	})

	return ctx.ExecutionState.Emit(channel, PayloadType, []any{
		map[string]any{
			"experimentId": spec.ExperimentID,
			"minPerArm":    spec.MinPerArm,
			"counts":       counts,
			"met":          met,
		},
	})
}

func decode(raw any) (Spec, error) {
	var spec Spec
	if err := mapstructure.Decode(raw, &spec); err != nil {
		return Spec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}
	spec.ExperimentID = strings.TrimSpace(spec.ExperimentID)
	if spec.ExperimentID == "" {
		return Spec{}, fmt.Errorf("experimentId is required")
	}
	if spec.MinPerArm < 1 {
		return Spec{}, fmt.Errorf("minPerArm must be >= 1")
	}
	return spec, nil
}

func (c *Status) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *Status) Actions() []core.Action                  { return nil }
func (c *Status) HandleAction(_ core.ActionContext) error { return fmt.Errorf("%s does not support actions", ComponentName) }
func (c *Status) Cancel(_ core.ExecutionContext) error    { return nil }
func (c *Status) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *Status) Cleanup(_ core.SetupContext) error { return nil }
