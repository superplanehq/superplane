package support

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//
// Dummy application implementation for testing
//

type DummyApplication struct {
	onSync func(ctx core.SyncContext) error
}

func NewDummyApplication(onSync func(ctx core.SyncContext) error) *DummyApplication {
	return &DummyApplication{
		onSync: onSync,
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
	return t.onSync(ctx)
}

func (t *DummyApplication) HandleRequest(ctx core.HTTPRequestContext) {
}

func (t *DummyApplication) RequestWebhook(ctx core.AppInstallationContext, configuration any) error {
	return nil
}

func (t *DummyApplication) SetupWebhook(ctx core.AppInstallationContext, options core.WebhookOptions) (any, error) {
	return nil, nil
}

func (t *DummyApplication) CleanupWebhook(ctx core.AppInstallationContext, options core.WebhookOptions) error {
	return nil
}
