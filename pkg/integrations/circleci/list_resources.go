package circleci

import (
	"github.com/superplanehq/superplane/pkg/core"
)

func (c *CircleCI) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}
