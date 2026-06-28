package s3

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func ListBuckets(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := ctx.Parameters["region"]
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, credentials, region)
	buckets, err := client.ListBuckets()
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 buckets: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(buckets))
	for _, bucket := range buckets {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: bucket.Name,
			ID:   bucket.Name,
		})
	}

	return resources, nil
}
