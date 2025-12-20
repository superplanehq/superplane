package blueprints

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ValidateNodeConfigurations(nodes []models.Node, registry *registry.Registry) error {
	for _, node := range nodes {
		if err := ValidateNodeConfiguration(node, registry); err != nil {
			return err
		}
	}

	return nil
}

func validateAndMarkNodeErrors(nodes []models.Node, registry *registry.Registry) []models.Node {
	result := make([]models.Node, len(nodes))

	for i, node := range nodes {
		result[i] = node

		if node.ErrorMessage != nil && *node.ErrorMessage != "" {
			continue
		}

		if err := ValidateNodeConfiguration(node, registry); err != nil {
			errorMsg := err.Error()
			result[i].ErrorMessage = &errorMsg
		} else {
			result[i].ErrorMessage = nil
		}
	}

	return result
}

func ValidateNodeConfiguration(node models.Node, registry *registry.Registry) error {
	switch node.Type {
	case models.NodeTypeComponent:
		if node.Ref.Component == nil {
			return fmt.Errorf("node %s: component is required", node.ID)
		}

		component, err := registry.GetComponent(node.Ref.Component.Name)
		if err != nil {
			return fmt.Errorf("node %s: unknown component %s", node.ID, node.Ref.Component.Name)
		}

		return validateConfiguration(node.ID, node.Configuration, component)

	default:
		return fmt.Errorf("node %s: unknown node type %s", node.ID, node.Type)
	}

}

func validateConfiguration(nodeID string, config any, component core.Component) error {
	configFields := component.Configuration()

	// Convert config to map for easier validation
	var configMap map[string]any
	if err := mapstructure.Decode(config, &configMap); err != nil {
		return fmt.Errorf("node %s: invalid configuration format", nodeID)
	}

	// Validate configuration using the components validation function
	if err := configuration.ValidateConfiguration(configFields, configMap); err != nil {
		return fmt.Errorf("node %s: %w", nodeID, err)
	}

	return nil
}
