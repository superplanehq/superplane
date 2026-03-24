package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	armAPIVersionServiceBus = "2024-01-01"
)

// serviceBusDataClient wraps HTTP calls to the Azure Service Bus data plane REST API.
// It authenticates using a Service Bus-scoped OAuth2 token (servicebus.azure.net).
type serviceBusDataClient struct {
	armClient     *armClient
	namespaceFQDN string // e.g. "myns.servicebus.windows.net"
	httpClient    *http.Client
}

func newServiceBusDataClient(armClient *armClient, namespaceName string) *serviceBusDataClient {
	return &serviceBusDataClient{
		armClient:     armClient,
		namespaceFQDN: fmt.Sprintf("%s.servicebus.windows.net", namespaceName),
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
}

// sendMessage posts a message to a Service Bus queue or topic.
// entityPath is the queue name or topic name.
func (c *serviceBusDataClient) sendMessage(ctx context.Context, entityPath, body, contentType string) error {
	token, err := c.armClient.serviceBusToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Service Bus token: %w", err)
	}

	if token == "" {
		return fmt.Errorf("Service Bus token is empty: integration needs to sync")
	}

	url := fmt.Sprintf("https://%s/%s/messages", c.namespaceFQDN, entityPath)
	c.armClient.logger.Debugf("Service Bus POST %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	} else {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Service Bus send message failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// serviceBusARMURL builds the ARM resource URL for a Service Bus resource path.
func (c *armClient) serviceBusARMURL(subscriptionID, resourceGroup, namespaceName, subPath string) string {
	base := fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ServiceBus/namespaces/%s",
		c.getBaseURL(), subscriptionID, resourceGroup, namespaceName)
	if subPath != "" {
		return base + "/" + subPath + "?api-version=" + armAPIVersionServiceBus
	}
	return base + "?api-version=" + armAPIVersionServiceBus
}

// serviceBusARMListURL builds a paginated list ARM URL for a Service Bus sub-resource.
func (c *armClient) serviceBusARMListURL(subscriptionID, resourceGroup, namespaceName, subResource string) string {
	return fmt.Sprintf("%s/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ServiceBus/namespaces/%s/%s?api-version=%s",
		c.getBaseURL(), subscriptionID, resourceGroup, namespaceName, subResource, armAPIVersionServiceBus)
}

// createOrUpdateServiceBusQueue creates or updates a Service Bus queue via ARM.
func (c *armClient) createOrUpdateServiceBusQueue(ctx context.Context, subscriptionID, resourceGroup, namespaceName, queueName string) (*armServiceBusQueue, error) {
	url := c.serviceBusARMURL(subscriptionID, resourceGroup, namespaceName, "queues/"+queueName)
	body := map[string]any{
		"properties": map[string]any{},
	}

	rawResult, err := c.putAndPoll(ctx, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create queue: %w", err)
	}

	var queue armServiceBusQueue
	if err := json.Unmarshal(rawResult, &queue); err != nil {
		return nil, fmt.Errorf("failed to parse queue response: %w", err)
	}

	return &queue, nil
}

// getServiceBusQueue retrieves a Service Bus queue via ARM.
func (c *armClient) getServiceBusQueue(ctx context.Context, subscriptionID, resourceGroup, namespaceName, queueName string) (*armServiceBusQueue, error) {
	url := c.serviceBusARMURL(subscriptionID, resourceGroup, namespaceName, "queues/"+queueName)

	var queue armServiceBusQueue
	if err := c.get(ctx, url, &queue); err != nil {
		return nil, fmt.Errorf("failed to get queue: %w", err)
	}

	return &queue, nil
}

// deleteServiceBusQueue deletes a Service Bus queue via ARM.
func (c *armClient) deleteServiceBusQueue(ctx context.Context, subscriptionID, resourceGroup, namespaceName, queueName string) error {
	url := c.serviceBusARMURL(subscriptionID, resourceGroup, namespaceName, "queues/"+queueName)
	if err := c.deleteAndPoll(ctx, url); err != nil {
		if isARMNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete queue: %w", err)
	}
	return nil
}

// createOrUpdateServiceBusTopic creates or updates a Service Bus topic via ARM.
func (c *armClient) createOrUpdateServiceBusTopic(ctx context.Context, subscriptionID, resourceGroup, namespaceName, topicName string) (*armServiceBusTopic, error) {
	url := c.serviceBusARMURL(subscriptionID, resourceGroup, namespaceName, "topics/"+topicName)
	body := map[string]any{
		"properties": map[string]any{},
	}

	rawResult, err := c.putAndPoll(ctx, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create topic: %w", err)
	}

	var topic armServiceBusTopic
	if err := json.Unmarshal(rawResult, &topic); err != nil {
		return nil, fmt.Errorf("failed to parse topic response: %w", err)
	}

	return &topic, nil
}

// getServiceBusTopic retrieves a Service Bus topic via ARM.
func (c *armClient) getServiceBusTopic(ctx context.Context, subscriptionID, resourceGroup, namespaceName, topicName string) (*armServiceBusTopic, error) {
	url := c.serviceBusARMURL(subscriptionID, resourceGroup, namespaceName, "topics/"+topicName)

	var topic armServiceBusTopic
	if err := c.get(ctx, url, &topic); err != nil {
		return nil, fmt.Errorf("failed to get topic: %w", err)
	}

	return &topic, nil
}

// deleteServiceBusTopic deletes a Service Bus topic via ARM.
func (c *armClient) deleteServiceBusTopic(ctx context.Context, subscriptionID, resourceGroup, namespaceName, topicName string) error {
	url := c.serviceBusARMURL(subscriptionID, resourceGroup, namespaceName, "topics/"+topicName)
	if err := c.deleteAndPoll(ctx, url); err != nil {
		if isARMNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete topic: %w", err)
	}
	return nil
}
