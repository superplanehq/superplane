package sns

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

// OnTopicMessage triggers workflow runs when SNS topic messages are published.
type OnTopicMessage struct{}

// Name returns the trigger name.
func (t *OnTopicMessage) Name() string {
	return "aws.sns.onTopicMessage"
}

// Label returns the trigger label.
func (t *OnTopicMessage) Label() string {
	return "SNS • On Topic Message"
}

// Description returns a short trigger description.
func (t *OnTopicMessage) Description() string {
	return "Listen to AWS SNS topic notification events"
}

// Documentation returns detailed Markdown documentation.
func (t *OnTopicMessage) Documentation() string {
	return `The On Topic Message trigger starts a workflow execution when a message is published to an AWS SNS topic.

## Use Cases

- **Event-driven automation**: React to messages published by external systems
- **Notification processing**: Handle SNS payloads in workflow steps
- **Routing and enrichment**: Trigger downstream workflows based on topic activity`
}

// Icon returns the icon slug.
func (t *OnTopicMessage) Icon() string {
	return "aws"
}

// Color returns the trigger color.
func (t *OnTopicMessage) Color() string {
	return "gray"
}

// Configuration returns the trigger configuration schema.
func (t *OnTopicMessage) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		topicField(),
	}
}

// Setup validates configuration and provisions EventBridge routing when required.
func (t *OnTopicMessage) Setup(ctx core.TriggerContext) error {
	scope := t.Name()

	var metadata OnTopicMessageMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("%s: failed to decode trigger metadata: %w", scope, err)
	}

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

	if metadata.SubscriptionID != "" && metadata.Region == region && metadata.TopicArn == topicArn {
		return nil
	}

	var integrationMetadata common.IntegrationMetadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return fmt.Errorf("%s: failed to decode integration metadata: %w", scope, err)
	}

	rules := map[string]common.EventBridgeRuleMetadata{}
	if integrationMetadata.EventBridge != nil && integrationMetadata.EventBridge.Rules != nil {
		rules = integrationMetadata.EventBridge.Rules
	}

	rule, ok := rules[Source]
	if !ok || !slices.Contains(rule.DetailTypes, DetailTypeTopicNotification) {
		if err := ctx.Metadata.Set(OnTopicMessageMetadata{
			Region:   region,
			TopicArn: topicArn,
		}); err != nil {
			return fmt.Errorf("%s: failed to set trigger metadata: %w", scope, err)
		}

		return t.provisionRule(ctx.Integration, ctx.Requests, region)
	}

	subscriptionID, err := ctx.Integration.Subscribe(t.subscriptionPattern(region))
	if err != nil {
		return fmt.Errorf("%s: failed to subscribe trigger to EventBridge pattern: %w", scope, err)
	}

	if err := ctx.Metadata.Set(OnTopicMessageMetadata{
		Region:         region,
		TopicArn:       topicArn,
		SubscriptionID: subscriptionID.String(),
	}); err != nil {
		return fmt.Errorf("%s: failed to persist trigger metadata: %w", scope, err)
	}

	return nil
}

// provisionRule schedules integration-level EventBridge rule provisioning.
func (t *OnTopicMessage) provisionRule(integration core.IntegrationContext, requests core.RequestContext, region string) error {
	if err := integration.ScheduleActionCall("provisionRule", common.ProvisionRuleParameters{
		Region:     region,
		Source:     Source,
		DetailType: DetailTypeTopicNotification,
	}, time.Second); err != nil {
		return fmt.Errorf("%s: failed to schedule integration rule provisioning in region %q: %w", t.Name(), region, err)
	}

	if err := requests.ScheduleActionCall("checkRuleAvailability", map[string]any{}, 5*time.Second); err != nil {
		return fmt.Errorf("%s: failed to schedule rule availability check: %w", t.Name(), err)
	}

	return nil
}

// subscriptionPattern builds the integration subscription selector.
func (t *OnTopicMessage) subscriptionPattern(region string) *common.EventBridgeEvent {
	return &common.EventBridgeEvent{
		Region:     region,
		DetailType: DetailTypeTopicNotification,
		Source:     Source,
	}
}

// Actions returns supported trigger actions.
func (t *OnTopicMessage) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "checkRuleAvailability",
			Description: "Check if an EventBridge rule is available",
		},
	}
}

// HandleAction handles custom trigger actions.
func (t *OnTopicMessage) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case "checkRuleAvailability":
		return t.checkRuleAvailability(ctx)
	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

// checkRuleAvailability retries until the integration rule is available.
func (t *OnTopicMessage) checkRuleAvailability(ctx core.TriggerActionContext) (map[string]any, error) {
	scope := t.Name()

	var metadata OnTopicMessageMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return nil, fmt.Errorf("%s: failed to decode trigger metadata: %w", scope, err)
	}

	var integrationMetadata common.IntegrationMetadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return nil, fmt.Errorf("%s: failed to decode integration metadata: %w", scope, err)
	}

	rules := map[string]common.EventBridgeRuleMetadata{}
	if integrationMetadata.EventBridge != nil && integrationMetadata.EventBridge.Rules != nil {
		rules = integrationMetadata.EventBridge.Rules
	}

	rule, ok := rules[Source]
	if !ok {
		ctx.Logger.Infof("Rule not found for source %s - checking again in 10 seconds", Source)
		if err := ctx.Requests.ScheduleActionCall("checkRuleAvailability", map[string]any{}, 10*time.Second); err != nil {
			return nil, fmt.Errorf("%s: failed to reschedule rule availability check: %w", scope, err)
		}

		return nil, nil
	}

	if !slices.Contains(rule.DetailTypes, DetailTypeTopicNotification) {
		ctx.Logger.Infof("Rule does not have detail type '%s' - checking again in 10 seconds", DetailTypeTopicNotification)
		if err := ctx.Requests.ScheduleActionCall("checkRuleAvailability", map[string]any{}, 10*time.Second); err != nil {
			return nil, fmt.Errorf("%s: failed to reschedule rule availability check for detail type: %w", scope, err)
		}

		return nil, nil
	}

	subscriptionID, err := ctx.Integration.Subscribe(t.subscriptionPattern(metadata.Region))
	if err != nil {
		return nil, fmt.Errorf("%s: failed to subscribe trigger to EventBridge pattern: %w", scope, err)
	}

	metadata.SubscriptionID = subscriptionID.String()
	if err := ctx.Metadata.Set(metadata); err != nil {
		return nil, fmt.Errorf("%s: failed to persist trigger metadata: %w", scope, err)
	}

	return nil, nil
}

// OnIntegrationMessage filters messages by topic and emits trigger payloads.
func (t *OnTopicMessage) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	var metadata OnTopicMessageMetadata
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata); err != nil {
		return fmt.Errorf("%s: failed to decode node metadata: %w", t.Name(), err)
	}

	topicArn := extractTopicArn(ctx.Message)
	if topicArn == "" {
		return fmt.Errorf("%s: missing topic ARN in event message (expected in detail.topicArn, detail.TopicArn, or resources[0])", t.Name())
	}

	if topicArn != metadata.TopicArn {
		ctx.Logger.Infof("Skipping event for topic %s, expected %s", topicArn, metadata.TopicArn)
		return nil
	}

	if err := ctx.Events.Emit("aws.sns.topic.message", ctx.Message); err != nil {
		return fmt.Errorf("%s: failed to emit topic message event: %w", t.Name(), err)
	}

	return nil
}

// HandleWebhook handles webhook requests for this trigger.
func (t *OnTopicMessage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

// Cleanup handles trigger cleanup.
func (t *OnTopicMessage) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// extractTopicArn extracts the topic ARN from message detail or resources.
func extractTopicArn(message any) string {
	payload, ok := message.(map[string]any)
	if !ok {
		return ""
	}

	detail, ok := payload["detail"].(map[string]any)
	if ok {
		candidates := []string{
			stringValue(detail["topicArn"]),
			stringValue(detail["TopicArn"]),
			stringValue(detail["topic-arn"]),
		}

		for _, candidate := range candidates {
			if candidate != "" {
				return candidate
			}
		}
	}

	resources, ok := payload["resources"].([]any)
	if !ok || len(resources) == 0 {
		return ""
	}

	return stringValue(resources[0])
}

// stringValue converts any value to a trimmed string when possible.
func stringValue(value any) string {
	if value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}
