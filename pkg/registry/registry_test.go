package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type runTitleTestTrigger struct {
	name            string
	defaultRunTitle string
}

func (t *runTitleTestTrigger) Name() string                         { return t.name }
func (t *runTitleTestTrigger) Label() string                        { return t.name }
func (t *runTitleTestTrigger) Description() string                  { return "" }
func (t *runTitleTestTrigger) Documentation() string                { return "" }
func (t *runTitleTestTrigger) Icon() string                         { return "" }
func (t *runTitleTestTrigger) Color() string                        { return "" }
func (t *runTitleTestTrigger) ExampleData() map[string]any          { return nil }
func (t *runTitleTestTrigger) Configuration() []configuration.Field { return nil }
func (t *runTitleTestTrigger) DefaultRunTitle() string              { return t.defaultRunTitle }
func (t *runTitleTestTrigger) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (t *runTitleTestTrigger) Setup(ctx core.TriggerContext) error { return nil }
func (t *runTitleTestTrigger) Actions() []core.Action              { return nil }
func (t *runTitleTestTrigger) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}
func (t *runTitleTestTrigger) Cleanup(ctx core.TriggerContext) error { return nil }

type runTitleTestIntegration struct {
	name     string
	triggers []core.Trigger
}

func (i *runTitleTestIntegration) Name() string                         { return i.name }
func (i *runTitleTestIntegration) Label() string                        { return i.name }
func (i *runTitleTestIntegration) Icon() string                         { return "" }
func (i *runTitleTestIntegration) Description() string                  { return "" }
func (i *runTitleTestIntegration) Instructions() string                 { return "" }
func (i *runTitleTestIntegration) Configuration() []configuration.Field { return nil }
func (i *runTitleTestIntegration) Components() []core.Component         { return nil }
func (i *runTitleTestIntegration) Triggers() []core.Trigger             { return i.triggers }
func (i *runTitleTestIntegration) Sync(ctx core.SyncContext) error      { return nil }
func (i *runTitleTestIntegration) Actions() []core.Action               { return nil }
func (i *runTitleTestIntegration) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
func (i *runTitleTestIntegration) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}
func (i *runTitleTestIntegration) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return nil, nil
}
func (i *runTitleTestIntegration) HandleRequest(ctx core.HTTPRequestContext) {}

func TestDefaultRunTitleForTrigger(t *testing.T) {
	mu.Lock()
	originalTriggers := registeredTriggers
	originalIntegrations := registeredIntegrations
	registeredTriggers = map[string]core.Trigger{}
	registeredIntegrations = map[string]core.Integration{}
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		registeredTriggers = originalTriggers
		registeredIntegrations = originalIntegrations
		mu.Unlock()
	})

	RegisterTrigger("test.trigger", &runTitleTestTrigger{
		name:            "test.trigger",
		defaultRunTitle: "  direct title  ",
	})

	assert.Equal(t, "direct title", DefaultRunTitleForTrigger("test.trigger"))
	assert.Equal(t, "", DefaultRunTitleForTrigger("missing.trigger"))
}

func TestDefaultRunTitleForTrigger_FromIntegrationTrigger(t *testing.T) {
	mu.Lock()
	originalTriggers := registeredTriggers
	originalIntegrations := registeredIntegrations
	registeredTriggers = map[string]core.Trigger{}
	registeredIntegrations = map[string]core.Integration{}
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		registeredTriggers = originalTriggers
		registeredIntegrations = originalIntegrations
		mu.Unlock()
	})

	RegisterIntegration("test.integration", &runTitleTestIntegration{
		name: "test.integration",
		triggers: []core.Trigger{
			&runTitleTestTrigger{
				name:            "test.integration.trigger",
				defaultRunTitle: "integration title",
			},
		},
	})

	assert.Equal(t, "integration title", DefaultRunTitleForTrigger("test.integration.trigger"))
}

func TestNormalizeRunTitleTemplateForTrigger(t *testing.T) {
	mu.Lock()
	originalTriggers := registeredTriggers
	originalIntegrations := registeredIntegrations
	registeredTriggers = map[string]core.Trigger{}
	registeredIntegrations = map[string]core.Integration{}
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		registeredTriggers = originalTriggers
		registeredIntegrations = originalIntegrations
		mu.Unlock()
	})

	RegisterTrigger("test.trigger", &runTitleTestTrigger{
		name:            "test.trigger",
		defaultRunTitle: "default title",
	})

	assert.Equal(t, "", NormalizeRunTitleTemplateForTrigger("test.trigger", ""))
	assert.Equal(t, "", NormalizeRunTitleTemplateForTrigger("test.trigger", "   "))
	assert.Equal(t, "", NormalizeRunTitleTemplateForTrigger("test.trigger", "  default title  "))
	assert.Equal(t, "custom title", NormalizeRunTitleTemplateForTrigger("test.trigger", "  custom title  "))
	assert.Equal(t, "custom title", NormalizeRunTitleTemplateForTrigger("missing.trigger", "  custom title  "))
}
