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

	default:
		return []core.IntegrationResource{}, nil
	}
}

func toIntegrationResources(repositories []*github.Repository) []core.IntegrationResource {
	resources := make([]core.IntegrationResource, 0, len(repositories))
	for _, repo := range repositories {
		resources = append(resources, core.IntegrationResource{
			Type: "repository",
			Name: repo.GetName(),
			ID:   fmt.Sprintf("%d", repo.GetID()),
		})
	}
	return resources
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
