package blueprint

import (
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/registry"
)

/*
 * Blueprint component encapsulates a reusable workflow as a component-like building block.
 * Blueprints allow users to define complex workflows once and reuse them across multiple places,
 * promoting modularity and maintainability.
 */
type Blueprint struct{}

func init() {
	registry.RegisterComponent("blueprint", &Blueprint{})
}

func (b *Blueprint) IsUserVisible() bool {
	return false
}

/*
 * We have to implement the Name, Label, Description, Icon, Color,
 * but in practice blueprints are configured via their own blueprint definition,
 * not via component-level fields.
 *
 * These things are mostly placeholders to satisfy the Component interface.
 */
func (b *Blueprint) Name() string {
	return "blueprint"
}

func (b *Blueprint) Label() string {
	return "Blueprint"
}

func (b *Blueprint) Description() string {
	return "Blueprint"
}

func (b *Blueprint) Icon() string {
	return "check"
}

func (b *Blueprint) Color() string {
	return "blue"
}

func (b *Blueprint) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (b *Blueprint) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (b *Blueprint) Setup(ctx components.SetupContext) error {
	return nil
}

/*
 * The execution is currently handled by the workflow engine directly,
 * so this is a no-op.
 *
 * TODO: Next step is to wire this up so that the blueprint component
 * actually triggers the execution of the encapsulated workflow, and
 * remove the custom handling in the workflow engine.
 */
func (b *Blueprint) Execute(ctx components.ExecutionContext) error {
	return nil
}

func (b *Blueprint) Actions() []components.Action {
	return []components.Action{}
}

func (b *Blueprint) HandleAction(ctx components.ActionContext) error {
	return nil
}
