package bitbucket

import (
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type NodeMetadata struct {
	Repository *RepositoryMetadata `json:"repository" mapstructure:"repository"`
}

type RepositoryMetadata struct {
	UUID     string `json:"uuid" mapstructure:"uuid"`
	Name     string `json:"name" mapstructure:"name"`
	FullName string `json:"full_name" mapstructure:"full_name"`
	Slug     string `json:"slug" mapstructure:"slug"`
}

func ensureRepoInMetadata(http core.HTTPContext, ctx core.MetadataContext, integration core.IntegrationContext, repository string) (*RepositoryMetadata, error) {
	if repository == "" {
		return nil, fmt.Errorf("repository is required")
	}

	var nodeMetadata NodeMetadata
	if err := mapstructure.Decode(ctx.Get(), &nodeMetadata); err != nil {
		return nil, fmt.Errorf("failed to decode node metadata: %w", err)
	}

	if nodeMetadata.Repository != nil && repositoryMetadataMatches(*nodeMetadata.Repository, repository) {
		return nodeMetadata.Repository, nil
	}

	var integrationMetadata Metadata
	if err := mapstructure.Decode(integration.GetMetadata(), &integrationMetadata); err != nil {
		return nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(integrationMetadata.AuthType, http, integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	repositories, err := client.ListRepositories(integrationMetadata.Workspace.Slug)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	repoIndex := slices.IndexFunc(repositories, func(r Repository) bool {
		return repositoryMatches(r, repository)
	})

	if repoIndex == -1 {
		return nil, fmt.Errorf("repository %s is not accessible to workspace", repository)
	}

	repoMetadata := &RepositoryMetadata{
		UUID:     repositories[repoIndex].UUID,
		Name:     repositories[repoIndex].Name,
		FullName: repositories[repoIndex].FullName,
		Slug:     repositories[repoIndex].Slug,
	}

	return repoMetadata, ctx.Set(NodeMetadata{Repository: repoMetadata})
}

func repositoryMetadataMatches(repo RepositoryMetadata, repository string) bool {
	return repo.FullName == repository || repo.Name == repository || repo.Slug == repository || repo.UUID == repository
}

func repositoryMatches(repo Repository, repository string) bool {
	return repo.FullName == repository || repo.Name == repository || repo.Slug == repository || repo.UUID == repository
}
