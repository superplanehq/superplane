package registry

import (
	"fmt"
	"runtime/debug"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

/*
 * PanicableComponent wraps a Component implementation
 * around logic for handling panics.
 */
type PanicableComponent struct {
	underlying core.Component
}

func NewPanicableComponent(c core.Component) core.Component {
	return &PanicableComponent{underlying: c}
}

/*
 * Non-panicking methods.
 * These are mostly definition methods, so they won't panic.
 */
func (s *PanicableComponent) Name() string {
	return s.underlying.Name()
}

func (s *PanicableComponent) Label() string {
	return s.underlying.Label()
}

func (s *PanicableComponent) Description() string {
	return s.underlying.Description()
}

func (s *PanicableComponent) Icon() string {
	return s.underlying.Icon()
}

func (s *PanicableComponent) Color() string {
	return s.underlying.Color()
}

func (s *PanicableComponent) Configuration() []configuration.Field {
	return s.underlying.Configuration()
}

func (s *PanicableComponent) Actions() []core.Action {
	return s.underlying.Actions()
}

func (s *PanicableComponent) OutputChannels(config any) []core.OutputChannel {
	return s.underlying.OutputChannels(config)
}

/*
 * Panicking methods.
 * These are where the component logic is implemented,
 * so they could panic, and if they do, the system shouldn't crash.
 */
func (s *PanicableComponent) Setup(ctx core.SetupContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Errorf("Component %s panicked in Setup(): %v\nStack: %s",
				s.underlying.Name(), r, debug.Stack())
			err = fmt.Errorf("component %s panicked in Setup(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.Setup(ctx)
}

func (s *PanicableComponent) Execute(ctx core.ExecutionContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Errorf("Component %s panicked in Execute(): %v\nStack: %s",
				s.underlying.Name(), r, debug.Stack())
			err = fmt.Errorf("component %s panicked in Execute(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.Execute(ctx)
}

func (s *PanicableComponent) ProcessQueueItem(ctx core.ProcessQueueContext) (id *uuid.UUID, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("component panicked in ProcessQueueItem(): %v", r)
		}
	}()
	return s.underlying.ProcessQueueItem(ctx)
}

func (s *PanicableComponent) HandleAction(ctx core.ActionContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Errorf("Component %s panicked in HandleAction(%s): %v\nStack: %s",
				s.underlying.Name(), ctx.Name, r, debug.Stack())
			err = fmt.Errorf("component %s panicked in HandleAction(%s): %v",
				s.underlying.Name(), ctx.Name, r)
		}
	}()
	return s.underlying.HandleAction(ctx)
}

func (s *PanicableComponent) HandleWebhook(ctx core.WebhookRequestContext) (status int, err error) {
	defer func() {
		if r := recover(); r != nil {
			status = 500
			err = fmt.Errorf("component panicked in HandleWebhook(): %v", r)
		}
	}()
	return s.underlying.HandleWebhook(ctx)
}

func (s *PanicableComponent) Cancel(ctx core.ExecutionContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Errorf("Component %s panicked in Cancel(): %v\nStack: %s",
				s.underlying.Name(), r, debug.Stack())
			err = fmt.Errorf("component %s panicked in Cancel(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.Cancel(ctx)
}
