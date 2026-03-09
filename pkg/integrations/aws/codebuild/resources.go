package codebuild

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func ListProjects(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	region := ctx.Parameters["region"]
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, region)

	projects, err := client.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to list CodeBuild projects: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(projects))
	for _, project := range projects {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: project,
			ID:   project,
		})
	}

	return resources, nil
}
