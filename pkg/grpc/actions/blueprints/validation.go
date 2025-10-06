package blueprints

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/primitives"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ValidateNodes(nodes []models.Node, registry *registry.Registry) error {
	for _, node := range nodes {
		if node.Ref.Primitive == nil {
			return fmt.Errorf("node %s: primitive is required", node.ID)
		}

		primitive, err := registry.GetPrimitive(node.Ref.Primitive.Name)
		if err != nil {
			return fmt.Errorf("node %s: unknown primitive %s", node.ID, node.Ref.Primitive.Name)
		}

		// Validate configuration
		if err := validateConfiguration(node.ID, node.Configuration, primitive); err != nil {
			return err
		}
	}

	return nil
}

func validateConfiguration(nodeID string, config any, primitive primitives.Primitive) error {
	configFields := primitive.Configuration()

	// Convert config to map for easier validation
	var configMap map[string]any
	if err := mapstructure.Decode(config, &configMap); err != nil {
		return fmt.Errorf("node %s: invalid configuration format", nodeID)
	}

	// Check required fields
	for _, field := range configFields {
		if field.Required {
			value, exists := configMap[field.Name]
			if !exists {
				return fmt.Errorf("node %s: required configuration field '%s' is missing", nodeID, field.Name)
			}

			// Check if the value is empty
			if value == nil || value == "" {
				return fmt.Errorf("node %s: required configuration field '%s' cannot be empty", nodeID, field.Name)
			}
		}
	}

	return nil
}
