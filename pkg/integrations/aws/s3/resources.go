package s3

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func ListBuckets(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("list S3 buckets: region is required")
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("list S3 buckets: failed to load AWS credentials from integration: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	buckets, err := client.ListBuckets()
	if err != nil {
		return nil, fmt.Errorf("list S3 buckets: failed to list buckets: %w", err)
	}

	var resources []core.IntegrationResource
	for _, bucket := range buckets {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: bucket.Name,
			ID:   bucket.Name,
		})
	}

	return resources, nil
}
