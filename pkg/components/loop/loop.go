package loop

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "loop"
const PayloadType = "loop.finished"

func init() {
	registry.RegisterComponent(ComponentName, &Loop{})
}

type Loop struct{}

func (l *Loop) Name() string {
	return ComponentName
}

func (l *Loop) Label() string {
	return "Loop"
}

func (l *Loop) Description() string {
	return "Repeat nested steps within a loop container"
}

func (l *Loop) Icon() string {
	return "repeat"
}

func (l *Loop) Color() string {
	return "sky"
}

func (l *Loop) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (l *Loop) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (l *Loop) Execute(ctx core.ExecutionContext) error {
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PayloadType,
		[]any{map[string]any{}},
	)
}

func (l *Loop) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (l *Loop) Actions() []core.Action {
	return []core.Action{}
}

func (l *Loop) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("loop does not support actions")
}

func (l *Loop) Setup(ctx core.SetupContext) error {
	return nil
}

func (l *Loop) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (l *Loop) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
