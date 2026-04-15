package azure

import (
	"context"

	"github.com/sirupsen/logrus"
)

// ServiceBusQueueResult is the output returned by queue create/get operations.
type ServiceBusQueueResult struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	NamespaceName string         `json:"namespaceName"`
	ResourceGroup string         `json:"resourceGroup"`
	Properties    map[string]any `json:"properties,omitempty"`
}

// ServiceBusTopicResult is the output returned by topic create/get operations.
type ServiceBusTopicResult struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	NamespaceName string         `json:"namespaceName"`
	ResourceGroup string         `json:"resourceGroup"`
	Properties    map[string]any `json:"properties,omitempty"`
}

// CreateServiceBusQueue creates a Service Bus queue and returns its details.
func CreateServiceBusQueue(ctx context.Context, provider *AzureProvider, resourceGroup, namespaceName, queueName string, logger *logrus.Entry) (*ServiceBusQueueResult, error) {
	logger.Infof("Creating Service Bus queue: %s in namespace %s", queueName, namespaceName)

	queue, err := provider.getClient().createOrUpdateServiceBusQueue(ctx, provider.GetSubscriptionID(), resourceGroup, namespaceName, queueName)
	if err != nil {
		return nil, err
	}

	return queueToResult(queue, namespaceName, resourceGroup), nil
}

// GetServiceBusQueue retrieves a Service Bus queue and returns its details.
func GetServiceBusQueue(ctx context.Context, provider *AzureProvider, resourceGroup, namespaceName, queueName string, logger *logrus.Entry) (*ServiceBusQueueResult, error) {
	logger.Infof("Getting Service Bus queue: %s in namespace %s", queueName, namespaceName)

	queue, err := provider.getClient().getServiceBusQueue(ctx, provider.GetSubscriptionID(), resourceGroup, namespaceName, queueName)
	if err != nil {
		return nil, err
	}

	return queueToResult(queue, namespaceName, resourceGroup), nil
}

// DeleteServiceBusQueue deletes a Service Bus queue.
func DeleteServiceBusQueue(ctx context.Context, provider *AzureProvider, resourceGroup, namespaceName, queueName string, logger *logrus.Entry) (map[string]any, error) {
	logger.Infof("Deleting Service Bus queue: %s in namespace %s", queueName, namespaceName)

	if err := provider.getClient().deleteServiceBusQueue(ctx, provider.GetSubscriptionID(), resourceGroup, namespaceName, queueName); err != nil {
		return nil, err
	}

	return map[string]any{
		"name":          queueName,
		"namespaceName": namespaceName,
		"resourceGroup": resourceGroup,
		"deleted":       true,
	}, nil
}

// SendServiceBusMessage sends a message to a Service Bus queue.
func SendServiceBusMessage(ctx context.Context, provider *AzureProvider, namespaceName, queueName, body, contentType string, logger *logrus.Entry) (map[string]any, error) {
	logger.Infof("Sending message to Service Bus queue: %s in namespace %s", queueName, namespaceName)

	sbClient := newServiceBusDataClient(provider.getClient(), namespaceName)
	if err := sbClient.sendMessage(ctx, queueName, body, contentType); err != nil {
		return nil, err
	}

	return map[string]any{
		"queue":         queueName,
		"namespaceName": namespaceName,
		"sent":          true,
	}, nil
}

// CreateServiceBusTopic creates a Service Bus topic and returns its details.
func CreateServiceBusTopic(ctx context.Context, provider *AzureProvider, resourceGroup, namespaceName, topicName string, logger *logrus.Entry) (*ServiceBusTopicResult, error) {
	logger.Infof("Creating Service Bus topic: %s in namespace %s", topicName, namespaceName)

	topic, err := provider.getClient().createOrUpdateServiceBusTopic(ctx, provider.GetSubscriptionID(), resourceGroup, namespaceName, topicName)
	if err != nil {
		return nil, err
	}

	return topicToResult(topic, namespaceName, resourceGroup), nil
}

// GetServiceBusTopic retrieves a Service Bus topic and returns its details.
func GetServiceBusTopic(ctx context.Context, provider *AzureProvider, resourceGroup, namespaceName, topicName string, logger *logrus.Entry) (*ServiceBusTopicResult, error) {
	logger.Infof("Getting Service Bus topic: %s in namespace %s", topicName, namespaceName)

	topic, err := provider.getClient().getServiceBusTopic(ctx, provider.GetSubscriptionID(), resourceGroup, namespaceName, topicName)
	if err != nil {
		return nil, err
	}

	return topicToResult(topic, namespaceName, resourceGroup), nil
}

// DeleteServiceBusTopic deletes a Service Bus topic.
func DeleteServiceBusTopic(ctx context.Context, provider *AzureProvider, resourceGroup, namespaceName, topicName string, logger *logrus.Entry) (map[string]any, error) {
	logger.Infof("Deleting Service Bus topic: %s in namespace %s", topicName, namespaceName)

	if err := provider.getClient().deleteServiceBusTopic(ctx, provider.GetSubscriptionID(), resourceGroup, namespaceName, topicName); err != nil {
		return nil, err
	}

	return map[string]any{
		"name":          topicName,
		"namespaceName": namespaceName,
		"resourceGroup": resourceGroup,
		"deleted":       true,
	}, nil
}

// PublishServiceBusMessage publishes a message to a Service Bus topic.
func PublishServiceBusMessage(ctx context.Context, provider *AzureProvider, namespaceName, topicName, body, contentType string, logger *logrus.Entry) (map[string]any, error) {
	logger.Infof("Publishing message to Service Bus topic: %s in namespace %s", topicName, namespaceName)

	sbClient := newServiceBusDataClient(provider.getClient(), namespaceName)
	if err := sbClient.sendMessage(ctx, topicName, body, contentType); err != nil {
		return nil, err
	}

	return map[string]any{
		"topic":         topicName,
		"namespaceName": namespaceName,
		"published":     true,
	}, nil
}

// queueToResult converts an ARM queue response to the output shape.
func queueToResult(queue *armServiceBusQueue, namespaceName, resourceGroup string) *ServiceBusQueueResult {
	props := map[string]any{
		"messageCount":                     queue.Properties.MessageCount,
		"sizeInBytes":                      queue.Properties.SizeInBytes,
		"maxSizeInMegabytes":               queue.Properties.MaxSizeInMegabytes,
		"lockDuration":                     queue.Properties.LockDuration,
		"maxDeliveryCount":                 queue.Properties.MaxDeliveryCount,
		"requiresDuplicateDetection":       queue.Properties.RequiresDuplicateDetection,
		"requiresSession":                  queue.Properties.RequiresSession,
		"defaultMessageTimeToLive":         queue.Properties.DefaultMessageTimeToLive,
		"deadLetteringOnMessageExpiration": queue.Properties.DeadLetteringOnMessageExpiration,
		"enableBatchedOperations":          queue.Properties.EnableBatchedOperations,
		"status":                           queue.Properties.Status,
		"createdAt":                        queue.Properties.CreatedAt,
		"updatedAt":                        queue.Properties.UpdatedAt,
		"activeMessageCount":               queue.Properties.CountDetails.ActiveMessageCount,
		"deadLetterMessageCount":           queue.Properties.CountDetails.DeadLetterMessageCount,
	}

	return &ServiceBusQueueResult{
		ID:            queue.ID,
		Name:          queue.Name,
		NamespaceName: namespaceName,
		ResourceGroup: resourceGroup,
		Properties:    props,
	}
}

// topicToResult converts an ARM topic response to the output shape.
func topicToResult(topic *armServiceBusTopic, namespaceName, resourceGroup string) *ServiceBusTopicResult {
	props := map[string]any{
		"sizeInBytes":                topic.Properties.SizeInBytes,
		"maxSizeInMegabytes":         topic.Properties.MaxSizeInMegabytes,
		"defaultMessageTimeToLive":   topic.Properties.DefaultMessageTimeToLive,
		"requiresDuplicateDetection": topic.Properties.RequiresDuplicateDetection,
		"enableBatchedOperations":    topic.Properties.EnableBatchedOperations,
		"enablePartitioning":         topic.Properties.EnablePartitioning,
		"supportOrdering":            topic.Properties.SupportOrdering,
		"status":                     topic.Properties.Status,
		"createdAt":                  topic.Properties.CreatedAt,
		"updatedAt":                  topic.Properties.UpdatedAt,
		"subscriptionCount":          topic.Properties.SubscriptionCount,
		"activeMessageCount":         topic.Properties.CountDetails.ActiveMessageCount,
		"deadLetterMessageCount":     topic.Properties.CountDetails.DeadLetterMessageCount,
	}

	return &ServiceBusTopicResult{
		ID:            topic.ID,
		Name:          topic.Name,
		NamespaceName: namespaceName,
		ResourceGroup: resourceGroup,
		Properties:    props,
	}
}
