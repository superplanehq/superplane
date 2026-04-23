package contexts

import (
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

type IntegrationCapabilityRegistry struct {
	integration *models.Integration
}

func NewIntegrationCapabilityRegistry(integration *models.Integration) *IntegrationCapabilityRegistry {
	return &IntegrationCapabilityRegistry{integration: integration}
}

func (r *IntegrationCapabilityRegistry) RegisterComponents(components []core.Component) error {
	for _, component := range components {
		r.integration.Capabilities = append(r.integration.Capabilities, core.CapabilityDefinition{
			Type: core.IntegrationCapabilityTypeComponent,
			Component: &core.ComponentDefinition{
				Name:        component.Name(),
				Label:       component.Label(),
				Description: component.Description(),
			},
		})
	}
	return nil
}

func (r *IntegrationCapabilityRegistry) RegisterTriggers(triggers []core.Trigger) error {
	for _, trigger := range triggers {
		r.integration.Capabilities = append(r.integration.Capabilities, core.CapabilityDefinition{
			Type: core.IntegrationCapabilityTypeTrigger,
			Trigger: &core.TriggerDefinition{
				Name:        trigger.Name(),
				Label:       trigger.Label(),
				Description: trigger.Description(),
			},
		})
	}

	return nil
}
