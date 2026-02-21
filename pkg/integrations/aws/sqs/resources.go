package sqs

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func ListQueues(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := ctx.Parameters["region"]
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, credentials, region)
	queues, err := client.ListQueues("")
	if err != nil {
		return nil, fmt.Errorf("failed to list SQS queues: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(queues))
	for _, queue := range queues {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: queue.Name,
			ID:   queue.URL,
		})
	}

	return resources, nil
}
