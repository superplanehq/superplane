package registry

import (
	"fmt"
	"runtime/debug"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

/*
 * PanicableIntegration wraps an Integration implementation
 * around logic for handling panics.
 */
type PanicableIntegration struct {
	underlying core.Integration
}

func NewPanicableIntegration(i core.Integration) core.Integration {
	return &PanicableIntegration{underlying: i}
}

/*
 * Non-panicking methods.
 * These are mostly definition methods, so they won't panic.
 */
func (s *PanicableIntegration) Name() string {
	return s.underlying.Name()
}

func (s *PanicableIntegration) Label() string {
	return s.underlying.Label()
}

func (s *PanicableIntegration) Icon() string {
	return s.underlying.Icon()
}

func (s *PanicableIntegration) Description() string {
	return s.underlying.Description()
}

func (s *PanicableIntegration) Instructions() string {
	return s.underlying.Instructions()
}

func (s *PanicableIntegration) Configuration() []configuration.Field {
	return s.underlying.Configuration()
}

func (s *PanicableIntegration) Components() []core.Component {
	components := s.underlying.Components()
	safe := make([]core.Component, len(components))
	for i, c := range components {
		safe[i] = NewPanicableComponent(c)
	}
	return safe
}

func (s *PanicableIntegration) Triggers() []core.Trigger {
	triggers := s.underlying.Triggers()
	safe := make([]core.Trigger, len(triggers))
	for i, t := range triggers {
		safe[i] = NewPanicableTrigger(t)
	}
	return safe
}

/*
 * Panicking methods.
 * These are where the integration logic is implemented,
 * so they could panic, and if they do, the system shouldn't crash.
 */
func (s *PanicableIntegration) Sync(ctx core.SyncContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("integration %s panicked in Sync(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.Sync(ctx)
}

func (s *PanicableIntegration) ListResources(resourceType string, ctx core.ListResourcesContext) (resources []core.IntegrationResource, err error) {
	defer func() {
		if r := recover(); r != nil {
			resources = nil
			err = fmt.Errorf("integration %s panicked in ListResources(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.ListResources(resourceType, ctx)
}

func (s *PanicableIntegration) HandleRequest(ctx core.HTTPRequestContext) {
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.Errorf("Integration %s panicked in HandleRequest(): %v\nStack: %s",
				s.underlying.Name(), r, debug.Stack())
			ctx.Response.WriteHeader(500)
		}
	}()
	s.underlying.HandleRequest(ctx)
}

func (s *PanicableIntegration) CompareWebhookConfig(a, b any) (result bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = false
			err = fmt.Errorf("integration %s panicked in CompareWebhookConfig(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.CompareWebhookConfig(a, b)
}

func (s *PanicableIntegration) SetupWebhook(ctx core.SetupWebhookContext) (metadata any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("integration %s panicked in SetupWebhook(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.SetupWebhook(ctx)
}

func (s *PanicableIntegration) CleanupWebhook(ctx core.CleanupWebhookContext) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("integration %s panicked in CleanupWebhook(): %v",
				s.underlying.Name(), r)
		}
	}()
	return s.underlying.CleanupWebhook(ctx)
}
