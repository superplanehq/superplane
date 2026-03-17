package registry

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/extensions"
)

type ExtensionIntegration struct {
	manifest   extensions.IntegrationManifest
	components []core.Component
	triggers   []core.Trigger
}

func NewExtensionIntegration(manifest extensions.IntegrationManifest, components []core.Component, triggers []core.Trigger) *ExtensionIntegration {
	return &ExtensionIntegration{manifest: manifest, components: components, triggers: triggers}
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
	return s.manifest.Configuration
}

func (s *ExtensionIntegration) Actions() []core.Action {
	return s.manifest.Actions
}

func (s *ExtensionIntegration) Components() []core.Component {
	return s.components
}

func (s *ExtensionIntegration) Triggers() []core.Trigger {
	return s.triggers
}

func (s *ExtensionIntegration) Sync(ctx core.SyncContext) (err error) {
	return fmt.Errorf("not implemented")
}

func (s *ExtensionIntegration) Cleanup(ctx core.IntegrationCleanupContext) (err error) {
	return fmt.Errorf("not implemented")
}

func (s *ExtensionIntegration) HandleAction(ctx core.IntegrationActionContext) (err error) {
	return fmt.Errorf("not implemented")
}

func (s *ExtensionIntegration) ListResources(resourceType string, ctx core.ListResourcesContext) (resources []core.IntegrationResource, err error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *ExtensionIntegration) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}
