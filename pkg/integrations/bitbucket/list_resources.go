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
	resourceTypeStatusCheck   = "status_check"
)

func (b *Bitbucket) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != resourceTypeRepository && resourceType != resourceTypeDefaultBranch && resourceType != resourceTypeStatusCheck {
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
		repository := strings.TrimSpace(ctx.Parameters["repository"])
		if repository == "" && metadata.AuthType == AuthTypeRepositoryAccessToken && metadata.Repository != nil {
			repository = metadata.Repository.FullName
		}
		return listDefaultBranch(ctx.Integration, client, repository)
	}

	if resourceType == resourceTypeStatusCheck {
		repository := strings.TrimSpace(ctx.Parameters["repository"])
		if repository == "" && metadata.AuthType == AuthTypeRepositoryAccessToken && metadata.Repository != nil {
			repository = metadata.Repository.FullName
		}
		return b.listStatusCheckResources(ctx, client, repository)
	}

	if metadata.AuthType == AuthTypeRepositoryAccessToken {
		if metadata.Repository == nil {
			return []core.IntegrationResource{}, nil
		}
		repo, err := client.GetRepository(metadata.Repository.FullName)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories: %w", err)
		}
		return []core.IntegrationResource{{Type: resourceType, Name: repo.FullName, ID: repo.UUID}}, nil
	}

	if metadata.Workspace == nil {
		return []core.IntegrationResource{}, nil
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
