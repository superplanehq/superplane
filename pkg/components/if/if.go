package ifp

import (
	"fmt"
	"net/http"
	"time"

	"github.com/expr-lang/expr"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "if"
const ChannelNameTrue = "true"
const ChannelNameFalse = "false"

func init() {
	registry.RegisterComponent(ComponentName, &If{})
}

type If struct{}

type Spec struct {
	Expression string `json:"expression"`
}

func (f *If) Name() string {
	return ComponentName
}

func (f *If) Label() string {
	return "If"
}

func (f *If) Description() string {
	return "Route events based on expression"
}

func (f *If) Icon() string {
	return "split"
}

func (f *If) Color() string {
	return "red"
}

func (f *If) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: "true", Label: "True"},
		{Name: "false", Label: "False"},
	}
}

func (f *If) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "expression",
			Type:        configuration.FieldTypeString,
			Description: "Boolean expression to evaluate",
			Required:    true,
		},
	}
}

func (f *If) Execute(ctx core.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
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
		return err
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
		return ctx.ExecutionStateContext.Emit(
			ChannelNameTrue,
			"if.executed",
			[]any{map[string]any{}},
		)
	}

	return ctx.ExecutionStateContext.Emit(
		ChannelNameFalse,
		"if.executed",
		[]any{map[string]any{}},
	)
}

func (f *If) Actions() []core.Action {
	return []core.Action{}
}

func (f *If) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("if does not support actions")
}

func (f *If) Setup(ctx core.SetupContext) error {
	return nil
}

func (f *If) ProcessQueueItem(ctx core.ProcessQueueContext) (*models.WorkflowNodeExecution, error) {
	return ctx.DefaultProcessing()
}

func (f *If) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (f *If) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
