package support

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//
// Dummy application implementation for testing
//

type DummyIntegration struct {
	actions      []core.Action
	handleAction func(ctx core.IntegrationActionContext) error
	onSync       func(ctx core.SyncContext) error
	onCleanup    func(ctx core.IntegrationCleanupContext) error
}

type DummyIntegrationOptions struct {
	Actions      []core.Action
	HandleAction func(ctx core.IntegrationActionContext) error
	OnSync       func(ctx core.SyncContext) error
}

func NewDummyIntegration(
	options DummyIntegrationOptions,
) *DummyIntegration {
	return &DummyIntegration{
		actions:      options.Actions,
		handleAction: options.HandleAction,
		onSync:       options.OnSync,
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

func (t *DummyIntegration) Components() []core.Component {
	return []core.Component{}
}

func (t *DummyIntegration) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (t *DummyIntegration) Actions() []core.Action {
	return t.actions
}

func (t *DummyIntegration) HandleAction(ctx core.IntegrationActionContext) error {
	if t.handleAction == nil {
		return nil
	}
	return t.handleAction(ctx)
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
	return []core.IntegrationResource{}, nil
}

func (t *DummyIntegration) HandleRequest(ctx core.HTTPRequestContext) {
}

type DummyWebhookHandlerOptions struct {
	SetupFunc         func(ctx core.WebhookHandlerContext) (any, error)
	CleanupFunc       func(ctx core.WebhookHandlerContext) error
	CompareConfigFunc func(a, b any) (bool, error)
}

type DummyWebhookHandler struct {
	setupFunc         func(ctx core.WebhookHandlerContext) (any, error)
	cleanupFunc       func(ctx core.WebhookHandlerContext) error
	compareConfigFunc func(a, b any) (bool, error)
}

func NewDummyWebhookHandler(options DummyWebhookHandlerOptions) *DummyWebhookHandler {
	return &DummyWebhookHandler{
		setupFunc:         options.SetupFunc,
		cleanupFunc:       options.CleanupFunc,
		compareConfigFunc: options.CompareConfigFunc,
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
