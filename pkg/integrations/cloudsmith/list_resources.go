package cloudsmith

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

func (c *Cloudsmith) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "repository":
		return listRepositories(ctx)
	case "package":
		return listPackages(ctx)
	case "organization":
		return listOrganizations(ctx)
	case "vulnerabilityPolicy":
		return listVulnerabilityPolicies(ctx)
	default:
		return []core.IntegrationResource{}, nil
	}
}

func listRepositories(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	repositories, err := client.ListRepositories()
	if err != nil {
		return nil, fmt.Errorf("error listing repositories: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(repositories))
	for _, repository := range repositories {
		name := repository.Name
		if name == "" {
			name = repository.Slug
		}

		resources = append(resources, core.IntegrationResource{
			Type: "repository",
			Name: fmt.Sprintf("%s/%s", repository.Namespace, name),
			ID:   fmt.Sprintf("%s/%s", repository.Namespace, repository.Slug),
		})
	}

	return resources, nil
}

func listPackages(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	repositoryID := ctx.Parameters["repository"]
	if repositoryID == "" || strings.Contains(repositoryID, "{{") {
		return []core.IntegrationResource{}, nil
	}

	owner, repo, err := parseRepositoryID(repositoryID)
	if err != nil {
		return nil, fmt.Errorf("invalid repository %q: %w", repositoryID, err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	packages, err := client.ListPackages(owner, repo)
	if err != nil {
		return nil, fmt.Errorf("error listing packages: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(packages))
	for _, pkg := range packages {
		name := pkg.Name
		if name == "" {
			name = pkg.SlugPerm
		}
		if pkg.Version != "" {
			name = fmt.Sprintf("%s %s", name, pkg.Version)
		}
		resources = append(resources, core.IntegrationResource{
			Type: "package",
			Name: name,
			ID:   pkg.SlugPerm,
		})
	}

	return resources, nil
}

func listOrganizations(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	organizations, err := client.ListOrganizations()
	if err != nil {
		return nil, fmt.Errorf("error listing organizations: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(organizations))
	for _, organization := range organizations {
		name := organization.Name
		if name == "" {
			name = organization.Slug
		}
		resources = append(resources, core.IntegrationResource{
			Type: "organization",
			Name: name,
			ID:   organization.Slug,
		})
	}

	return resources, nil
}

func listVulnerabilityPolicies(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	org := ctx.Parameters["organization"]
	if org == "" || strings.Contains(org, "{{") {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	policies, err := client.ListVulnerabilityPolicies(org)
	if err != nil {
		return nil, fmt.Errorf("error listing vulnerability policies: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(policies))
	for _, policy := range policies {
		name := policy.Name
		if name == "" {
			name = policy.SlugPerm
		}
		resources = append(resources, core.IntegrationResource{
			Type: "vulnerabilityPolicy",
			Name: name,
			ID:   policy.SlugPerm,
		})
	}

	return resources, nil
}
