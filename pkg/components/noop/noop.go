package noop

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "noop"

func init() {
	registry.RegisterComponent(ComponentName, &NoOp{})
}

type NoOp struct{}

func (c *NoOp) Name() string {
	return ComponentName
}

func (c *NoOp) Label() string {
	return "No Operation"
}

func (c *NoOp) Description() string {
	return "Just pass events through without any additional processing"
}

func (c *NoOp) Icon() string {
	return "check"
}

func (c *NoOp) Color() string {
	return "blue"
}

func (c *NoOp) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (c *NoOp) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{}
}

func (c *NoOp) Execute(ctx components.ExecutionContext) error {
	return ctx.ExecutionStateContext.Pass(map[string][]any{
		components.DefaultOutputChannel.Name: {ctx.Data},
	})
}

func (c *NoOp) Actions() []components.Action {
	return []components.Action{}
}

func (c *NoOp) HandleAction(ctx components.ActionContext) error {
	return fmt.Errorf("noop does not support actions")
}
