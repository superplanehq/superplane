package sns

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func ListTopics(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("list SNS topics: region is required")
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("list SNS topics: failed to load AWS credentials from integration: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	topics, err := client.ListTopics()
	if err != nil {
		return nil, fmt.Errorf("list SNS topics: failed to list topics in region %q: %w", region, err)
	}

	var resources []core.IntegrationResource
	for _, topic := range topics {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: topic.Name,
			ID:   topic.TopicArn,
		})
	}

	return resources, nil
}

func ListSubscriptions(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	region := ctx.Parameters["region"]
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	topicArn := ctx.Parameters["topicArn"]
	if topicArn == "" {
		return nil, fmt.Errorf("topic ARN is required")
	}

	ctx.Logger.Infof("listing subscriptions for topic %q in region %q", topicArn, region)

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS credentials from integration: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	subscriptions, err := client.ListSubscriptionsByTopic(topicArn)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions in region %q: %w", region, err)
	}

	var resources []core.IntegrationResource
	for _, subscription := range subscriptions {
		parts := strings.Split(subscription.SubscriptionArn, ":")
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: parts[len(parts)-1],
			ID:   subscription.SubscriptionArn,
		})
	}

	return resources, nil
}
