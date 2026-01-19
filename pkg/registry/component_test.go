package registry

import (
	"testing"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// panickingComponent is a component that panics in all panicable methods
type panickingComponent struct {
	name string
}

func (p *panickingComponent) Name() string                                   { return p.name }
func (p *panickingComponent) Label() string                                  { return "Panicking Component" }
func (p *panickingComponent) Description() string                            { return "description" }
func (p *panickingComponent) Icon() string                                   { return "icon" }
func (p *panickingComponent) Color() string                                  { return "red" }
func (p *panickingComponent) ExampleOutput() map[string]any                  { return nil }
func (p *panickingComponent) Configuration() []configuration.Field           { return nil }
func (p *panickingComponent) Actions() []core.Action                         { return nil }
func (p *panickingComponent) OutputChannels(config any) []core.OutputChannel { return nil }
func (p *panickingComponent) Setup(ctx core.SetupContext) error              { panic("setup panic") }
func (p *panickingComponent) Execute(ctx core.ExecutionContext) error        { panic("execute panic") }
func (p *panickingComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	panic("process queue item panic")
}
func (p *panickingComponent) HandleAction(ctx core.ActionContext) error { panic("handle action panic") }
func (p *panickingComponent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	panic("handle webhook panic")
}
func (p *panickingComponent) Cancel(ctx core.ExecutionContext) error { panic("cancel panic") }

func TestPanicableComponent_Setup_CatchesPanic(t *testing.T) {
	comp := &panickingComponent{name: "panicking-comp"}
	panicable := NewPanicableComponent(comp)
	ctx := core.SetupContext{
		Logger: log.NewEntry(log.StandardLogger()),
	}

	err := panicable.Setup(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panicking-comp panicked in Setup()")
	assert.Contains(t, err.Error(), "setup panic")
}

func TestPanicableComponent_Execute_CatchesPanic(t *testing.T) {
	comp := &panickingComponent{name: "panicking-comp"}
	panicable := NewPanicableComponent(comp)
	ctx := core.ExecutionContext{
		Logger: log.NewEntry(log.StandardLogger()),
	}

	err := panicable.Execute(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panicking-comp panicked in Execute()")
	assert.Contains(t, err.Error(), "execute panic")
}

func TestPanicableComponent_ProcessQueueItem_CatchesPanic(t *testing.T) {
	comp := &panickingComponent{name: "panicking-comp"}
	panicable := NewPanicableComponent(comp)
	ctx := core.ProcessQueueContext{}

	id, err := panicable.ProcessQueueItem(ctx)

	require.Error(t, err)
	assert.Nil(t, id)
	assert.Contains(t, err.Error(), "component panicked in ProcessQueueItem()")
	assert.Contains(t, err.Error(), "process queue item panic")
}

func TestPanicableComponent_HandleAction_CatchesPanic(t *testing.T) {
	comp := &panickingComponent{name: "panicking-comp"}
	panicable := NewPanicableComponent(comp)
	ctx := core.ActionContext{
		Name:   "test-action",
		Logger: log.NewEntry(log.StandardLogger()),
	}

	err := panicable.HandleAction(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panicking-comp panicked in HandleAction(test-action)")
	assert.Contains(t, err.Error(), "handle action panic")
}

func TestPanicableComponent_HandleWebhook_CatchesPanic(t *testing.T) {
	comp := &panickingComponent{name: "panicking-comp"}
	panicable := NewPanicableComponent(comp)
	ctx := core.WebhookRequestContext{}

	status, err := panicable.HandleWebhook(ctx)

	require.Error(t, err)
	assert.Equal(t, 500, status)
	assert.Contains(t, err.Error(), "component panicked in HandleWebhook()")
	assert.Contains(t, err.Error(), "handle webhook panic")
}

func TestPanicableComponent_Cancel_CatchesPanic(t *testing.T) {
	comp := &panickingComponent{name: "panicking-comp"}
	panicable := NewPanicableComponent(comp)
	ctx := core.ExecutionContext{
		Logger: log.NewEntry(log.StandardLogger()),
	}

	err := panicable.Cancel(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panicking-comp panicked in Cancel()")
	assert.Contains(t, err.Error(), "cancel panic")
}
