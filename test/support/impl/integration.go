package impl

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DummyIntegration struct {
	actions       []core.Action
	triggers      []core.Trigger
	hooks         []core.Hook
	handleHook    func(ctx core.IntegrationHookContext) error
	onSync        func(ctx core.SyncContext) error
	onCleanup     func(ctx core.IntegrationCleanupContext) error
	listResources func(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error)
}

type DummyIntegrationOptions struct {
	Actions       []core.Action
	Triggers      []core.Trigger
	Hooks         []core.Hook
	HandleHook    func(ctx core.IntegrationHookContext) error
	OnSync        func(ctx core.SyncContext) error
	OnCleanup     func(ctx core.IntegrationCleanupContext) error
	ListResources func(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error)
}

func NewDummyIntegration(options DummyIntegrationOptions) *DummyIntegration {
	return &DummyIntegration{
		actions:       options.Actions,
		triggers:      options.Triggers,
		hooks:         options.Hooks,
		handleHook:    options.HandleHook,
		onSync:        options.OnSync,
		onCleanup:     options.OnCleanup,
		listResources: options.ListResources,
	}
}

func (t *DummyIntegration) Name() string {
	return "dummy"
}

func (t *DummyIntegration) Label() string {
	return "Just a dummy application used in unit tests"
}

func (t *DummyIntegration) Icon() string {
	return "test"
}

func (t *DummyIntegration) Instructions() string {
	return "Just a dummy application used in unit tests"
}

func (t *DummyIntegration) Description() string {
	return "Just a dummy application used in unit tests"
}

func (t *DummyIntegration) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *DummyIntegration) Actions() []core.Action {
	return t.actions
}

func (t *DummyIntegration) Triggers() []core.Trigger {
	return t.triggers
}

func (t *DummyIntegration) Hooks() []core.Hook {
	return t.hooks
}

func (t *DummyIntegration) HandleHook(ctx core.IntegrationHookContext) error {
	if t.handleHook == nil {
		return nil
	}
	return t.handleHook(ctx)
}

func (t *DummyIntegration) Sync(ctx core.SyncContext) error {
	if t.onSync == nil {
		return nil
	}
	return t.onSync(ctx)
}

func (t *DummyIntegration) Cleanup(ctx core.IntegrationCleanupContext) error {
	if t.onCleanup == nil {
		return nil
	}
	return t.onCleanup(ctx)
}

func (t *DummyIntegration) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if t.listResources != nil {
		return t.listResources(resourceType, ctx)
	}
	return []core.IntegrationResource{}, nil
}

func (t *DummyIntegration) HandleRequest(ctx core.HTTPRequestContext) {
}

type DummyIntegrationTriggerOptions struct {
	Name                 string
	OnIntegrationMessage func(ctx core.IntegrationMessageContext) error
}

type DummyIntegrationTrigger struct {
	name                 string
	onIntegrationMessage func(ctx core.IntegrationMessageContext) error
}

func NewDummyIntegrationTrigger(options DummyIntegrationTriggerOptions) *DummyIntegrationTrigger {
	return &DummyIntegrationTrigger{
		name:                 options.Name,
		onIntegrationMessage: options.OnIntegrationMessage,
	}
}

func (t *DummyIntegrationTrigger) Name() string {
	return t.name
}

func (t *DummyIntegrationTrigger) Label() string {
	return t.name
}

func (t *DummyIntegrationTrigger) Description() string {
	return t.name
}

func (t *DummyIntegrationTrigger) Documentation() string {
	return t.name
}

func (t *DummyIntegrationTrigger) Icon() string {
	return "dummy"
}

func (t *DummyIntegrationTrigger) Color() string {
	return "dummy"
}

func (t *DummyIntegrationTrigger) ExampleData() map[string]any {
	return map[string]any{}
}

func (t *DummyIntegrationTrigger) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *DummyIntegrationTrigger) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (t *DummyIntegrationTrigger) Setup(ctx core.TriggerContext) error {
	return nil
}

func (t *DummyIntegrationTrigger) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *DummyIntegrationTrigger) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return map[string]any{}, nil
}

func (t *DummyIntegrationTrigger) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (t *DummyIntegrationTrigger) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	if t.onIntegrationMessage == nil {
		return nil
	}

	return t.onIntegrationMessage(ctx)
}

type DummyWebhookHandlerOptions struct {
	SetupFunc         func(ctx core.WebhookHandlerContext) (any, error)
	CleanupFunc       func(ctx core.WebhookHandlerContext) error
	CompareConfigFunc func(a, b any) (bool, error)
	MergeFunc         func(current, requested any) (any, bool, error)
}

type DummyWebhookHandler struct {
	setupFunc         func(ctx core.WebhookHandlerContext) (any, error)
	cleanupFunc       func(ctx core.WebhookHandlerContext) error
	compareConfigFunc func(a, b any) (bool, error)
	mergeFunc         func(current, requested any) (any, bool, error)
}

func NewDummyWebhookHandler(options DummyWebhookHandlerOptions) *DummyWebhookHandler {
	return &DummyWebhookHandler{
		setupFunc:         options.SetupFunc,
		cleanupFunc:       options.CleanupFunc,
		compareConfigFunc: options.CompareConfigFunc,
		mergeFunc:         options.MergeFunc,
	}
}

func (t *DummyWebhookHandler) CompareConfig(a, b any) (bool, error) {
	if t.compareConfigFunc == nil {
		return false, nil
	}
	return t.compareConfigFunc(a, b)
}

func (t *DummyWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	if t.setupFunc == nil {
		return map[string]any{}, nil
	}
	return t.setupFunc(ctx)
}

func (t *DummyWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	if t.cleanupFunc == nil {
		return nil
	}
	return t.cleanupFunc(ctx)
}

func (t *DummyWebhookHandler) Merge(current, requested any) (any, bool, error) {
	if t.mergeFunc == nil {
		return current, false, nil
	}
	return t.mergeFunc(current, requested)
}
