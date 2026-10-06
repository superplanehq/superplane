package bitbucket

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	resourceTypeRepository    = "repository"
	resourceTypeDefaultBranch = "default_branch"
)

func (b *Bitbucket) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != resourceTypeRepository && resourceType != resourceTypeDefaultBranch {
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

	if resourceType == resourceTypeDefaultBranch {
		return listDefaultBranch(ctx.Integration, client, ctx.Parameters["repository"])
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

func listDefaultBranch(integration core.IntegrationContext, client *Client, repository string) ([]core.IntegrationResource, error) {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return []core.IntegrationResource{}, nil
	}
	if err := requireRepositoryInWorkspace(integration, repository); err != nil {
		return nil, err
	}

	branch, err := client.GetMainBranch(repository)
	if err != nil {
		return nil, fmt.Errorf("failed to read the main branch: %w", err)
	}
	if branch == "" {
		return []core.IntegrationResource{}, nil
	}

	return []core.IntegrationResource{{Type: resourceTypeDefaultBranch, Name: branch, ID: branch}}, nil
}
