package registry_test

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
	support "github.com/superplanehq/superplane/test/support"
	supportcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestRegistry_FindComponentHook(t *testing.T) {
	t.Run("finds hook on registered component", func(t *testing.T) {
		called := false
		component := support.NewDummyComponent(support.DummyComponentOptions{
			Name:  "unit_component",
			Hooks: []core.Hook{{Name: "unit-hook", Type: core.HookTypeUser}},
			HandleHookFunc: func(ctx core.ActionHookContext) error {
				called = true
				assert.Equal(t, "unit-hook", ctx.Name)
				return nil
			},
		})

		r := &registry.Registry{
			Components:   map[string]core.Component{"unit_component": registry.NewPanicableComponent(component)},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{},
		}

		gotComponent, hook, err := r.FindComponentHook("unit_component", "unit-hook")
		require.NoError(t, err)
		require.NotNil(t, hook)
		assert.Equal(t, "unit-hook", hook.Name)
		assert.Equal(t, "unit_component", gotComponent.Name())

		err = gotComponent.HandleHook(core.ActionHookContext{
			Name:           "unit-hook",
			Logger:         log.NewEntry(log.StandardLogger()),
			HTTP:           &supportcontexts.HTTPContext{},
			Metadata:       &supportcontexts.MetadataContext{},
			ExecutionState: &supportcontexts.ExecutionStateContext{KVs: map[string]string{}},
			Auth:           &supportcontexts.AuthContext{},
			Requests:       &supportcontexts.RequestContext{},
			Integration:    &supportcontexts.IntegrationContext{},
			Notifications:  &supportcontexts.NotificationContext{},
			Secrets:        &supportcontexts.SecretsContext{Values: map[string][]byte{}},
		})
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("finds hook on integration component", func(t *testing.T) {
		called := false
		component := support.NewDummyComponent(support.DummyComponentOptions{
			Name:  "unitapp.action",
			Hooks: []core.Hook{{Name: "integration-action", Type: core.HookTypeUser}},
			HandleHookFunc: func(ctx core.ActionHookContext) error {
				called = true
				assert.Equal(t, "integration-action", ctx.Name)
				return nil
			},
		})

		integration := support.NewDummyIntegration(support.DummyIntegrationOptions{
			Components: []core.Component{component},
		})

		r := &registry.Registry{
			Components:   map[string]core.Component{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{"unitapp": registry.NewPanicableIntegration(integration)},
		}

		gotComponent, hook, err := r.FindComponentHook("unitapp.action", "integration-action")
		require.NoError(t, err)
		require.NotNil(t, hook)
		assert.Equal(t, "integration-action", hook.Name)
		assert.Equal(t, "unitapp.action", gotComponent.Name())

		err = gotComponent.HandleHook(core.ActionHookContext{
			Name:           "integration-action",
			Logger:         log.NewEntry(log.StandardLogger()),
			HTTP:           &supportcontexts.HTTPContext{},
			Metadata:       &supportcontexts.MetadataContext{},
			ExecutionState: &supportcontexts.ExecutionStateContext{KVs: map[string]string{}},
			Auth:           &supportcontexts.AuthContext{},
			Requests:       &supportcontexts.RequestContext{},
			Integration:    &supportcontexts.IntegrationContext{},
			Notifications:  &supportcontexts.NotificationContext{},
			Secrets:        &supportcontexts.SecretsContext{Values: map[string][]byte{}},
		})
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("returns error when component is missing", func(t *testing.T) {
		r := &registry.Registry{
			Components:   map[string]core.Component{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{},
		}

		_, _, err := r.FindComponentHook("missing_component", "any-hook")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "component missing_component not registered")
	})

	t.Run("returns error when hook is missing", func(t *testing.T) {
		component := support.NewDummyComponent(support.DummyComponentOptions{
			Name:  "unit_component",
			Hooks: []core.Hook{},
		})

		r := &registry.Registry{
			Components:   map[string]core.Component{"unit_component": registry.NewPanicableComponent(component)},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{},
		}

		_, _, err := r.FindComponentHook("unit_component", "missing-hook")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hook missing-hook not found for component unit_component")
	})
}

func TestRegistry_FindTriggerHook(t *testing.T) {
	t.Run("finds hook on registered trigger", func(t *testing.T) {
		called := false
		trigger := support.NewDummyTrigger(support.DummyTriggerOptions{
			Name:  "unit_trigger",
			Hooks: []core.Hook{{Name: "emit", Type: core.HookTypeInternal}},
			HandleHookFunc: func(ctx core.TriggerHookContext) (map[string]any, error) {
				called = true
				assert.Equal(t, "emit", ctx.Name)
				return map[string]any{"ok": true}, nil
			},
		})

		r := &registry.Registry{
			Components:   map[string]core.Component{},
			Triggers:     map[string]core.Trigger{"unit_trigger": registry.NewPanicableTrigger(trigger)},
			Integrations: map[string]core.Integration{},
		}

		gotTrigger, hook, err := r.FindTriggerHook("unit_trigger", "emit")
		require.NoError(t, err)
		require.NotNil(t, hook)
		assert.Equal(t, "emit", hook.Name)
		assert.Equal(t, "unit_trigger", gotTrigger.Name())

		_, err = gotTrigger.HandleHook(core.TriggerHookContext{
			Name:        "emit",
			Logger:      log.NewEntry(log.StandardLogger()),
			HTTP:        &supportcontexts.HTTPContext{},
			Metadata:    &supportcontexts.MetadataContext{},
			Requests:    &supportcontexts.RequestContext{},
			Events:      &supportcontexts.EventContext{},
			Webhook:     &supportcontexts.NodeWebhookContext{},
			Integration: &supportcontexts.IntegrationContext{},
		})
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("returns error when trigger is missing", func(t *testing.T) {
		r := &registry.Registry{
			Components:   map[string]core.Component{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{},
		}

		_, _, err := r.FindTriggerHook("missing_trigger", "emit")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "trigger missing_trigger not registered")
	})

	t.Run("returns error when hook is missing", func(t *testing.T) {
		trigger := support.NewDummyTrigger(support.DummyTriggerOptions{
			Name:  "unit_trigger",
			Hooks: []core.Hook{},
		})

		r := &registry.Registry{
			Components:   map[string]core.Component{},
			Triggers:     map[string]core.Trigger{"unit_trigger": registry.NewPanicableTrigger(trigger)},
			Integrations: map[string]core.Integration{},
		}

		_, _, err := r.FindTriggerHook("unit_trigger", "missing-hook")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hook missing-hook not found for trigger unit_trigger")
	})
}

func TestRegistry_FindIntegrationHook(t *testing.T) {
	t.Run("finds hook on registered integration", func(t *testing.T) {
		called := false
		integration := support.NewDummyIntegration(support.DummyIntegrationOptions{
			Hooks: []core.Hook{{Name: "provision", Type: core.HookTypeUser}},
			HandleHook: func(ctx core.IntegrationHookContext) error {
				called = true
				assert.Equal(t, "provision", ctx.Name)
				return nil
			},
		})

		r := &registry.Registry{
			Components:   map[string]core.Component{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{"unit_integration": registry.NewPanicableIntegration(integration)},
		}

		gotIntegration, hook, err := r.FindIntegrationHook("unit_integration", "provision")
		require.NoError(t, err)
		require.NotNil(t, hook)
		assert.Equal(t, "provision", hook.Name)
		assert.Equal(t, "dummy", gotIntegration.Name())

		err = gotIntegration.HandleHook(core.IntegrationHookContext{
			Name:        "provision",
			Parameters:  map[string]any{},
			Logger:      log.NewEntry(log.StandardLogger()),
			Requests:    &supportcontexts.RequestContext{},
			Integration: &supportcontexts.IntegrationContext{},
			HTTP:        &supportcontexts.HTTPContext{},
		})
		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("returns error when integration is missing", func(t *testing.T) {
		r := &registry.Registry{
			Components:   map[string]core.Component{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{},
		}

		_, _, err := r.FindIntegrationHook("missing_integration", "provision")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "integration missing_integration not registered")
	})

	t.Run("returns error when hook is missing", func(t *testing.T) {
		integration := support.NewDummyIntegration(support.DummyIntegrationOptions{
			Hooks: []core.Hook{},
		})

		r := &registry.Registry{
			Components:   map[string]core.Component{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{"unit_integration": registry.NewPanicableIntegration(integration)},
		}

		_, _, err := r.FindIntegrationHook("unit_integration", "missing-hook")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hook missing-hook not found for integration unit_integration")
	})
}
