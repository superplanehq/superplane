package factories

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// Keep stored bindings independent of editable connection names.
func resolveConfigurationBindingName(tx *gorm.DB, organizationID uuid.UUID, requested *factoryConfigurationBinding) (*factoryConfigurationBinding, error) {
	if requested == nil {
		return nil, nil
	}
	binding := *requested
	if binding.IntegrationName == "" {
		return &binding, nil
	}
	if binding.IntegrationID != "" {
		return nil, configurationPreparationError("binding.integration_name", "Use integration_name or integration_id, not both.")
	}
	integrations, err := factoryAutomationIntegrations(tx, organizationID)
	if err != nil {
		return nil, err
	}
	for _, integration := range integrations {
		if integration.InstallationName != binding.IntegrationName {
			continue
		}
		if binding.IntegrationID != "" {
			return nil, configurationPreparationError("binding.integration_name", "The integration name is ambiguous. Use an integration ID.")
		}
		binding.IntegrationID = integration.ID.String()
	}
	if binding.IntegrationID == "" {
		return nil, configurationPreparationError("binding.integration_name", "The integration name must match an existing connection in this organization.")
	}
	binding.IntegrationName = ""
	return &binding, nil
}

func portableConfigurationBinding(integrations []models.Integration, binding *factoryConfigurationBinding) *factoryConfigurationBinding {
	if binding == nil {
		return nil
	}
	exported := *binding
	for _, integration := range integrations {
		if integration.ID.String() != binding.IntegrationID || integration.InstallationName == "" {
			continue
		}
		matches := 0
		for _, candidate := range integrations {
			if candidate.InstallationName == integration.InstallationName {
				matches++
			}
		}
		if matches == 1 {
			exported.IntegrationName = integration.InstallationName
			exported.IntegrationID = ""
		}
		break
	}
	return &exported
}

func configurationBindingIntentEqual(current *factoryConfigurationAutomationTarget, requested *factoryConfigurationBinding) bool {
	intent := func(binding *factoryConfigurationBinding) factoryConfigurationBinding {
		if binding == nil {
			return factoryConfigurationBinding{}
		}
		value := *binding
		if value.IntegrationName != "" {
			value.IntegrationName, value.IntegrationID = "", current.binding.IntegrationID
		}
		return value
	}
	desired := intent(requested)
	// A graph can supply public bindings, such as Linear project filters,
	// without a separate explicit binding in the saved manifest.
	return intent(current.explicitBinding) == desired || intent(current.exportedBinding) == desired
}
