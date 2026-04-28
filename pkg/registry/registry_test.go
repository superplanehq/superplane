package registry_test

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	supportcontexts "github.com/superplanehq/superplane/test/support/contexts"
	"github.com/superplanehq/superplane/test/support/impl"
)

func TestRegistry_FindActionHook(t *testing.T) {
	t.Run("finds hook on registered action", func(t *testing.T) {
		called := false
		action := impl.NewDummyAction(impl.DummyActionOptions{
			Name:  "unit_action",
			Hooks: []core.Hook{{Name: "unit-hook", Type: core.HookTypeUser}},
			HandleHookFunc: func(ctx core.ActionHookContext) error {
				called = true
				assert.Equal(t, "unit-hook", ctx.Name)
				return nil
			},
		})

		r := &registry.Registry{
			Actions:      map[string]core.Action{"unit_action": registry.NewPanicableAction(action)},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{},
		}

		gotAction, hook, err := r.FindActionHook("unit_action", "unit-hook")
		require.NoError(t, err)
		require.NotNil(t, hook)
		assert.Equal(t, "unit-hook", hook.Name)
		assert.Equal(t, "unit_action", gotAction.Name())

		err = gotAction.HandleHook(core.ActionHookContext{
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
		component := impl.NewDummyAction(impl.DummyActionOptions{
			Name:  "unitapp.action",
			Hooks: []core.Hook{{Name: "integration-action", Type: core.HookTypeUser}},
			HandleHookFunc: func(ctx core.ActionHookContext) error {
				called = true
				assert.Equal(t, "integration-action", ctx.Name)
				return nil
			},
		})

		integration := impl.NewDummyIntegration(impl.DummyIntegrationOptions{
			Actions: []core.Action{component},
		})

		r := &registry.Registry{
			Actions:      map[string]core.Action{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{"unitapp": registry.NewPanicableIntegration(integration)},
		}

		gotAction, hook, err := r.FindActionHook("unitapp.action", "integration-action")
		require.NoError(t, err)
		require.NotNil(t, hook)
		assert.Equal(t, "integration-action", hook.Name)
		assert.Equal(t, "unitapp.action", gotAction.Name())

		err = gotAction.HandleHook(core.ActionHookContext{
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
			Actions:      map[string]core.Action{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{},
		}

		_, _, err := r.FindActionHook("missing_action", "any-hook")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "action missing_action not registered")
	})

	t.Run("returns error when hook is missing", func(t *testing.T) {
		action := impl.NewDummyAction(impl.DummyActionOptions{
			Name:  "unit_action",
			Hooks: []core.Hook{},
		})

		r := &registry.Registry{
			Actions:      map[string]core.Action{"unit_action": registry.NewPanicableAction(action)},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{},
		}

		_, _, err := r.FindActionHook("unit_action", "missing-hook")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hook missing-hook not found for action unit_action")
	})
}

func TestRegistry_FindTriggerHook(t *testing.T) {
	t.Run("finds hook on registered trigger", func(t *testing.T) {
		called := false
		trigger := impl.NewDummyTrigger(impl.DummyTriggerOptions{
			Name:  "unit_trigger",
			Hooks: []core.Hook{{Name: "emit", Type: core.HookTypeInternal}},
			HandleHookFunc: func(ctx core.TriggerHookContext) (map[string]any, error) {
				called = true
				assert.Equal(t, "emit", ctx.Name)
				return map[string]any{"ok": true}, nil
			},
		})

		r := &registry.Registry{
			Actions:      map[string]core.Action{},
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
			Actions:      map[string]core.Action{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{},
		}

		_, _, err := r.FindTriggerHook("missing_trigger", "emit")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "trigger missing_trigger not registered")
	})

	t.Run("returns error when hook is missing", func(t *testing.T) {
		trigger := impl.NewDummyTrigger(impl.DummyTriggerOptions{
			Name:  "unit_trigger",
			Hooks: []core.Hook{},
		})

		r := &registry.Registry{
			Actions:      map[string]core.Action{},
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
		integration := impl.NewDummyIntegration(impl.DummyIntegrationOptions{
			Hooks: []core.Hook{{Name: "provision", Type: core.HookTypeUser}},
			HandleHook: func(ctx core.IntegrationHookContext) error {
				called = true
				assert.Equal(t, "provision", ctx.Name)
				return nil
			},
		})

		r := &registry.Registry{
			Actions:      map[string]core.Action{},
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
			Actions:      map[string]core.Action{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{},
		}

		_, _, err := r.FindIntegrationHook("missing_integration", "provision")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "integration missing_integration not registered")
	})

	t.Run("returns error when hook is missing", func(t *testing.T) {
		integration := impl.NewDummyIntegration(impl.DummyIntegrationOptions{
			Hooks: []core.Hook{},
		})

		r := &registry.Registry{
			Actions:      map[string]core.Action{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{"unit_integration": registry.NewPanicableIntegration(integration)},
		}

		_, _, err := r.FindIntegrationHook("unit_integration", "missing-hook")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hook missing-hook not found for integration unit_integration")
	})
}

func TestRegistry_FindConfigurableComponent(t *testing.T) {
	t.Run("finds action", func(t *testing.T) {
		action := impl.NewDummyAction(impl.DummyActionOptions{Name: "unit_action"})

		r := &registry.Registry{
			Actions:      map[string]core.Action{"unit_action": registry.NewPanicableAction(action)},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{},
			Widgets:      map[string]core.Widget{},
		}

		component, err := r.FindConfigurableComponent("unit_action")
		require.NoError(t, err)

		foundAction, ok := component.(core.Action)
		require.True(t, ok)
		assert.Equal(t, "unit_action", foundAction.Name())
	})

	t.Run("finds integration action", func(t *testing.T) {
		action := impl.NewDummyAction(impl.DummyActionOptions{Name: "unitapp.action"})
		integration := impl.NewDummyIntegration(impl.DummyIntegrationOptions{
			Actions: []core.Action{action},
		})

		r := &registry.Registry{
			Actions:      map[string]core.Action{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{"unitapp": registry.NewPanicableIntegration(integration)},
			Widgets:      map[string]core.Widget{},
		}

		component, err := r.FindConfigurableComponent("unitapp.action")
		require.NoError(t, err)

		foundAction, ok := component.(core.Action)
		require.True(t, ok)
		assert.Equal(t, "unitapp.action", foundAction.Name())
	})

	t.Run("finds trigger", func(t *testing.T) {
		trigger := impl.NewDummyTrigger(impl.DummyTriggerOptions{Name: "unit_trigger"})

		r := &registry.Registry{
			Actions:      map[string]core.Action{},
			Triggers:     map[string]core.Trigger{"unit_trigger": registry.NewPanicableTrigger(trigger)},
			Integrations: map[string]core.Integration{},
			Widgets:      map[string]core.Widget{},
		}

		component, err := r.FindConfigurableComponent("unit_trigger")
		require.NoError(t, err)

		foundTrigger, ok := component.(core.Trigger)
		require.True(t, ok)
		assert.Equal(t, "unit_trigger", foundTrigger.Name())
	})

	t.Run("finds widget", func(t *testing.T) {
		widget := impl.NewDummyWidget(impl.DummyWidgetOptions{Name: "unit_widget"})

		r := &registry.Registry{
			Actions:      map[string]core.Action{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{},
			Widgets:      map[string]core.Widget{"unit_widget": widget},
		}

		component, err := r.FindConfigurableComponent("unit_widget")
		require.NoError(t, err)

		foundWidget, ok := component.(core.Widget)
		require.True(t, ok)
		assert.Equal(t, "unit_widget", foundWidget.Name())
	})

	t.Run("prefers action when action and trigger share the same name", func(t *testing.T) {
		action := impl.NewDummyAction(impl.DummyActionOptions{Name: "shared_name"})
		trigger := impl.NewDummyTrigger(impl.DummyTriggerOptions{Name: "shared_name"})

		r := &registry.Registry{
			Actions:      map[string]core.Action{"shared_name": registry.NewPanicableAction(action)},
			Triggers:     map[string]core.Trigger{"shared_name": registry.NewPanicableTrigger(trigger)},
			Integrations: map[string]core.Integration{},
			Widgets:      map[string]core.Widget{"shared_name": impl.NewDummyWidget(impl.DummyWidgetOptions{Name: "shared_name"})},
		}

		component, err := r.FindConfigurableComponent("shared_name")
		require.NoError(t, err)

		foundAction, ok := component.(core.Action)
		require.True(t, ok)
		assert.Equal(t, "shared_name", foundAction.Name())
	})

	t.Run("returns error when component is missing", func(t *testing.T) {
		r := &registry.Registry{
			Actions:      map[string]core.Action{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{},
			Widgets:      map[string]core.Widget{},
		}

		_, err := r.FindConfigurableComponent("missing_component")
		require.Error(t, err)
		assert.EqualError(t, err, "component missing_component not found")
	})
}

func TestRegistry_ComponentType(t *testing.T) {
	t.Run("returns component for action", func(t *testing.T) {
		action := impl.NewDummyAction(impl.DummyActionOptions{Name: "unit_action"})

		r := &registry.Registry{
			Actions:      map[string]core.Action{"unit_action": registry.NewPanicableAction(action)},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{},
			Widgets:      map[string]core.Widget{},
		}

		componentType, err := r.ComponentType("unit_action")
		require.NoError(t, err)
		assert.Equal(t, models.NodeTypeComponent, componentType)
	})

	t.Run("returns component for integration action", func(t *testing.T) {
		action := impl.NewDummyAction(impl.DummyActionOptions{Name: "unitapp.action"})
		integration := impl.NewDummyIntegration(impl.DummyIntegrationOptions{
			Actions: []core.Action{action},
		})

		r := &registry.Registry{
			Actions:      map[string]core.Action{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{"unitapp": registry.NewPanicableIntegration(integration)},
			Widgets:      map[string]core.Widget{},
		}

		componentType, err := r.ComponentType("unitapp.action")
		require.NoError(t, err)
		assert.Equal(t, models.NodeTypeComponent, componentType)
	})

	t.Run("returns trigger for trigger", func(t *testing.T) {
		trigger := impl.NewDummyTrigger(impl.DummyTriggerOptions{Name: "unit_trigger"})

		r := &registry.Registry{
			Actions:      map[string]core.Action{},
			Triggers:     map[string]core.Trigger{"unit_trigger": registry.NewPanicableTrigger(trigger)},
			Integrations: map[string]core.Integration{},
			Widgets:      map[string]core.Widget{},
		}

		componentType, err := r.ComponentType("unit_trigger")
		require.NoError(t, err)
		assert.Equal(t, models.NodeTypeTrigger, componentType)
	})

	t.Run("returns widget for widget", func(t *testing.T) {
		widget := impl.NewDummyWidget(impl.DummyWidgetOptions{Name: "unit_widget"})

		r := &registry.Registry{
			Actions:      map[string]core.Action{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{},
			Widgets:      map[string]core.Widget{"unit_widget": widget},
		}

		componentType, err := r.ComponentType("unit_widget")
		require.NoError(t, err)
		assert.Equal(t, models.NodeTypeWidget, componentType)
	})

	t.Run("prefers component when action and trigger share the same name", func(t *testing.T) {
		action := impl.NewDummyAction(impl.DummyActionOptions{Name: "shared_name"})
		trigger := impl.NewDummyTrigger(impl.DummyTriggerOptions{Name: "shared_name"})

		r := &registry.Registry{
			Actions:      map[string]core.Action{"shared_name": registry.NewPanicableAction(action)},
			Triggers:     map[string]core.Trigger{"shared_name": registry.NewPanicableTrigger(trigger)},
			Integrations: map[string]core.Integration{},
			Widgets:      map[string]core.Widget{"shared_name": impl.NewDummyWidget(impl.DummyWidgetOptions{Name: "shared_name"})},
		}

		componentType, err := r.ComponentType("shared_name")
		require.NoError(t, err)
		assert.Equal(t, models.NodeTypeComponent, componentType)
	})

	t.Run("returns error when component is missing", func(t *testing.T) {
		r := &registry.Registry{
			Actions:      map[string]core.Action{},
			Triggers:     map[string]core.Trigger{},
			Integrations: map[string]core.Integration{},
			Widgets:      map[string]core.Widget{},
		}

		componentType, err := r.ComponentType("missing_component")
		require.Error(t, err)
		assert.Empty(t, componentType)
		assert.EqualError(t, err, "component missing_component not found")
	})
}
