package codepipeline

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func ListPipelines(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	region := ctx.Parameters["region"]
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, region)

	pipelines, err := client.ListPipelines()
	if err != nil {
		return nil, fmt.Errorf("failed to list CodePipeline pipelines: %w", err)
	}

	// Pipeline name is used as ID because AWS ListPipelines does not return ARN.
	resources := make([]core.IntegrationResource, 0, len(pipelines))
	for _, pipeline := range pipelines {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: pipeline.Name,
			ID:   pipeline.Name,
		})
	}

	return resources, nil
}
