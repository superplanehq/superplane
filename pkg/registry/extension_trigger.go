package registry

import (
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/extensions"
)

type ExtensionTrigger struct {
	manifest extensions.TriggerManifest
}

func NewExtensionTrigger(manifest extensions.TriggerManifest) ExtensionTrigger {
	return ExtensionTrigger{manifest: manifest}
}

func (s *ExtensionTrigger) Name() string {
	return s.manifest.Name
}

func (s *ExtensionTrigger) Label() string {
	return s.manifest.Label
}

func (s *ExtensionTrigger) Description() string {
	return s.manifest.Description
}

func (s *ExtensionTrigger) Documentation() string {
	return ""
}

func (s *ExtensionTrigger) Icon() string {
	return s.manifest.Icon
}

func (s *ExtensionTrigger) Color() string {
	return s.manifest.Color
}

func (s *ExtensionTrigger) ExampleData() map[string]any {
	return nil
}

func (s *ExtensionTrigger) Configuration() []configuration.Field {
	// return s.manifest.Configuration
	return nil
}

func (s *ExtensionTrigger) Actions() []core.Action {
	// return s.manifest.Actions
	return nil
}

func (s *ExtensionTrigger) Setup(ctx core.TriggerContext) (err error) {
	return nil
}

func (s *ExtensionTrigger) HandleWebhook(ctx core.WebhookRequestContext) (status int, response *core.WebhookResponseBody, err error) {
	return http.StatusOK, nil, nil
}

func (s *ExtensionTrigger) HandleAction(ctx core.TriggerActionContext) (result map[string]any, err error) {
	return nil, nil
}

func (s *ExtensionTrigger) Cleanup(ctx core.TriggerContext) (err error) {
	return nil
}

func (s *ExtensionTrigger) OnIntegrationMessage(ctx core.IntegrationMessageContext) (err error) {
	return nil
}
