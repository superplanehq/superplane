package bitbucket

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
)

func (b *Bitbucket) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case ResourceTypeRepository:
		return b.listRepositories(resourceType, ctx)
	case ResourceTypeMember:
		return b.listMembers(resourceType, ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func (b *Bitbucket) listRepositories(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, metadata, err := integrationClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
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

// listMembers backs the reviewer pickers. The resource ID is the account UUID, which
// is what the Bitbucket API expects when setting reviewers on a pull request.
func (b *Bitbucket) listMembers(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, metadata, err := integrationClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	members, err := client.ListWorkspaceMembers(metadata.Workspace.Slug)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace members: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(members))
	for _, member := range members {
		if member.UUID == "" {
			continue
		}

		name := member.DisplayName
		if name == "" {
			name = member.Nickname
		}

		if name == "" {
			name = member.UUID
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: name,
			ID:   member.UUID,
		})
	}

	return resources, nil
}
