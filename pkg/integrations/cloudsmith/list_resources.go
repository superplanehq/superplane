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

// listPackages lists the packages in the repository passed as the "repository"
// parameter (the value of the Repository field, in the form owner/repository).
// The resource ID is the package's permanent slug, which the action combines
// with the chosen repository to fetch the package. Returns an empty list when
// the repository is unset or an unresolved expression.
func listPackages(ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	repositoryID := strings.TrimSpace(ctx.Parameters["repository"])
	if repositoryID == "" || strings.Contains(repositoryID, "{{") {
		return []core.IntegrationResource{}, nil
	}

	owner, identifier, err := parseRepositoryID(repositoryID)
	if err != nil {
		return nil, fmt.Errorf("invalid repository %q: %w", repositoryID, err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	packages, err := client.ListPackages(owner, identifier)
	if err != nil {
		return nil, fmt.Errorf("error listing packages: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(packages))
	for _, pkg := range packages {
		label := pkg.Name
		if pkg.Version != "" {
			label = fmt.Sprintf("%s %s", pkg.Name, pkg.Version)
		}
		if pkg.License != "" {
			label = fmt.Sprintf("%s (%s)", label, pkg.License)
		}

		resources = append(resources, core.IntegrationResource{
			Type: "package",
			Name: label,
			ID:   pkg.SlugPerm,
		})
	}

	return resources, nil
}
