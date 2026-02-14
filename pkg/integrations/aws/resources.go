package aws

import (
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/codeartifact"
	"github.com/superplanehq/superplane/pkg/integrations/aws/ecr"
	"github.com/superplanehq/superplane/pkg/integrations/aws/ecs"
	"github.com/superplanehq/superplane/pkg/integrations/aws/lambda"
	"github.com/superplanehq/superplane/pkg/integrations/aws/sns"
)

func (a *AWS) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "lambda.function":
		return lambda.ListFunctions(ctx, resourceType)

	case "ecr.repository":
		return ecr.ListRepositories(ctx, resourceType)

	case "ecs.cluster":
		return ecs.ListClusters(ctx, resourceType)

	case "ecs.service":
		return ecs.ListServices(ctx, resourceType)

	case "ecs.taskDefinition":
		return ecs.ListTaskDefinitions(ctx, resourceType)

	case "ecs.task":
		return ecs.ListTasks(ctx, resourceType)

	case "codeartifact.repository":
		return codeartifact.ListRepositories(ctx, resourceType)

	case "codeartifact.domain":
		return codeartifact.ListDomains(ctx, resourceType)

	case "sns.topic":
		return sns.ListTopics(ctx, resourceType)

	case "sns.subscription":
		return sns.ListSubscriptions(ctx, resourceType)

	default:
		return []core.IntegrationResource{}, nil
	}
}
