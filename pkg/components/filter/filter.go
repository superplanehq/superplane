package filter

import (
	"fmt"
	"net/http"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "filter"

func init() {
	registry.RegisterComponent(ComponentName, &Filter{})
}

type Spec struct {
	Expression string `json:"expression"`
}

type Filter struct{}

func (f *Filter) Name() string {
	return ComponentName
}

func (f *Filter) Label() string {
	return "Filter"
}

func (f *Filter) Description() string {
	return "Filter events based on their content"
}

func (f *Filter) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (f *Filter) Icon() string {
	return "funnel"
}

func (f *Filter) Color() string {
	return "red"
}

func (f *Filter) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "expression",
			Label:       "Filter Expression",
			Type:        configuration.FieldTypeExpression,
			Description: "Boolean expression to filter data",
			Required:    true,
		},
	}
}

func (f *Filter) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Store the expression in metadata so it can be retrieved later
	// even if the node configuration changes
	metadata := map[string]any{
		"expression": spec.Expression,
	}
	err = ctx.Metadata.Set(metadata)
	if err != nil {
		return fmt.Errorf("error setting metadata: %w", err)
	}

	env := map[string]any{
		"$": ctx.Data,
	}

	vm, err := expr.Compile(spec.Expression, []expr.Option{
		expr.Env(env),
		expr.AsBool(),
		expr.WithContext("ctx"),
		expr.Timezone(time.UTC.String()),
	}...)

	if err != nil {
		return fmt.Errorf("expression compilation failed: %w", err)
	}

	output, err := expr.Run(vm, env)
	if err != nil {
		return fmt.Errorf("expression evaluation failed: %w", err)
	}

	matches, ok := output.(bool)
	if !ok {
		return fmt.Errorf("expression must evaluate to boolean, got %T", output)
	}

	if matches {
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"filter.executed",
			[]any{map[string]any{}},
		)
	}

	return ctx.ExecutionState.Pass()
}

func (f *Filter) Actions() []core.Action {
	return []core.Action{}
}

func (f *Filter) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("filter does not support actions")
}

func (f *Filter) Setup(ctx core.SetupContext) error {
	return nil
}

func (f *Filter) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (f *Filter) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (f *Filter) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
