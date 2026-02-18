package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	webhookMetadataTopicID        = "topicId"
	webhookMetadataSubscriptionID = "subscriptionId"
	webhookMetadataSinkID         = "sinkId"
	resourceIDPrefix              = "sp-vm-"
)

type webhookConfig struct {
	ProjectID string `mapstructure:"projectId"`
	Region    string `mapstructure:"region"`
}

type WebhookHandler struct{}

func (h *WebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	var config webhookConfig
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("decode webhook configuration: %w", err)
	}
	projectID := strings.TrimSpace(config.ProjectID)
	if projectID == "" {
		return nil, fmt.Errorf("project ID is required for On VM Created webhook")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("create GCP client: %w", err)
	}

	webhookID := ctx.Webhook.GetID()
	topicID := resourceIDPrefix + strings.ReplaceAll(webhookID, "-", "")
	subscriptionID := topicID
	sinkID := topicID

	reqCtx := context.Background()

	if err := CreateTopic(reqCtx, client, projectID, topicID); err != nil {
		return nil, fmt.Errorf("create Pub/Sub topic: %w", err)
	}
	time.Sleep(2 * time.Second)

	topicFullName := fmt.Sprintf("projects/%s/topics/%s", projectID, topicID)
	writerIdentity, err := CreateVMCreatedSink(reqCtx, client, projectID, sinkID, topicFullName)
	if err != nil {
		_ = DeleteTopic(reqCtx, client, projectID, topicID)
		return nil, fmt.Errorf("create Logging sink: %w", err)
	}

	if err := GrantTopicPublish(reqCtx, client, projectID, topicID, writerIdentity); err != nil {
		_ = DeleteSink(reqCtx, client, projectID, sinkID)
		_ = DeleteTopic(reqCtx, client, projectID, topicID)
		return nil, fmt.Errorf("grant sink permission to publish to topic: %w", err)
	}

	pushEndpoint := ctx.Webhook.GetURL()
	if err := CreatePushSubscription(reqCtx, client, projectID, topicID, subscriptionID, pushEndpoint); err != nil {
		_ = DeleteSink(reqCtx, client, projectID, sinkID)
		_ = DeleteTopic(reqCtx, client, projectID, topicID)
		return nil, fmt.Errorf("create push subscription: %w", err)
	}

	return map[string]string{
		webhookMetadataTopicID:        topicID,
		webhookMetadataSubscriptionID: subscriptionID,
		webhookMetadataSinkID:         sinkID,
	}, nil
}

func (h *WebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	var config webhookConfig
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return fmt.Errorf("decode webhook configuration: %w", err)
	}
	projectID := strings.TrimSpace(config.ProjectID)
	if projectID == "" {
		return nil
	}

	meta := ctx.Webhook.GetMetadata()
	metaMap, _ := meta.(map[string]any)
	if metaMap == nil {
		return nil
	}
	subID := getMetaString(metaMap, webhookMetadataSubscriptionID)
	sinkID := getMetaString(metaMap, webhookMetadataSinkID)
	topicID := getMetaString(metaMap, webhookMetadataTopicID)

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("create GCP client for cleanup: %w", err)
	}
	reqCtx := context.Background()

	if subID != "" {
		_ = DeleteSubscription(reqCtx, client, projectID, subID)
	}
	if sinkID != "" {
		_ = DeleteSink(reqCtx, client, projectID, sinkID)
	}
	if topicID != "" {
		_ = DeleteTopic(reqCtx, client, projectID, topicID)
	}
	return nil
}

func (h *WebhookHandler) CompareConfig(a, b any) (bool, error) {
	var ca, cb webhookConfig
	if err := mapstructure.Decode(a, &ca); err != nil {
		return false, err
	}
	if err := mapstructure.Decode(b, &cb); err != nil {
		return false, err
	}
	return strings.TrimSpace(ca.ProjectID) == strings.TrimSpace(cb.ProjectID) &&
		strings.TrimSpace(ca.Region) == strings.TrimSpace(cb.Region), nil
}

func (h *WebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func getMetaString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}
