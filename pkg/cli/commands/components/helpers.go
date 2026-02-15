package components

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func parseIntegrationScopedName(name string) (string, string, bool) {
	integrationName, resourceName, hasDot := strings.Cut(name, ".")
	if !hasDot || integrationName == "" || resourceName == "" {
		return "", "", false
	}

	return integrationName, resourceName, true
}

func findIntegrationDefinitionByName(
	ctx core.CommandContext,
	name string,
) (openapi_client.IntegrationsIntegrationDefinition, error) {
	response, _, err := ctx.API.IntegrationAPI.IntegrationsListIntegrations(ctx.Context).Execute()
	if err != nil {
		return openapi_client.IntegrationsIntegrationDefinition{}, err
	}

	for _, integration := range response.GetIntegrations() {
		if integration.GetName() == name {
			return integration, nil
		}
	}

	return openapi_client.IntegrationsIntegrationDefinition{}, fmt.Errorf("integration %q not found", name)
}

func findIntegrationComponentByName(
	integration openapi_client.IntegrationsIntegrationDefinition,
	name string,
) (openapi_client.ComponentsComponent, error) {
	for _, component := range integration.GetComponents() {
		componentName := component.GetName()
		if componentName == name || componentName == fmt.Sprintf("%s.%s", integration.GetName(), name) {
			return component, nil
		}
	}

	return openapi_client.ComponentsComponent{}, fmt.Errorf("component %q not found in integration %q", name, integration.GetName())
}
