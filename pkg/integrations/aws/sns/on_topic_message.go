package sns

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type OnTopicMessageConfiguration struct {
	Region   string `json:"region" mapstructure:"region"`
	TopicArn string `json:"topicArn" mapstructure:"topicArn"`
}

type OnTopicMessageMetadata struct {
	Region   string `json:"region" mapstructure:"region"`
	TopicArn string `json:"topicArn" mapstructure:"topicArn"`
}

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
	var config OnTopicMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode trigger configuration: %w", err)
	}

	var metadata OnTopicMessageMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode trigger metadata: %w", err)
	}

	region, err := requireRegion(config.Region)
	if err != nil {
		return fmt.Errorf("invalid region: %w", err)
	}

	topicArn, err := requireTopicArn(config.TopicArn)
	if err != nil {
		return fmt.Errorf("invalid topic ARN: %w", err)
	}

	if metadata.Region == region && metadata.TopicArn == topicArn {
		return nil
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials from integration: %w", err)
	}

	client := NewClient(ctx.HTTP, credentials, region)
	topic, err := client.GetTopic(topicArn)
	if err != nil {
		return fmt.Errorf("failed to get topic %q in region %q: %w", topicArn, region, err)
	}

	err = ctx.Metadata.Set(OnTopicMessageMetadata{
		Region:   region,
		TopicArn: topicArn,
	})

	if err != nil {
		return fmt.Errorf("failed to persist trigger metadata: %w", err)
	}

	return ctx.Integration.RequestWebhook(common.WebhookConfiguration{
		Region: region,
		Type:   common.WebhookTypeSNS,
		SNS: &common.SNSWebhookConfiguration{
			TopicArn: topic.TopicArn,
		},
	})
}

func (t *OnTopicMessage) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnTopicMessage) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

type SubscriptionMessage struct {
	Type              string                      `json:"Type"`
	MessageID         string                      `json:"MessageId"`
	TopicArn          string                      `json:"TopicArn"`
	Subject           string                      `json:"Subject"`
	Message           string                      `json:"Message"`
	Timestamp         string                      `json:"Timestamp"`
	SignatureVersion  string                      `json:"SignatureVersion"`
	Signature         string                      `json:"Signature"`
	SigningCertURL    string                      `json:"SigningCertURL"`
	UnsubscribeURL    string                      `json:"UnsubscribeURL"`
	SubscribeURL      string                      `json:"SubscribeURL"`
	Token             string                      `json:"Token"`
	MessageAttributes map[string]MessageAttribute `json:"MessageAttributes"`
}

type MessageAttribute struct {
	Type  string `json:"Type"`
	Value string `json:"Value"`
}

func (t *OnTopicMessage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	var config OnTopicMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode trigger configuration: %w", err)
	}

	var message SubscriptionMessage
	if err := json.Unmarshal(ctx.Body, &message); err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to decode SNS webhook payload: %w", err)
	}

	switch message.Type {
	case "SubscriptionConfirmation":
		return t.confirmSubscription(ctx, config, message)

	case "Notification":
		return t.emitTopicNotification(ctx, message, config)

	case "UnsubscribeConfirmation":
		return http.StatusOK, nil

	default:
		return http.StatusBadRequest, fmt.Errorf("unsupported SNS message type %q", message.Type)
	}
}

func (t *OnTopicMessage) confirmSubscription(ctx core.WebhookRequestContext, config OnTopicMessageConfiguration, message SubscriptionMessage) (int, error) {
	if strings.TrimSpace(message.TopicArn) != config.TopicArn {
		ctx.Logger.Infof("message topic ARN %s does not match configured topic ARN %s, ignoring", message.TopicArn, config.TopicArn)
		return http.StatusOK, nil
	}

	if message.SubscribeURL == "" {
		ctx.Logger.Errorf("missing SubscribeURL")
		return http.StatusBadRequest, fmt.Errorf("missing SubscribeURL")
	}

	subscribeURL, err := url.Parse(message.SubscribeURL)
	if err != nil {
		ctx.Logger.Errorf("invalid SubscribeURL: %v", err)
		return http.StatusBadRequest, fmt.Errorf("invalid SubscribeURL: %w", err)
	}

	if subscribeURL.Scheme != "https" {
		ctx.Logger.Errorf("SubscribeURL must use https")
		return http.StatusBadRequest, fmt.Errorf("SubscribeURL must use https")
	}

	host := strings.ToLower(subscribeURL.Hostname())
	if host == "" {
		ctx.Logger.Errorf("SubscribeURL host is required")
		return http.StatusBadRequest, fmt.Errorf("SubscribeURL host is required")
	}

	if !strings.HasSuffix(host, ".amazonaws.com") && !strings.HasSuffix(host, ".amazonaws.com.cn") {
		ctx.Logger.Errorf("SubscribeURL host must be an AWS SNS domain")
		return http.StatusBadRequest, fmt.Errorf("SubscribeURL host must be an AWS SNS domain")
	}

	req, err := http.NewRequest(http.MethodGet, subscribeURL.String(), nil)
	if err != nil {
		ctx.Logger.Errorf("failed to create request to confirm subscription: %v", err)
		return http.StatusInternalServerError, fmt.Errorf("failed to create request: %w", err)
	}

	response, err := ctx.HTTP.Do(req)
	if err != nil {
		ctx.Logger.Errorf("failed to confirm SNS subscription: %v", err)
		return http.StatusInternalServerError, fmt.Errorf("failed to confirm SNS subscription: %w", err)
	}

	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseBody, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			ctx.Logger.Errorf("failed to read response body: %v", readErr)
			return http.StatusInternalServerError, fmt.Errorf(
				"SNS subscription confirmation failed with status %d and unreadable body: %v",
				response.StatusCode,
				readErr,
			)
		}

		ctx.Logger.Errorf("SNS subscription confirmation failed with status %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
		return http.StatusInternalServerError, fmt.Errorf(
			"SNS subscription confirmation failed with status %d: %s",
			response.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	ctx.Logger.Info("Subscription confirmation was successful")
	return http.StatusOK, nil
}

func (t *OnTopicMessage) emitTopicNotification(ctx core.WebhookRequestContext, message SubscriptionMessage, config OnTopicMessageConfiguration) (int, error) {
	topicArn := strings.TrimSpace(message.TopicArn)
	if topicArn == "" {
		ctx.Logger.Errorf("missing TopicArn in SNS notification payload")
		return http.StatusBadRequest, fmt.Errorf("missing TopicArn in SNS notification payload")
	}

	if topicArn != config.TopicArn {
		ctx.Logger.Infof("message topic ARN %s does not match configured topic ARN %s, ignoring", topicArn, config.TopicArn)
		return http.StatusOK, nil
	}

	if err := ctx.Events.Emit("aws.sns.topic.message", message); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit topic message event: %w", err)
	}

	return http.StatusOK, nil
}

func (t *OnTopicMessage) Cleanup(ctx core.TriggerContext) error {
	return nil
}
