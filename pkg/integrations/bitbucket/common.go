package bitbucket

import (
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type NodeMetadata struct {
	Repository *Repository `json:"repository" mapstructure:"repository"`
}

func ensureRepoInMetadata(ctx core.MetadataContext, integration core.IntegrationContext, repository string) error {
	if repository == "" {
		return fmt.Errorf("repository is required")
	}

	var nodeMetadata NodeMetadata
	if err := mapstructure.Decode(ctx.Get(), &nodeMetadata); err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	if nodeMetadata.Repository != nil && repositoryMatches(*nodeMetadata.Repository, repository) {
		return nil
	}

	var integrationMetadata Metadata
	if err := mapstructure.Decode(integration.GetMetadata(), &integrationMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	repoIndex := slices.IndexFunc(integrationMetadata.Repositories, func(r Repository) bool {
		return repositoryMatches(r, repository)
	})
	if repoIndex == -1 {
		return fmt.Errorf("repository %s is not accessible to workspace", repository)
	}

	return ctx.Set(NodeMetadata{Repository: &integrationMetadata.Repositories[repoIndex]})
}

func repositoryMatches(repo Repository, repository string) bool {
	return repo.FullName == repository || repo.Name == repository || repo.Slug == repository || repo.UUID == repository
}
