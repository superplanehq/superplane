package registry

import (
	"fmt"
	"runtime/debug"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

/*
 * PanicableTrigger wraps a Trigger implementation
 * around logic for handling panics.
 */
type PanicableTrigger struct {
	underlying core.Trigger
}

func NewPanicableTrigger(t core.Trigger) core.Trigger {
	return &PanicableTrigger{underlying: t}
}

/*
 * Non-panicking methods.
 * These are mostly definition methods, so they won't panic.
 */
func (s *PanicableTrigger) Name() string {
	return s.underlying.Name()
}

func (s *PanicableTrigger) Label() string {
	return s.underlying.Label()
}

func (s *PanicableTrigger) Description() string {
	return s.underlying.Description()
}

func (s *PanicableTrigger) Icon() string {
	return s.underlying.Icon()
}

func (s *PanicableTrigger) Color() string {
	return s.underlying.Color()
}

func (s *PanicableTrigger) Configuration() []configuration.Field {
	return s.underlying.Configuration()
}

func (s *PanicableTrigger) Actions() []core.Action {
	return s.underlying.Actions()
}

/*
 * Panicking methods.
 * These are where the component logic is implemented,
 * so they could panic, and if they do, the system shouldn't crash.
 */
func (s *PanicableTrigger) Setup(ctx core.TriggerContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Errorf("Trigger %s panicked in Setup(): %v\nStack: %s",
				s.underlying.Name(), r, debug.Stack())
			err = fmt.Errorf("trigger %s panicked in Setup(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.Setup(ctx)
}

func (s *PanicableTrigger) HandleWebhook(ctx core.WebhookRequestContext) (status int, err error) {
	defer func() {
		if r := recover(); r != nil {
			status = 500
			err = fmt.Errorf("trigger %s panicked in HandleWebhook(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.HandleWebhook(ctx)
}

func (s *PanicableTrigger) HandleAction(ctx core.TriggerActionContext) (result map[string]any, err error) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Errorf("Trigger %s panicked in HandleAction(%s): %v\nStack: %s",
				s.underlying.Name(), ctx.Name, r, debug.Stack())
			result = nil
			err = fmt.Errorf("trigger %s panicked in HandleAction(%s): %v",
				s.underlying.Name(), ctx.Name, r)
		}
	}()
	return s.underlying.HandleAction(ctx)
}

func (s *PanicableTrigger) OnAppMessage(ctx core.AppMessageContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Errorf("Trigger %s panicked in OnAppMessage(): %v\nStack: %s",
				s.underlying.Name(), r, debug.Stack())
			err = fmt.Errorf("trigger %s panicked in OnAppMessage(): %v",
				s.underlying.Name(), r)
		}
	}()

	appTrigger, ok := s.underlying.(core.AppTrigger)
	if !ok {
		return fmt.Errorf("trigger %s is not an app trigger", s.underlying.Name())
	}

	return appTrigger.OnAppMessage(ctx)
}
