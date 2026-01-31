package support

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//
// Dummy application implementation for testing
//

type DummyIntegration struct {
	onSync                 func(ctx core.SyncContext) error
	onCompareWebhookConfig func(a, b any) (bool, error)
	onSetupWebhook         func(ctx core.SetupWebhookContext) (any, error)
	onCleanup              func(ctx core.IntegrationCleanupContext) error
}

type DummyIntegrationOptions struct {
	Actions                []core.Action
	HandleAction           func(ctx core.IntegrationActionContext) error
	OnSync                 func(ctx core.SyncContext) error
	OnCompareWebhookConfig func(a, b any) (bool, error)
	OnSetupWebhook         func(ctx core.SetupWebhookContext) (any, error)
	OnCleanup              func(ctx core.IntegrationCleanupContext) error
}

func NewDummyIntegration(
	options DummyIntegrationOptions,
) *DummyIntegration {
	return &DummyIntegration{
		onSync:                 options.OnSync,
		onCompareWebhookConfig: options.OnCompareWebhookConfig,
		onSetupWebhook:         options.OnSetupWebhook,
		onCleanup:              options.OnCleanup,
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
	return t.Actions()
}

func (t *DummyIntegration) HandleAction(ctx core.IntegrationActionContext) error {
	if t.HandleAction == nil {
		return nil
	}
	return t.HandleAction(ctx)
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

func (t *DummyIntegration) CompareWebhookConfig(a, b any) (bool, error) {
	if t.onCompareWebhookConfig != nil {
		return t.onCompareWebhookConfig(a, b)
	}
	return true, nil
}

func (t *DummyIntegration) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	if t.onSetupWebhook == nil {
		return nil, nil
	}
	return t.onSetupWebhook(ctx)
}

func (t *DummyIntegration) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}
