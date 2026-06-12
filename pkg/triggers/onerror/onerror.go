package onerror

import (
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

// TriggerName is the registered name of the On Error trigger.
const TriggerName = "onError"

// PayloadType is the event type emitted whenever the trigger fires.
const PayloadType = "onError.triggered"

func init() {
	registry.RegisterTrigger(TriggerName, &OnError{})
}

// OnError is a canvas-wide error handler. It is not wired through edges like a
// regular trigger; instead, whenever any execution in the canvas finishes with
// an "error" result (the component could not execute), the failure path emits a
// root event on every OnError node in that canvas. See contexts.DispatchOnError.
type OnError struct{}

func (t *OnError) Name() string {
	return TriggerName
}

func (t *OnError) Label() string {
	return "On Error"
}

func (t *OnError) Description() string {
	return "Run a workflow whenever any node in this canvas errors out"
}

func (t *OnError) Documentation() string {
	return `The On Error trigger reacts to unexpected failures anywhere in the canvas.

## How It Works

Whenever any node in the same canvas finishes in the **error** state (it could not
execute - e.g. a network failure, invalid credentials, or an unhandled exception),
every On Error node in that canvas fires a new run. This happens independently of how
your nodes are connected: On Error nodes do not need an incoming connection.

If you place multiple On Error nodes on a canvas, all of them receive every error.

> Note: this reacts to the **error** state, not the **failed** state. A component that
> executed successfully but produced a failure outcome (e.g. an HTTP request returning
> 404, or a rejected approval) does **not** trigger On Error. Use a Filter or the
> component's failure output channel for those cases.

## Event Data

Each emitted event carries:

- ` + "`node`" + `: the node that errored - its ` + "`id`" + `, ` + "`name`" + `, and ` + "`component`" + ` type
- ` + "`error`" + `: the failure ` + "`reason`" + ` and human-readable ` + "`message`" + `
- ` + "`run`" + `: the ` + "`id`" + ` of the run that failed
- ` + "`root`" + `: the event that started the failed run - the originating ` + "`node`" + ` and
  its ` + "`payload`" + `
- ` + "`payloads`" + `: the payloads emitted by every upstream node in the failed run,
  keyed by node name

## Examples

- ` + "`$['On Error'].node.name`" + `: the name of the node that errored
- ` + "`$['On Error'].error.message`" + `: the error message
- ` + "`$['On Error'].root.payload.data.version`" + `: data from the event that triggered the failed run
- ` + "`$['On Error'].payloads['Build'].data.version`" + `: data from an upstream node in the failed run`
}

func (t *OnError) Icon() string {
	return "triangle-alert"
}

func (t *OnError) Color() string {
	return "red"
}

func (t *OnError) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *OnError) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (t *OnError) Setup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnError) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnError) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnError) Cleanup(ctx core.TriggerContext) error {
	return nil
}
