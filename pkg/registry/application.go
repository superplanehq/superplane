package registry

import (
	"fmt"
	"runtime/debug"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

/*
 * PanicableApplication wraps an Application implementation
 * around logic for handling panics.
 */
type PanicableApplication struct {
	underlying core.Application
}

func NewPanicableApplication(a core.Application) core.Application {
	return &PanicableApplication{underlying: a}
}

/*
 * Non-panicking methods.
 * These are mostly definition methods, so they won't panic.
 */
func (s *PanicableApplication) Name() string {
	return s.underlying.Name()
}

func (s *PanicableApplication) Label() string {
	return s.underlying.Label()
}

func (s *PanicableApplication) Icon() string {
	return s.underlying.Icon()
}

func (s *PanicableApplication) Description() string {
	return s.underlying.Description()
}

func (s *PanicableApplication) InstallationInstructions() string {
	return s.underlying.InstallationInstructions()
}

func (s *PanicableApplication) Configuration() []configuration.Field {
	return s.underlying.Configuration()
}

func (s *PanicableApplication) Components() []core.Component {
	components := s.underlying.Components()
	safe := make([]core.Component, len(components))
	for i, c := range components {
		safe[i] = NewPanicableComponent(c)
	}
	return safe
}

func (s *PanicableApplication) Triggers() []core.Trigger {
	triggers := s.underlying.Triggers()
	safe := make([]core.Trigger, len(triggers))
	for i, t := range triggers {
		safe[i] = NewPanicableTrigger(t)
	}
	return safe
}

/*
 * Panicking methods.
 * These are where the application logic is implemented,
 * so they could panic, and if they do, the system shouldn't crash.
 */
func (s *PanicableApplication) Sync(ctx core.SyncContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("application %s panicked in Sync(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.Sync(ctx)
}

func (s *PanicableApplication) ListResources(resourceType string, ctx core.ListResourcesContext) (resources []core.ApplicationResource, err error) {
	defer func() {
		if r := recover(); r != nil {
			resources = nil
			err = fmt.Errorf("application %s panicked in ListResources(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.ListResources(resourceType, ctx)
}

func (s *PanicableApplication) HandleRequest(ctx core.HTTPRequestContext) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Errorf("Application %s panicked in HandleRequest(): %v\nStack: %s",
				s.underlying.Name(), r, debug.Stack())
			ctx.Response.WriteHeader(500)
		}
	}()
	s.underlying.HandleRequest(ctx)
}

func (s *PanicableApplication) CompareWebhookConfig(a, b any) (result bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = false
			err = fmt.Errorf("application %s panicked in CompareWebhookConfig(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.CompareWebhookConfig(a, b)
}

func (s *PanicableApplication) SetupWebhook(ctx core.SetupWebhookContext) (metadata any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("application %s panicked in SetupWebhook(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.SetupWebhook(ctx)
}

func (s *PanicableApplication) CleanupWebhook(ctx core.CleanupWebhookContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("application %s panicked in CleanupWebhook(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.CleanupWebhook(ctx)
}
