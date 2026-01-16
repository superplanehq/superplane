package support

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//
// Dummy application implementation for testing
//

type DummyApplication struct {
	onSync         func(ctx core.SyncContext) error
	onSetupWebhook func(ctx core.SetupWebhookContext) (any, error)
}

func NewDummyApplication(onSync func(ctx core.SyncContext) error) *DummyApplication {
	return NewDummyApplicationWithSetupWebhook(onSync, nil)
}

func NewDummyApplicationWithSetupWebhook(
	onSync func(ctx core.SyncContext) error,
	onSetupWebhook func(ctx core.SetupWebhookContext) (any, error),
) *DummyApplication {
	return &DummyApplication{
		onSync:         onSync,
		onSetupWebhook: onSetupWebhook,
	}
}

func (t *DummyApplication) Name() string {
	return "dummy"
}

func (t *DummyApplication) Label() string {
	return "Just a dummy application used in unit tests"
}

func (t *DummyApplication) Icon() string {
	return "test"
}

func (t *DummyApplication) InstallationInstructions() string {
	return "Just a dummy application used in unit tests"
}

func (t *DummyApplication) Description() string {
	return "Just a dummy application used in unit tests"
}

func (t *DummyApplication) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *DummyApplication) Components() []core.Component {
	return []core.Component{}
}

func (t *DummyApplication) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (t *DummyApplication) Sync(ctx core.SyncContext) error {
	if t.onSync == nil {
		return nil
	}
	return t.onSync(ctx)
}

func (t *DummyApplication) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.ApplicationResource, error) {
	return []core.ApplicationResource{}, nil
}

func (t *DummyApplication) HandleRequest(ctx core.HTTPRequestContext) {
}

func (t *DummyApplication) CompareWebhookConfig(a, b any) (bool, error) {
	return false, nil
}

func (t *DummyApplication) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	if t.onSetupWebhook == nil {
		return nil, nil
	}
	return t.onSetupWebhook(ctx)
}

func (t *DummyApplication) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}
