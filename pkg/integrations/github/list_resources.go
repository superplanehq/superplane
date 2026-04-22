package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

func (g *GitHub) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "repository":
		return g.listRepositoryResources(ctx)
	case "branch":
		return g.listBranchResources(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func (g *GitHub) listRepositoryResources(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
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

	// Fetch repositories accessible to the installation from GitHub API
	// This ensures we always get the latest list including newly created repos
	var allRepos []*github.Repository
	opts := &github.ListOptions{
		PerPage: 100, // Maximum per page
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

func (g *GitHub) listBranchResources(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	repository := ctx.Parameters["repository"]
	if repository == "" {
		return []core.IntegrationResource{}, nil
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode application metadata: %w", err)
	}

	client, err := NewClient(ctx.Integration, metadata.GitHubApp.ID, metadata.InstallationID)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	var allBranches []*github.Branch
	opts := &github.BranchListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		branches, resp, err := client.Repositories.ListBranches(
			context.Background(),
			metadata.Owner,
			repository,
			opts,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to list branches: %w", err)
		}

		allBranches = append(allBranches, branches...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	resources := make([]core.IntegrationResource, 0, len(allBranches))
	for _, branch := range allBranches {
		if branch.Name != nil {
			resources = append(resources, core.IntegrationResource{
				Type: "branch",
				Name: *branch.Name,
				ID:   *branch.Name,
			})
		}
	}

	return resources, nil
}
