package blueprints

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ValidateNodes(nodes []models.Node, registry *registry.Registry) error {
	for _, node := range nodes {
		if node.Ref.Component == nil {
			return fmt.Errorf("node %s: component is required", node.ID)
		}

		component, err := registry.GetComponent(node.Ref.Component.Name)
		if err != nil {
			return fmt.Errorf("node %s: unknown component %s", node.ID, node.Ref.Component.Name)
		}

		// Validate configuration
		if err := validateConfiguration(node.ID, node.Configuration, component); err != nil {
			return err
		}
	}

	return nil
}

func validateConfiguration(nodeID string, config any, component components.Component) error {
	configFields := component.Configuration()

	// Convert config to map for easier validation
	var configMap map[string]any
	if err := mapstructure.Decode(config, &configMap); err != nil {
		return fmt.Errorf("node %s: invalid configuration format", nodeID)
	}

	// Validate configuration using the components validation function
	if err := components.ValidateConfiguration(configFields, configMap); err != nil {
		return fmt.Errorf("node %s: %w", nodeID, err)
	}

	return nil
}
