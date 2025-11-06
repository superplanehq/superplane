package wait

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterComponent("wait", &Wait{})
}

type Wait struct{}

type Spec struct {
	Duration Duration `json:"duration"`
}

type Duration struct {
	Value int    `json:"value"`
	Unit  string `json:"unit"`
}

func (w *Wait) Name() string {
	return "wait"
}

func (w *Wait) Label() string {
	return "Wait"
}

func (w *Wait) Description() string {
	return "Wait for a certain amount of time"
}

func (w *Wait) Icon() string {
	return "alarm-clock"
}

func (w *Wait) Color() string {
	return "yellow"
}

func (w *Wait) IsUserVisible() bool { return true }

func (w *Wait) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (w *Wait) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "duration",
			Label:    "Set wait interval",
			Type:     configuration.FieldTypeObject,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: []configuration.Field{
						{
							Name:     "value",
							Label:    "How long should I wait?",
							Type:     configuration.FieldTypeNumber,
							Required: true,
						},
						{
							Name:     "unit",
							Label:    "Unit",
							Type:     configuration.FieldTypeSelect,
							Required: true,
							TypeOptions: &configuration.TypeOptions{
								Select: &configuration.SelectTypeOptions{
									Options: []configuration.FieldOption{
										{
											Label: "Seconds",
											Value: "seconds",
										},
										{
											Label: "Minutes",
											Value: "minutes",
										},
										{
											Label: "Hours",
											Value: "hours",
										},
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

func (w *Wait) Execute(ctx components.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	interval := findInterval(spec)
	if interval == 0 {
		return fmt.Errorf("invalid interval: %v", spec.Duration)
	}

	return ctx.RequestContext.ScheduleActionCall("timeReached", map[string]any{}, interval)
}

func (w *Wait) Actions() []components.Action {
	return []components.Action{
		{
			Name: "timeReached",
		},
	}
}

func (w *Wait) HandleAction(ctx components.ActionContext) error {
	switch ctx.Name {
	case "timeReached":
		return ctx.ExecutionStateContext.Pass(map[string][]any{
			components.DefaultOutputChannel.Name: {map[string]any{}},
		})

	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (w *Wait) Setup(ctx components.SetupContext) error {
	return nil
}

func findInterval(spec Spec) time.Duration {
	switch spec.Duration.Unit {
	case "seconds":
		return time.Duration(spec.Duration.Value) * time.Second
	case "minutes":
		return time.Duration(spec.Duration.Value) * time.Minute
	case "hours":
		return time.Duration(spec.Duration.Value) * time.Hour
	default:
		return 0
	}
}
