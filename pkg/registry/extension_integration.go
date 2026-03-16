package registry

import (
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/extensions"
)

type ExtensionIntegration struct {
	manifest extensions.IntegrationManifest
}

func NewExtensionIntegration(manifest extensions.IntegrationManifest) ExtensionIntegration {
	return ExtensionIntegration{manifest: manifest}
}

func (s *ExtensionIntegration) Name() string {
	return s.manifest.Name
}

func (s *ExtensionIntegration) Label() string {
	return s.manifest.Label
}

func (s *ExtensionIntegration) Icon() string {
	return s.manifest.Icon
}

func (s *ExtensionIntegration) Description() string {
	return s.manifest.Description
}

func (s *ExtensionIntegration) Instructions() string {
	return s.manifest.Instructions
}

func (s *ExtensionIntegration) Configuration() []configuration.Field {
	// return s.manifest.Configuration
	return nil
}

func (s *ExtensionIntegration) Actions() []core.Action {
	// return s.manifest.Actions
	return nil
}

func (s *ExtensionIntegration) Components() []core.Component {
	return nil
}

func (s *ExtensionIntegration) Triggers() []core.Trigger {
	return nil
}

func (s *ExtensionIntegration) Sync(ctx core.SyncContext) (err error) {
	return nil
}

func (s *ExtensionIntegration) Cleanup(ctx core.IntegrationCleanupContext) (err error) {
	return nil
}

func (s *ExtensionIntegration) HandleAction(ctx core.IntegrationActionContext) (err error) {
	return nil
}

func (s *ExtensionIntegration) ListResources(resourceType string, ctx core.ListResourcesContext) (resources []core.IntegrationResource, err error) {
	return nil, nil
}

func (s *ExtensionIntegration) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}
