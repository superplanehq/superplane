package merge

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterComponent("merge", &Merge{})
}

// Merge is a component that passes its input downstream on
// the default channel. The queue/worker layer is responsible
// for aggregating inputs from multiple parents.
type Merge struct{}

func (m *Merge) Name() string        { return "merge" }
func (m *Merge) Label() string       { return "Merge" }
func (m *Merge) Description() string { return "Merge multiple upstream inputs and forward" }
func (m *Merge) Icon() string        { return "arrow-right-from-line" }
func (m *Merge) Color() string       { return "gray" }

func (m *Merge) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (m *Merge) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (m *Merge) Actions() []components.Action { return []components.Action{} }

func (m *Merge) Setup(ctx components.SetupContext) error {
	return nil
}

func (m *Merge) HandleAction(ctx components.ActionContext) error {
	return fmt.Errorf("merge does not support actions")
}

func (m *Merge) Execute(ctx components.ExecutionContext) error {
	return ctx.ExecutionStateContext.Pass(map[string][]any{
		components.DefaultOutputChannel.Name: {ctx.Data},
	})
}
