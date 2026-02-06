package circleci

import (
	"github.com/superplanehq/superplane/pkg/core"
)

func (c *CircleCI) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	// CircleCI doesn't have a concept of listable resources like projects
	// Users need to manually enter their project slug
	return []core.IntegrationResource{}, nil
}
