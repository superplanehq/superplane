package sns

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type OnTopicMessage struct{}

func (t *OnTopicMessage) Name() string {
	return "aws.sns.onTopicMessage"
}

func (t *OnTopicMessage) Label() string {
	return "SNS • On Topic Message"
}

func (t *OnTopicMessage) Description() string {
	return "Listen to AWS SNS topic notifications"
}

func (t *OnTopicMessage) Documentation() string {
	return `The On Topic Message trigger starts a workflow execution when a message is published to an AWS SNS topic.

## Use Cases

- **Event-driven automation**: React to messages published by external systems
- **Notification processing**: Handle SNS payloads in workflow steps
- **Routing and enrichment**: Trigger downstream workflows based on topic activity

## How it works

During setup, SuperPlane creates a webhook endpoint for this trigger and subscribes it to the selected SNS topic using HTTPS. SNS sends notification payloads to the webhook endpoint, which then emits workflow events.`
}

func (t *OnTopicMessage) Icon() string {
	return "aws"
}

func (t *OnTopicMessage) Color() string {
	return "gray"
}

func (t *OnTopicMessage) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		topicField(),
	}
}

func (t *OnTopicMessage) Setup(ctx core.TriggerContext) error {
	scope := t.Name()

	var config OnTopicMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: failed to decode trigger configuration: %w", scope, err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return fmt.Errorf("%s: invalid region: %w", scope, err)
	}

	topicArn, err := requireTopicArn(config.TopicArn)
	if err != nil {
		return fmt.Errorf("%s: invalid topic ARN: %w", scope, err)
	}

	if _, err := validateTopic(ctx.HTTP, ctx.Integration, region, topicArn); err != nil {
		return fmt.Errorf("%s: failed to validate topic %q: %w", scope, topicArn, err)
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("%s: failed to setup webhook: %w", scope, err)
	}

	protocol, err := requireWebhookProtocol(webhookURL)
	if err != nil {
		return fmt.Errorf("%s: invalid webhook URL %q: %w", scope, webhookURL, err)
	}

	var metadata OnTopicMessageMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("%s: failed to decode trigger metadata: %w", scope, err)
	}

	if metadata.Region == region &&
		metadata.TopicArn == topicArn &&
		metadata.WebhookURL == webhookURL &&
		strings.TrimSpace(metadata.SubscriptionArn) != "" {
		return nil
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("%s: failed to load AWS credentials from integration: %w", scope, err)
	}

	if err := cleanupExistingSubscription(ctx.HTTP, credentials, metadata); err != nil {
		return fmt.Errorf("%s: failed to cleanup existing subscription: %w", scope, err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	subscription, err := client.Subscribe(SubscribeParameters{
		TopicArn:              topicArn,
		Protocol:              protocol,
		Endpoint:              webhookURL,
		ReturnSubscriptionARN: true,
	})
	if err != nil {
		return fmt.Errorf("%s: failed to subscribe webhook endpoint to topic %q: %w", scope, topicArn, err)
	}

	subscriptionArn := strings.TrimSpace(subscription.SubscriptionArn)
	if subscriptionArn == "" {
		return fmt.Errorf("%s: subscription response did not include subscription ARN", scope)
	}

	if err := ctx.Metadata.Set(OnTopicMessageMetadata{
		Region:          region,
		TopicArn:        topicArn,
		WebhookURL:      webhookURL,
		SubscriptionArn: subscriptionArn,
	}); err != nil {
		return fmt.Errorf("%s: failed to persist trigger metadata: %w", scope, err)
	}

	return nil
}

func cleanupExistingSubscription(httpCtx core.HTTPContext, credentials *aws.Credentials, metadata OnTopicMessageMetadata) error {
	subscriptionArn := strings.TrimSpace(metadata.SubscriptionArn)
	if subscriptionArn == "" || strings.EqualFold(subscriptionArn, "pending confirmation") {
		return nil
	}

	region := strings.TrimSpace(metadata.Region)
	if region == "" {
		return nil
	}

	client := NewClient(httpCtx, credentials, region)
	if err := client.Unsubscribe(subscriptionArn); err != nil && !common.IsNotFoundErr(err) {
		return fmt.Errorf("unsubscribe existing subscription %q in region %q: %w", subscriptionArn, region, err)
	}

	return nil
}

func requireWebhookProtocol(webhookURL string) (string, error) {
	normalized := strings.TrimSpace(webhookURL)
	if normalized == "" {
		return "", fmt.Errorf("webhook URL is required")
	}

	parsedURL, err := url.Parse(normalized)
	if err != nil {
		return "", fmt.Errorf("parse webhook URL: %w", err)
	}

	protocol := strings.ToLower(strings.TrimSpace(parsedURL.Scheme))
	if protocol != "http" && protocol != "https" {
		return "", fmt.Errorf("webhook URL scheme must be http or https")
	}

	return protocol, nil
}

func (t *OnTopicMessage) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnTopicMessage) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnTopicMessage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	scope := t.Name()

	var config OnTopicMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("%s: failed to decode trigger configuration: %w", scope, err)
	}

	topicArn, err := requireTopicArn(config.TopicArn)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("%s: invalid configured topic ARN: %w", scope, err)
	}

	var message snsWebhookMessage
	if err := json.Unmarshal(ctx.Body, &message); err != nil {
		return http.StatusBadRequest, fmt.Errorf("%s: failed to decode SNS webhook payload: %w", scope, err)
	}

	switch strings.TrimSpace(message.Type) {
	case "SubscriptionConfirmation":
		return confirmSNSSubscription(message.SubscribeURL, topicArn, message.TopicArn)
	case "Notification":
		return t.emitTopicNotification(ctx, message, topicArn)
	case "UnsubscribeConfirmation":
		return http.StatusOK, nil
	default:
		return http.StatusBadRequest, fmt.Errorf("%s: unsupported SNS message type %q", scope, message.Type)
	}
}

func confirmSNSSubscription(subscribeURL string, configuredTopicArn string, payloadTopicArn string) (int, error) {
	if strings.TrimSpace(payloadTopicArn) != configuredTopicArn {
		return http.StatusOK, nil
	}

	normalizedSubscribeURL := strings.TrimSpace(subscribeURL)
	if normalizedSubscribeURL == "" {
		return http.StatusBadRequest, fmt.Errorf("aws.sns.onTopicMessage: missing SubscribeURL in SNS subscription confirmation")
	}

	parsedURL, err := url.Parse(normalizedSubscribeURL)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("aws.sns.onTopicMessage: invalid SubscribeURL: %w", err)
	}

	if parsedURL.Scheme != "https" {
		return http.StatusBadRequest, fmt.Errorf("aws.sns.onTopicMessage: SubscribeURL must use https")
	}

	host := strings.ToLower(strings.TrimSpace(parsedURL.Hostname()))
	if host == "" {
		return http.StatusBadRequest, fmt.Errorf("aws.sns.onTopicMessage: SubscribeURL host is required")
	}

	if !strings.HasSuffix(host, ".amazonaws.com") && !strings.HasSuffix(host, ".amazonaws.com.cn") {
		return http.StatusBadRequest, fmt.Errorf("aws.sns.onTopicMessage: SubscribeURL host must be an AWS SNS domain")
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// #nosec G107 -- SubscribeURL is validated as HTTPS AWS SNS domain before request.
	response, err := client.Get(normalizedSubscribeURL)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("aws.sns.onTopicMessage: failed to confirm SNS subscription: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseBody, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return http.StatusInternalServerError, fmt.Errorf(
				"aws.sns.onTopicMessage: SNS subscription confirmation failed with status %d and unreadable body: %w",
				response.StatusCode,
				readErr,
			)
		}

		return http.StatusInternalServerError, fmt.Errorf(
			"aws.sns.onTopicMessage: SNS subscription confirmation failed with status %d: %s",
			response.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	return http.StatusOK, nil
}

func (t *OnTopicMessage) emitTopicNotification(
	ctx core.WebhookRequestContext,
	message snsWebhookMessage,
	configuredTopicArn string,
) (int, error) {
	topicArn := strings.TrimSpace(message.TopicArn)
	if topicArn == "" {
		return http.StatusBadRequest, fmt.Errorf("%s: missing TopicArn in SNS notification payload", t.Name())
	}

	if topicArn != configuredTopicArn {
		return http.StatusOK, nil
	}

	region, account := parseTopicArnContext(topicArn)
	eventPayload := map[string]any{
		"type":              "Notification",
		"messageId":         strings.TrimSpace(message.MessageID),
		"topicArn":          topicArn,
		"subject":           strings.TrimSpace(message.Subject),
		"message":           message.Message,
		"timestamp":         strings.TrimSpace(message.Timestamp),
		"region":            region,
		"account":           account,
		"messageAttributes": message.MessageAttributes,
		"detail": map[string]any{
			"messageId": strings.TrimSpace(message.MessageID),
			"topicArn":  topicArn,
			"subject":   strings.TrimSpace(message.Subject),
			"message":   message.Message,
			"timestamp": strings.TrimSpace(message.Timestamp),
		},
	}

	if err := ctx.Events.Emit("aws.sns.topic.message", eventPayload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("%s: failed to emit topic message event: %w", t.Name(), err)
	}

	return http.StatusOK, nil
}

func parseTopicArnContext(topicArn string) (string, string) {
	parts := strings.Split(topicArn, ":")
	if len(parts) < 6 {
		return "", ""
	}

	return strings.TrimSpace(parts[3]), strings.TrimSpace(parts[4])
}

func (t *OnTopicMessage) Cleanup(ctx core.TriggerContext) error {
	scope := t.Name()

	var metadata OnTopicMessageMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("%s: failed to decode trigger metadata during cleanup: %w", scope, err)
	}

	subscriptionArn := strings.TrimSpace(metadata.SubscriptionArn)
	if subscriptionArn == "" || strings.EqualFold(subscriptionArn, "pending confirmation") {
		return nil
	}

	region, err := requireRegion(metadata.Region)
	if err != nil {
		return fmt.Errorf("%s: invalid metadata region during cleanup: %w", scope, err)
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("%s: failed to load AWS credentials from integration during cleanup: %w", scope, err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	if err := client.Unsubscribe(subscriptionArn); err != nil && !common.IsNotFoundErr(err) {
		return fmt.Errorf("%s: failed to cleanup subscription %q: %w", scope, subscriptionArn, err)
	}

	return nil
}
