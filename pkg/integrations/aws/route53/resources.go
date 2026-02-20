package route53

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func ListHostedZones(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	client := NewClient(ctx.HTTP, credentials)
	zones, err := client.ListHostedZones()
	if err != nil {
		return nil, fmt.Errorf("failed to list hosted zones: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(zones))
	for _, zone := range zones {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: zone.Name,
			ID:   zone.ID,
		})
	}

	return resources, nil
}
