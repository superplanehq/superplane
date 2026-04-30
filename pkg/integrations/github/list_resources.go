package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v84/github"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func (g *GitHub) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if ctx.Integration.LegacySetup() {
		return g.legacyListResources(resourceType, ctx)
	}

	switch resourceType {
	case "repository":
		return g.listRepositories(ctx)
	}

	return []core.IntegrationResource{}, nil
}

func (g *GitHub) legacyListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "repository" {
		return []core.IntegrationResource{}, nil
	}

	// Decode metadata to get GitHub App ID and Installation ID
	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode application metadata: %w", err)
	}

	// Create GitHub client
	client, err := NewClient(ctx.Integration, metadata.GitHubApp.ID, metadata.InstallationID)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	return g.listAppRepositories(client)
}

func (g *GitHub) listRepositories(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	authMethod, err := ctx.ParameterStorage.GetString(ParameterAuthMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication method: %w", err)
	}

	client, err := NewClientFromStorageContexts(ctx.ParameterStorage, ctx.Secrets)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	switch authMethod {
	case AuthMethodPAT:
		return g.listOwnerRepositories(client)
	case AuthMethodGitHubApp:
		return g.listAppRepositories(client)
	}

	return []core.IntegrationResource{}, nil
}

func (g *GitHub) listOwnerRepositories(client *github.Client) ([]core.IntegrationResource, error) {
	var allRepos []*github.Repository
	opts := &github.RepositoryListByAuthenticatedUserOptions{
		Affiliation: "owner",
		Sort:        "updated",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		repos, resp, err := client.Repositories.ListByAuthenticatedUser(context.Background(), opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories from GitHub API: %w", err)
		}

		allRepos = append(allRepos, repos...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	resources := make([]core.IntegrationResource, 0, len(allRepos))
	for _, repo := range allRepos {
		if repo.Name != nil && repo.ID != nil {
			resources = append(resources, core.IntegrationResource{
				Type: "repository",
				Name: *repo.Name,
				ID:   fmt.Sprintf("%d", *repo.ID),
			})
		}
	}

	return resources, nil
}

func (g *GitHub) listAppRepositories(client *github.Client) ([]core.IntegrationResource, error) {
	var allRepos []*github.Repository
	opts := &github.ListOptions{
		PerPage: 100,
	}

	for {
		repos, resp, err := client.Apps.ListRepos(context.Background(), opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories from GitHub API: %w", err)
		}

		allRepos = append(allRepos, repos.Repositories...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	resources := make([]core.IntegrationResource, 0, len(allRepos))
	for _, repo := range allRepos {
		if repo.Name != nil && repo.ID != nil {
			resources = append(resources, core.IntegrationResource{
				Type: "repository",
				Name: *repo.Name,
				ID:   fmt.Sprintf("%d", *repo.ID),
			})
		}
	}

	return resources, nil
}
