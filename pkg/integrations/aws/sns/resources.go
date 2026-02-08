package sns

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

// ListTopics lists SNS topics for integration resource selectors.
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

// ListSubscriptions lists SNS subscriptions for integration resource selectors.
func ListSubscriptions(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("list SNS subscriptions: region is required")
	}

	topicArn := strings.TrimSpace(ctx.Parameters["topicArn"])
	if topicArn == "" {
		topicArn = strings.TrimSpace(ctx.Parameters["topic"])
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("list SNS subscriptions: failed to load AWS credentials from integration: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	subscriptions, err := client.ListSubscriptions(topicArn)
	if err != nil {
		return nil, fmt.Errorf("list SNS subscriptions: failed to list subscriptions in region %q: %w", region, err)
	}

	var resources []core.IntegrationResource
	for _, subscription := range subscriptions {
		name := strings.TrimSpace(subscription.Endpoint)
		if name == "" {
			name = strings.TrimSpace(subscription.SubscriptionArn)
		}
		if name == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: name,
			ID:   subscription.SubscriptionArn,
		})
	}

	return resources, nil
}
