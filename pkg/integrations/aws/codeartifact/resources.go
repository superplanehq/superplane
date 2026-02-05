package codeartifact

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func ListRepositories(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := ctx.Parameters["region"]
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	domain := ctx.Parameters["domain"]
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	repositories, err := client.ListRepositories(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to list codeartifact repositories: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(repositories))
	for _, repository := range repositories {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: repository.Name,
			ID:   repository.Arn,
		})
	}

	return resources, nil
}

func ListDomains(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := ctx.Parameters["region"]
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	domains, err := client.ListDomains()
	if err != nil {
		return nil, fmt.Errorf("failed to list codeartifact domains: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(domains))
	for _, domain := range domains {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: domain.Name,
			ID:   domain.Arn,
		})
	}

	return resources, nil
}
