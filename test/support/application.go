package support

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//
// Dummy application implementation for testing
//

type DummyIntegration struct {
	onSync         func(ctx core.SyncContext) error
	onSetupWebhook func(ctx core.SetupWebhookContext) (any, error)
}

func NewDummyIntegration(onSync func(ctx core.SyncContext) error) *DummyIntegration {
	return NewDummyIntegrationWithSetupWebhook(onSync, nil)
}

func NewDummyIntegrationWithSetupWebhook(
	onSync func(ctx core.SyncContext) error,
	onSetupWebhook func(ctx core.SetupWebhookContext) (any, error),
) *DummyIntegration {
	return &DummyIntegration{
		onSync:         onSync,
		onSetupWebhook: onSetupWebhook,
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

func (t *DummyIntegration) InstallationInstructions() string {
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

func (t *DummyIntegration) Sync(ctx core.SyncContext) error {
	if t.onSync == nil {
		return nil
	}
	return t.onSync(ctx)
}

func (t *DummyIntegration) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.ApplicationResource, error) {
	return []core.ApplicationResource{}, nil
}

func (t *DummyIntegration) HandleRequest(ctx core.HTTPRequestContext) {
}

func (t *DummyIntegration) CompareWebhookConfig(a, b any) (bool, error) {
	return false, nil
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
