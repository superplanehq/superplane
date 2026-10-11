package bitbucket

import (
	"fmt"
	"slices"
	"strings"

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

func ensureRepoInMetadata(http core.HTTPContext, ctx core.MetadataWriter, integration core.IntegrationContext, repository string) (*RepositoryMetadata, error) {
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

	if integrationMetadata.AuthType == AuthTypeRepositoryAccessToken {
		if integrationMetadata.Repository != nil && !repositoryMetadataMatches(*integrationMetadata.Repository, repository) {
			return nil, fmt.Errorf("repository %s is not accessible to workspace", repository)
		}
		repo, err := client.GetRepository(repository)
		if err != nil {
			return nil, fmt.Errorf("repository %s is not accessible to workspace", repository)
		}
		// ponytail: exact-match confinement; per-repo permission checks if Bitbucket adds scoped listing
		if integrationMetadata.Repository != nil && !repositoryMatches(*repo, integrationMetadata.Repository.FullName) {
			return nil, fmt.Errorf("repository %s is not accessible to workspace", repository)
		}

		repoMetadata := &RepositoryMetadata{
			UUID:     repo.UUID,
			Name:     repo.Name,
			FullName: repo.FullName,
			Slug:     repo.Slug,
		}

		return repoMetadata, ctx.Set(NodeMetadata{Repository: repoMetadata})
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

// requireRepositoryInWorkspace rejects a repository outside the integration workspace.
// Setup cannot check an expression, so each request checks the resolved value.
func requireRepositoryInWorkspace(integration core.IntegrationContext, repository string) error {
	repository = strings.TrimSpace(repository)
	workspace, slug, ok := strings.Cut(repository, "/")
	slug = strings.TrimSuffix(strings.TrimSpace(slug), ".git")
	if !ok || strings.TrimSpace(workspace) == "" || slug == "" || strings.Contains(slug, "/") {
		return fmt.Errorf("repository must be in workspace/repository format: %q", repository)
	}

	var metadata Metadata
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}
	if metadata.AuthType == AuthTypeRepositoryAccessToken {
		if metadata.Repository == nil || !repositoryMetadataMatches(*metadata.Repository, repository) {
			return fmt.Errorf("repository %s is not accessible to workspace", repository)
		}
		return nil
	}
	configured := ""
	if metadata.Workspace != nil {
		configured = strings.TrimSpace(metadata.Workspace.Slug)
	}
	if configured == "" || !strings.EqualFold(configured, strings.TrimSpace(workspace)) {
		return fmt.Errorf("repository %s is not accessible to workspace", repository)
	}
	return nil
}

func repositoryMetadataMatches(repo RepositoryMetadata, repository string) bool {
	return repo.FullName == repository || repo.Name == repository || repo.Slug == repository || repo.UUID == repository
}

func repositoryMatches(repo Repository, repository string) bool {
	return repo.FullName == repository || repo.Name == repository || repo.Slug == repository || repo.UUID == repository
}
