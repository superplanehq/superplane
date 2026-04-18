package arms

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
	ComponentName    = "taguchiArms"
	PayloadType      = "taguchi.arm"
	ChannelNameEach  = "each"
	ChannelNameEmpty = "empty"
	MemoryKindArm    = "arm"
)

func init() {
	registry.RegisterComponent(ComponentName, &Arms{})
}

type Arms struct{}

type Spec struct {
	ExperimentID string `json:"experimentId"`
}

func (c *Arms) Name() string                    { return ComponentName }
func (c *Arms) Label() string                   { return "Taguchi Arms" }
func (c *Arms) Description() string {
	return "Fan out one execution per experiment arm"
}
func (c *Arms) Documentation() string {
	return `Reads arms persisted by ` + "`taguchi.plan`" + ` from canvas memory and emits one payload per arm on the ` + "`each`" + ` channel. Downstream nodes (typically ` + "`http`" + `) receive one execution per arm to deploy each variant.`
}
func (c *Arms) Icon() string                  { return "split" }
func (c *Arms) Color() string                 { return "amber" }
func (c *Arms) ExampleOutput() map[string]any { return map[string]any{} }
func (c *Arms) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ChannelNameEach, Label: "Each"},
		{Name: ChannelNameEmpty, Label: "Empty"},
	}
}
func (c *Arms) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "experimentId",
			Label:       "Experiment ID",
			Type:        configuration.FieldTypeExpression,
			Description: "Experiment namespace to read arms from",
			Required:    true,
		},
	}
}

func (c *Arms) Setup(ctx core.SetupContext) error {
	_, err := decode(ctx.Configuration)
	return err
}

func (c *Arms) Execute(ctx core.ExecutionContext) error {
	spec, err := decode(ctx.Configuration)
	if err != nil {
		return err
	}

	rows, err := ctx.CanvasMemory.Find("taguchi:"+spec.ExperimentID, map[string]any{"kind": MemoryKindArm})
	if err != nil {
		return fmt.Errorf("failed to read arms: %w", err)
	}

	if len(rows) == 0 {
		return ctx.ExecutionState.Emit(ChannelNameEmpty, PayloadType, []any{map[string]any{"experimentId": spec.ExperimentID}})
	}

	payloads := make([]any, 0, len(rows))
	for i, row := range rows {
		m, _ := row.(map[string]any)
		payloads = append(payloads, map[string]any{
			"experimentId": spec.ExperimentID,
			"arm_id":       m["arm_id"],
			"params":       m["params"],
			"index":        i,
			"totalCount":   len(rows),
		})
	}

	_ = ctx.Metadata.Set(map[string]any{
		"experimentId": spec.ExperimentID,
		"armCount":     len(rows),
	})

	return ctx.ExecutionState.Emit(ChannelNameEach, PayloadType, payloads)
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
	return spec, nil
}

func (c *Arms) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
func (c *Arms) Actions() []core.Action                  { return nil }
func (c *Arms) HandleAction(_ core.ActionContext) error { return fmt.Errorf("%s does not support actions", ComponentName) }
func (c *Arms) Cancel(_ core.ExecutionContext) error    { return nil }
func (c *Arms) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *Arms) Cleanup(_ core.SetupContext) error { return nil }
