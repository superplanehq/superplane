package bitbucket

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func (b *Bitbucket) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "repository" {
		return []core.IntegrationResource{}, nil
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(metadata.AuthType, ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	repositories, err := client.ListRepositories(metadata.Workspace.Slug)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(repositories))
	for _, repo := range repositories {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: repo.FullName,
			ID:   repo.UUID,
		})
	}

	return resources, nil
}
