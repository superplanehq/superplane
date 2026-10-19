package encryption

import (
	"fmt"
	"slices"

	"github.com/superplanehq/superplane/pkg/registry"
)

func sensitiveConfigurationFields(registry *registry.Registry, appName string) ([]string, error) {
	integration, err := registry.GetIntegration(appName)
	if err != nil {
		return nil, fmt.Errorf("load integration %s: %w", appName, err)
	}
	fields := make([]string, 0)
	for _, field := range integration.Configuration() {
		if field.Sensitive {
			fields = append(fields, field.Name)
		}
	}
	// The factory BYOK path stores this field for OpenRouter even though its
	// regular integration configuration does not declare it.
	if appName == "openrouter" && !slices.Contains(fields, "apiKey") {
		fields = append(fields, "apiKey")
	}
	return fields, nil
}
