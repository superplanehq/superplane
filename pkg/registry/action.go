package registry

import (
	"fmt"
	"runtime/debug"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

/*
 * PanicableAction wraps a Action implementation
 * around logic for handling panics.
 */
type PanicableAction struct {
	underlying core.Action
}

func NewPanicableAction(a core.Action) core.Action {
	return &PanicableAction{underlying: a}
}

/*
 * Non-panicking methods.
 * These are mostly definition methods, so they won't panic.
 */
func (s *PanicableAction) Name() string {
	return s.underlying.Name()
}

func (s *PanicableAction) Label() string {
	return s.underlying.Label()
}

func (s *PanicableAction) Description() string {
	return s.underlying.Description()
}

func (s *PanicableAction) Documentation() string {
	return s.underlying.Documentation()
}

func (s *PanicableAction) Icon() string {
	return s.underlying.Icon()
}

func (s *PanicableAction) Color() string {
	return s.underlying.Color()
}

func (s *PanicableAction) ExampleOutput() map[string]any {
	return s.underlying.ExampleOutput()
}

func (s *PanicableAction) Configuration() []configuration.Field {
	return s.underlying.Configuration()
}

func (s *PanicableAction) Hooks() []core.Hook {
	return s.underlying.Hooks()
}

func (s *PanicableAction) OutputChannels(config any) []core.OutputChannel {
	return s.underlying.OutputChannels(config)
}

/*
 * Panicking methods.
 * These are where the action logic is implemented,
 * so they could panic, and if they do, the system shouldn't crash.
 */
func (s *PanicableAction) Setup(ctx core.SetupContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Errorf("Action %s panicked in Setup(): %v\nStack: %s",
				s.underlying.Name(), r, debug.Stack())
			err = fmt.Errorf("action %s panicked in Setup(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.Setup(ctx)
}

func (s *PanicableAction) Execute(ctx core.ExecutionContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Errorf("Action %s panicked in Execute(): %v\nStack: %s",
				s.underlying.Name(), r, debug.Stack())
			err = fmt.Errorf("action %s panicked in Execute(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.Execute(ctx)
}

func (s *PanicableAction) ProcessQueueItem(ctx core.ProcessQueueContext) (id *uuid.UUID, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("action panicked in ProcessQueueItem(): %v", r)
		}
	}()
	return s.underlying.ProcessQueueItem(ctx)
}

func (s *PanicableAction) HandleHook(ctx core.ActionHookContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Errorf("Action %s panicked in HandleHook(%s): %v\nStack: %s",
				s.underlying.Name(), ctx.Name, r, debug.Stack())
			err = fmt.Errorf("action %s panicked in HandleHook(%s): %v",
				s.underlying.Name(), ctx.Name, r)
		}
	}()

	return s.underlying.HandleHook(ctx)
}

func (s *PanicableAction) HandleWebhook(ctx core.WebhookRequestContext) (status int, response *core.WebhookResponseBody, err error) {
	defer func() {
		if r := recover(); r != nil {
			status = 500
			err = fmt.Errorf("action panicked in HandleWebhook(): %v", r)
		}
	}()
	return s.underlying.HandleWebhook(ctx)
}

func (s *PanicableAction) Cancel(ctx core.ExecutionContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Errorf("Action %s panicked in Cancel(): %v\nStack: %s",
				s.underlying.Name(), r, debug.Stack())
			err = fmt.Errorf("action %s panicked in Cancel(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.Cancel(ctx)
}

func (s *PanicableAction) Cleanup(ctx core.SetupContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if ctx.Logger != nil {
				ctx.Logger.Errorf("Action %s panicked in Cleanup(): %v\nStack: %s",
					s.underlying.Name(), r, debug.Stack())
			}
			err = fmt.Errorf("action %s panicked in Cleanup(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.Cleanup(ctx)
}

func (s *PanicableAction) OnIntegrationMessage(ctx core.IntegrationMessageContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Errorf("Action %s panicked in OnIntegrationMessage(): %v\nStack: %s",
				s.underlying.Name(), r, debug.Stack())
			err = fmt.Errorf("action %s panicked in OnIntegrationMessage(): %v",
				s.underlying.Name(), r)
		}
	}()

	integrationAction, ok := s.underlying.(core.IntegrationAction)
	if !ok {
		return fmt.Errorf("action %s is not an integration action", s.underlying.Name())
	}

	return integrationAction.OnIntegrationMessage(ctx)
}
