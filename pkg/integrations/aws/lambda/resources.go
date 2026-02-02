package lambda

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func ListFunctions(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := common.RegionFromInstallation(ctx.Integration)
	if strings.TrimSpace(region) == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	functions, err := client.ListFunctions()
	if err != nil {
		return nil, fmt.Errorf("failed to list lambda functions: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(functions))
	for _, function := range functions {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: function.FunctionName,
			ID:   function.FunctionArn,
		})
	}

	return resources, nil
}
