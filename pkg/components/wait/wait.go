package wait

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
)

type Wait struct{}

type Spec struct {
	Until string `json:"until"`
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

func (w *Wait) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (w *Wait) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:     "until",
			Label:    "Wait Until",
			Type:     components.FieldTypeString,
			Required: true,
		},
	}
}

func (w *Wait) Execute(ctx components.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	until, err := time.Parse(time.RFC3339, spec.Until)
	if err != nil {
		return fmt.Errorf("error parsing until time: %v", err)
	}

	//
	// If we haven't reached the until time, schedule an action to run after the wait time.
	//
	if time.Now().Before(until) {
		return ctx.RequestContext.ScheduleActionCall("timeReached", map[string]any{}, time.Until(until))
	}

	//
	// Otherwise, just complete the execution.
	//
	return ctx.ExecutionStateContext.Pass(map[string][]any{
		components.DefaultOutputChannel.Name: {ctx.Data},
	})
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
		//
		// TODO: we need to complete the execution with the proper data.
		//
		return ctx.ExecutionStateContext.Pass(map[string][]any{
			components.DefaultOutputChannel.Name: {map[string]any{}},
		})

	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}
