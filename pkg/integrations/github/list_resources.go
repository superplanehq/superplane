package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

func (g *GitHub) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "repository":
		client, err := common.NewClient(ctx.Integration, ctx.HTTP)
		if err != nil {
			return nil, fmt.Errorf("failed to create client: %w", err)
		}

		repositories, err := client.ListRepositories()
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories: %w", err)
		}

		return toIntegrationResources(repositories), nil

	case "branch":
		return g.listBranchResources(ctx)

	case "default_branch":
		return g.listDefaultBranchResource(ctx)

	default:
		return []core.IntegrationResource{}, nil
	}
}

func toIntegrationResources(repositories []*github.Repository) []core.IntegrationResource {
	resources := make([]core.IntegrationResource, 0, len(repositories))
	for _, repo := range repositories {
		// Prefer owner/repo so consumers like claude.runCodeAgent get a cloneable
		// repository identifier (short names are ambiguous across owners).
		name := repo.GetFullName()
		if name == "" {
			name = repo.GetName()
		}
		resources = append(resources, core.IntegrationResource{
			Type: "repository",
			Name: name,
			ID:   fmt.Sprintf("%d", repo.GetID()),
		})
	}
	return resources
}

// listDefaultBranchResource resolves the default branch of a single
// repository, identified by ctx.Parameters["repository"]. It is used by
// onboarding to write the real default branch (main, master, staging, ...)
// into generated automations instead of hardcoding "main".
func (g *GitHub) listDefaultBranchResource(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	repository := ctx.Parameters["repository"]
	if repository == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	repo, err := client.FindRepository(repository)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	return toDefaultBranchResources(repo), nil
}

// toDefaultBranchResources returns the resolved default branch as a single
// IntegrationResource, falling back to "main" when GitHub reports no default
// branch (this can happen for empty repositories).
func toDefaultBranchResources(repo *github.Repository) []core.IntegrationResource {
	branch := repo.GetDefaultBranch()
	if branch == "" {
		branch = "main"
	}

	return []core.IntegrationResource{
		{
			Type: "default_branch",
			Name: branch,
			ID:   branch,
		},
	}
}

func (g *GitHub) listBranchResources(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	repository := ctx.Parameters["repository"]
	if repository == "" {
		return []core.IntegrationResource{}, nil
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	var allBranches []*github.Branch
	opts := &github.BranchListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		branches, resp, err := client.ListBranches(context.Background(), repository, opts)
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
