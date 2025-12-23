package noop

import (
	"fmt"
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "noop"
const PayloadType = "noop.finished"

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
	return "circle-off"
}

func (c *NoOp) Color() string {
	return "blue"
}

func (c *NoOp) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *NoOp) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (c *NoOp) Execute(ctx core.ExecutionContext) error {
	return ctx.ExecutionStateContext.Emit(
		core.DefaultOutputChannel.Name,
		PayloadType,
		[]any{map[string]any{}},
	)
}

func (c *NoOp) ProcessQueueItem(ctx core.ProcessQueueContext) (*models.WorkflowNodeExecution, error) {
	return ctx.DefaultProcessing()
}

func (c *NoOp) Actions() []core.Action {
	return []core.Action{}
}

func (c *NoOp) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("noop does not support actions")
}

func (c *NoOp) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *NoOp) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *NoOp) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
