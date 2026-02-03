package aws

import (
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/ecr"
	"github.com/superplanehq/superplane/pkg/integrations/aws/lambda"
)

func (a *AWS) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "lambda.function":
		return lambda.ListFunctions(ctx, resourceType)

	case "ecr.repository":
		return ecr.ListRepositories(ctx, resourceType)

	default:
		return []core.IntegrationResource{}, nil
	}
}
