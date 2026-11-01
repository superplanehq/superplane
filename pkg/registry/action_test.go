package registry

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/impl"
)

// panickingAction is an action that panics in all panicable methods
type panickingAction struct {
	name string
}

func (p *panickingAction) Name() string                                   { return p.name }
func (p *panickingAction) Label() string                                  { return "Panicking Action" }
func (p *panickingAction) Description() string                            { return "description" }
func (p *panickingAction) Documentation() string                          { return "" }
func (p *panickingAction) Icon() string                                   { return "icon" }
func (p *panickingAction) Color() string                                  { return "red" }
func (p *panickingAction) ExampleOutput() map[string]any                  { return nil }
func (p *panickingAction) Configuration() []configuration.Field           { return nil }
func (p *panickingAction) Hooks() []core.Hook                             { return nil }
func (p *panickingAction) OutputChannels(config any) []core.OutputChannel { return nil }
func (p *panickingAction) Setup(ctx core.SetupContext) error              { panic("setup panic") }
func (p *panickingAction) Execute(ctx core.ExecutionContext) error        { panic("execute panic") }
func (p *panickingAction) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	panic("process queue item panic")
}
func (p *panickingAction) HandleHook(ctx core.ActionHookContext) error {
	panic("handle hook panic")
}
func (p *panickingAction) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	panic("handle webhook panic")
}
func (p *panickingAction) Cancel(ctx core.ExecutionContext) error { panic("cancel panic") }
func (p *panickingAction) Cleanup(ctx core.SetupContext) error    { panic("cleanup panic") }

func TestPanicableAction_Setup_CatchesPanic(t *testing.T) {
	action := &panickingAction{name: "panicking-action"}
	panicable := NewPanicableAction(action)
	ctx := core.SetupContext{
		Logger: log.NewEntry(log.StandardLogger()),
	}

	err := panicable.Setup(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panicking-action panicked in Setup()")
	assert.Contains(t, err.Error(), "setup panic")
}

func TestPanicableAction_Execute_CatchesPanic(t *testing.T) {
	action := &panickingAction{name: "panicking-action"}
	panicable := NewPanicableAction(action)
	ctx := core.ExecutionContext{
		Logger: log.NewEntry(log.StandardLogger()),
	}

	err := panicable.Execute(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panicking-action panicked in Execute()")
	assert.Contains(t, err.Error(), "execute panic")
}

func TestPanicableAction_QueueItemProcessor_CatchesPanic(t *testing.T) {
	action := &panickingAction{name: "panicking-action"}
	panicable := NewPanicableAction(action)
	ctx := core.ProcessQueueContext{}

	processor := panicable.QueueItemProcessor()
	require.NotNil(t, processor)

	id, err := processor.ProcessQueueItem(ctx)

	require.Error(t, err)
	assert.Nil(t, id)
	assert.Contains(t, err.Error(), "action panicked in ProcessQueueItem()")
	assert.Contains(t, err.Error(), "process queue item panic")
}

func TestPanicableAction_QueueItemProcessor_NilWithoutImplementation(t *testing.T) {
	panicable := NewPanicableAction(impl.NewDummyAction(impl.DummyActionOptions{}))

	assert.Nil(t, panicable.QueueItemProcessor())
}

func TestPanicableAction_HandleHook_CatchesPanic(t *testing.T) {
	action := &panickingAction{name: "panicking-action"}
	panicable := NewPanicableAction(action)
	ctx := core.ActionHookContext{
		Name:   "test-hook",
		Logger: log.NewEntry(log.StandardLogger()),
	}

	err := panicable.HandleHook(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panicking-action panicked in HandleHook(test-hook)")
	assert.Contains(t, err.Error(), "handle hook panic")
}

func TestPanicableAction_HandleWebhook_CatchesPanic(t *testing.T) {
	action := &panickingAction{name: "panicking-action"}
	panicable := NewPanicableAction(action)
	ctx := core.WebhookRequestContext{}

	status, _, err := panicable.HandleWebhook(ctx)

	require.Error(t, err)
	assert.Equal(t, 500, status)
	assert.Contains(t, err.Error(), "action panicked in HandleWebhook()")
	assert.Contains(t, err.Error(), "handle webhook panic")
}

func TestPanicableAction_Cancel_CatchesPanic(t *testing.T) {
	action := &panickingAction{name: "panicking-action"}
	panicable := NewPanicableAction(action)
	ctx := core.ExecutionContext{
		Logger: log.NewEntry(log.StandardLogger()),
	}

	err := panicable.Cancel(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panicking-action panicked in Cancel()")
	assert.Contains(t, err.Error(), "cancel panic")
}

func TestPanicableAction_Cleanup_CatchesPanic(t *testing.T) {
	action := &panickingAction{name: "panicking-action"}
	panicable := NewPanicableAction(action)
	ctx := core.SetupContext{
		Logger: log.NewEntry(log.StandardLogger()),
	}

	err := panicable.Cleanup(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panicking-action panicked in Cleanup()")
	assert.Contains(t, err.Error(), "cleanup panic")
}

type validatingAction struct {
	panickingAction
}

func (v *validatingAction) ValidateNodeConfiguration(config map[string]any) error {
	if config["fail"] == true {
		return fmt.Errorf("invalid node configuration")
	}
	return nil
}

func TestPanicableAction_ValidateNodeConfiguration_Forwards(t *testing.T) {
	panicable := NewPanicableAction(&validatingAction{panickingAction: panickingAction{name: "validating-action"}})
	require.NoError(t, panicable.ValidateNodeConfiguration(map[string]any{}))

	err := panicable.ValidateNodeConfiguration(map[string]any{"fail": true})
	require.EqualError(t, err, "invalid node configuration")
}

func TestPanicableAction_ValidateNodeConfiguration_NoOpWhenMissing(t *testing.T) {
	panicable := NewPanicableAction(&panickingAction{name: "plain-action"})
	require.NoError(t, panicable.ValidateNodeConfiguration(nil))
}

type preparingAction struct {
	panickingAction
	prepare func(core.ExecutionContext) (core.PreparedExecution, error)
}

func (a *preparingAction) Prepare(ctx core.ExecutionContext) (core.PreparedExecution, error) {
	return a.prepare(ctx)
}

func TestPanicableAction_ExecutionPreparer_NilWithoutImplementation(t *testing.T) {
	action := NewPanicableAction(impl.NewDummyAction(impl.DummyActionOptions{}))
	require.Nil(t, action.ExecutionPreparer())
}

func TestPanicableAction_ExecutionPreparer_ForwardsContextsAndDispatchError(t *testing.T) {
	preparationID, dispatchID := uuid.New(), uuid.New()
	dispatchFailure := errors.New("dispatch failed")
	var dispatched bool
	action := NewPanicableAction(&preparingAction{
		prepare: func(ctx core.ExecutionContext) (core.PreparedExecution, error) {
			require.Equal(t, preparationID, ctx.ID)
			return func(ctx core.ExecutionContext) error {
				dispatched = true
				require.Equal(t, dispatchID, ctx.ID)
				return dispatchFailure
			}, nil
		},
	})
	preparer := action.ExecutionPreparer()
	require.NotNil(t, preparer)
	dispatch, err := preparer.Prepare(core.ExecutionContext{ID: preparationID})
	require.NoError(t, err)
	require.NotNil(t, dispatch)
	require.False(t, dispatched, "preparation must defer dispatch")
	require.ErrorIs(t, dispatch(core.ExecutionContext{ID: dispatchID}), dispatchFailure)
	require.True(t, dispatched)
}

func TestPanicableAction_ExecutionPreparer_WithoutDispatch(t *testing.T) {
	failure := errors.New("preparation failed")
	for _, test := range []struct {
		name     string
		dispatch core.PreparedExecution
		err      error
	}{
		{name: "no dispatch"},
		{name: "preparation error", err: failure},
		{name: "discard dispatch on error", dispatch: func(core.ExecutionContext) error {
			t.Fatal("failed preparation must not dispatch")
			return nil
		}, err: failure},
	} {
		t.Run(test.name, func(t *testing.T) {
			action := NewPanicableAction(&preparingAction{
				prepare: func(core.ExecutionContext) (core.PreparedExecution, error) {
					return test.dispatch, test.err
				},
			})
			dispatch, err := action.ExecutionPreparer().Prepare(core.ExecutionContext{})
			require.Nil(t, dispatch)
			if test.err != nil {
				require.ErrorIs(t, err, test.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPanicableAction_ExecutionPreparer_CatchesPanics(t *testing.T) {
	for _, phase := range []string{"Prepare", "PreparedExecution"} {
		t.Run(phase, func(t *testing.T) {
			action := NewPanicableAction(&preparingAction{
				panickingAction: panickingAction{name: "preparing-action"},
				prepare: func(core.ExecutionContext) (core.PreparedExecution, error) {
					if phase == "Prepare" {
						panic("preparation panic")
					}
					return func(core.ExecutionContext) error { panic("dispatch panic") }, nil
				},
			})
			ctx := core.ExecutionContext{Logger: log.NewEntry(log.New())}
			dispatch, err := action.ExecutionPreparer().Prepare(ctx)
			if phase == "Prepare" {
				require.Nil(t, dispatch)
				require.EqualError(t, err, "action preparing-action panicked in Prepare(): preparation panic")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, dispatch)
			require.EqualError(t, dispatch(ctx), "action preparing-action panicked in PreparedExecution(): dispatch panic")
		})
	}
}
