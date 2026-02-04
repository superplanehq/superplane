package ecr

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func ListRepositories(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := common.RegionFromInstallation(ctx.Integration)
	if strings.TrimSpace(region) == "" {
		return nil, fmt.Errorf("region is required")
	}

	ctx.Logger.Infof("listing ECR repositories in region %s", region)

	client := NewClient(ctx.HTTP, creds, region)
	repositories, err := client.ListRepositories()
	if err != nil {
		return nil, fmt.Errorf("failed to list ECR repositories: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(repositories))
	for _, repository := range repositories {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: repository.RepositoryName,
			ID:   repository.RepositoryArn,
		})
	}

	return resources, nil
}
