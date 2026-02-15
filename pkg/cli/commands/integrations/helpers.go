package integrations

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

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
