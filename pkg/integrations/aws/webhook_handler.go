package aws

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/pkg/integrations/aws/sns"
)

type WebhookHandler struct{}

func (h *WebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	var config common.WebhookConfiguration
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("failed to decode SNS webhook configuration: %w", err)
	}

	switch config.Type {
	case common.WebhookTypeSNS:
		return h.setupSNS(ctx, config)
	}

	return nil, fmt.Errorf("setup: unsupported webhook type: %s", config.Type)
}

func (h *WebhookHandler) setupSNS(ctx core.WebhookHandlerContext, config common.WebhookConfiguration) (any, error) {
	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS credentials from integration: %w", err)
	}

	client := sns.NewClient(ctx.HTTP, credentials, config.Region)
	subscription, err := client.Subscribe(sns.SubscribeParameters{
		TopicArn:              config.SNS.TopicArn,
		Protocol:              "https",
		Endpoint:              ctx.Webhook.GetURL(),
		ReturnSubscriptionARN: true,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to SNS topic %q in region %q: %w", config.SNS.TopicArn, config.Region, err)
	}

	return common.SNSWebhookMetadata{
		SubscriptionArn: subscription.SubscriptionArn,
	}, nil
}

func (h *WebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	var config common.WebhookConfiguration
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return fmt.Errorf("failed to decode SNS webhook configuration: %w", err)
	}

	switch config.Type {
	case common.WebhookTypeSNS:
		return h.cleanupSNS(ctx, config.Region)
	default:
		return fmt.Errorf("cleanup: unsupported webhook type: %s", config.Type)
	}
}

func (h *WebhookHandler) cleanupSNS(ctx core.WebhookHandlerContext, region string) error {
	metadata := common.SNSWebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode SNS webhook metadata: %w", err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("cleanup SNS: failed to load AWS credentials from integration: %w", err)
	}

	client := sns.NewClient(ctx.HTTP, credentials, region)
	err = client.Unsubscribe(metadata.SubscriptionArn)
	if err != nil && !common.IsNotFoundErr(err) {
		return fmt.Errorf("cleanup SNS: failed to unsubscribe existing subscription %q in region %q: %w", metadata.SubscriptionArn, region, err)
	}

	return nil
}

func (h *WebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := common.WebhookConfiguration{}
	configB := common.WebhookConfiguration{}

	err := mapstructure.Decode(a, &configA)
	if err != nil {
		return false, err
	}

	err = mapstructure.Decode(b, &configB)
	if err != nil {
		return false, err
	}

	if configA.Type != configB.Type {
		return false, nil
	}

	if configA.Type == common.WebhookTypeSNS {
		return configA.SNS.TopicArn == configB.SNS.TopicArn, nil
	}

	return false, nil
}

func (h *WebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}
