package pubsub

import (
	"context"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
)

const (
	ResourceTypeTopic        = "gcp.pubsub.topic"
	ResourceTypeSubscription = "gcp.pubsub.subscription"
)

func ListTopicResources(ctx context.Context, client *gcpcommon.Client) ([]core.IntegrationResource, error) {
	projectID := client.ProjectID()
	topics, err := ListTopics(ctx, client, projectID)
	if err != nil {
		return nil, fmt.Errorf("list topics: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(topics))
	for _, t := range topics {
		short := TopicShortName(t.Name)
		resources = append(resources, core.IntegrationResource{
			ID:   short,
			Name: short,
		})
	}
	return resources, nil
}

func ListSubscriptionResources(ctx context.Context, client *gcpcommon.Client, topic string) ([]core.IntegrationResource, error) {
	projectID := client.ProjectID()
	subs, err := ListSubscriptions(ctx, client, projectID)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}

	topic = normalizeTopicName(topic)
	resources := make([]core.IntegrationResource, 0, len(subs))
	for _, s := range subs {
		if topic != "" && TopicShortName(s.Topic) != topic {
			continue
		}

		short := SubscriptionShortName(s.Name)
		resources = append(resources, core.IntegrationResource{
			ID:   short,
			Name: short,
		})
	}
	return resources, nil
}

func normalizeTopicName(topic string) string {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return ""
	}

	if strings.Contains(topic, "/") {
		return TopicShortName(topic)
	}

	return topic
}
