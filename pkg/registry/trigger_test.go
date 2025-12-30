package registry

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// panickingTrigger is a trigger that panics in all panicable methods
type panickingTrigger struct {
	name string
}

func (p *panickingTrigger) Name() string                         { return p.name }
func (p *panickingTrigger) Label() string                        { return "Panicking Trigger" }
func (p *panickingTrigger) Description() string                  { return "description" }
func (p *panickingTrigger) Icon() string                         { return "icon" }
func (p *panickingTrigger) Color() string                        { return "blue" }
func (p *panickingTrigger) Configuration() []configuration.Field { return nil }
func (p *panickingTrigger) Actions() []core.Action               { return nil }
func (p *panickingTrigger) Setup(ctx core.TriggerContext) error  { panic("setup panic") }
func (p *panickingTrigger) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	panic("handle webhook panic")
}
func (p *panickingTrigger) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	panic("handle action panic")
}

func TestPanicableTrigger_Setup_CatchesPanic(t *testing.T) {
	trig := &panickingTrigger{name: "panicking-trigger"}
	panicable := NewPanicableTrigger(trig)
	ctx := core.TriggerContext{
		Logger: log.NewEntry(log.StandardLogger()),
	}

	err := panicable.Setup(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panicking-trigger panicked in Setup()")
	assert.Contains(t, err.Error(), "setup panic")
}

func TestPanicableTrigger_HandleWebhook_CatchesPanic(t *testing.T) {
	trig := &panickingTrigger{name: "panicking-trigger"}
	panicable := NewPanicableTrigger(trig)
	ctx := core.WebhookRequestContext{}

	status, err := panicable.HandleWebhook(ctx)

	require.Error(t, err)
	assert.Equal(t, 500, status)
	assert.Contains(t, err.Error(), "panicking-trigger panicked in HandleWebhook()")
	assert.Contains(t, err.Error(), "handle webhook panic")
}

func TestPanicableTrigger_HandleAction_CatchesPanic(t *testing.T) {
	trig := &panickingTrigger{name: "panicking-trigger"}
	panicable := NewPanicableTrigger(trig)
	ctx := core.TriggerActionContext{
		Name:   "test-action",
		Logger: log.NewEntry(log.StandardLogger()),
	}

	_, err := panicable.HandleAction(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panicking-trigger panicked in HandleAction(test-action)")
	assert.Contains(t, err.Error(), "handle action panic")
}
