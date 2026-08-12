package cloudwatch

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func ListLogGroups(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := ctx.Parameters["region"]
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, credentials, region)
	logGroups, err := client.DescribeLogGroups("")
	if err != nil {
		return nil, fmt.Errorf("failed to list CloudWatch log groups: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(logGroups))
	for _, logGroup := range logGroups {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: logGroup.Name,
			ID:   logGroup.Name,
		})
	}

	return resources, nil
}
