package impl

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DummyTriggerOptions struct {
	Name              string
	Hooks             []core.Hook
	HandleHookFunc    func(ctx core.TriggerHookContext) (map[string]any, error)
	SetupFunc         func(ctx core.TriggerContext) error
	HandleWebhookFunc func(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error)
	CleanupFunc       func(ctx core.SetupContext) error
}

type DummyTrigger struct {
	name              string
	hooks             []core.Hook
	handleHookFunc    func(ctx core.TriggerHookContext) (map[string]any, error)
	setupFunc         func(ctx core.TriggerContext) error
	handleWebhookFunc func(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error)
	cleanupFunc       func(ctx core.SetupContext) error
}

func NewDummyTrigger(options DummyTriggerOptions) *DummyTrigger {
	name := options.Name
	if name == "" {
		name = "dummy"
	}

	return &DummyTrigger{
		name:              name,
		hooks:             options.Hooks,
		handleHookFunc:    options.HandleHookFunc,
		setupFunc:         options.SetupFunc,
		handleWebhookFunc: options.HandleWebhookFunc,
		cleanupFunc:       options.CleanupFunc,
	}
}

func (t *DummyTrigger) Name() string {
	return t.name
}

func (t *DummyTrigger) Label() string {
	return "dummy"
}

func (t *DummyTrigger) Description() string {
	return "Just a dummy trigger used in unit tests"
}

func (t *DummyTrigger) Documentation() string {
	return ""
}

func (t *DummyTrigger) Icon() string {
	return "dummy"
}

func (t *DummyTrigger) Color() string {
	return "dummy"
}

func (t *DummyTrigger) ExampleData() map[string]any {
	return nil
}

func (t *DummyTrigger) Configuration() []configuration.Field {
	return nil
}

func (t *DummyTrigger) Hooks() []core.Hook {
	return t.hooks
}

func (t *DummyTrigger) Setup(ctx core.TriggerContext) error {
	if t.setupFunc == nil {
		return nil
	}
	return t.setupFunc(ctx)
}

func (t *DummyTrigger) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	if t.handleHookFunc != nil {
		return t.handleHookFunc(ctx)
	}
	return nil, nil
}

func (t *DummyTrigger) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (t *DummyTrigger) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	if t.handleWebhookFunc == nil {
		return 200, nil, nil
	}
	return t.handleWebhookFunc(ctx)
}
