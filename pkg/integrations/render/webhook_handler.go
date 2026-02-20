package render

import (
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type RenderWebhookHandler struct{}

type WebhookMetadata struct {
	WebhookID   string `json:"webhookId" mapstructure:"webhookId"`
	WorkspaceID string `json:"workspaceId" mapstructure:"workspaceId"`
}

type webhookSetupRequest struct {
	URL           string
	Configuration WebhookConfiguration
	Name          string
	EventFilter   []string
}

func (h *RenderWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA, err := decodeWebhookConfiguration(a)
	if err != nil {
		return false, fmt.Errorf("failed to decode webhook configuration A: %w", err)
	}

	configB, err := decodeWebhookConfiguration(b)
	if err != nil {
		return false, fmt.Errorf("failed to decode webhook configuration B: %w", err)
	}

	if configA.Strategy != configB.Strategy {
		return false, nil
	}

	if configA.Strategy == webhookStrategyResourceType {
		return configA.ResourceType == configB.ResourceType, nil
	}

	return true, nil
}

func (h *RenderWebhookHandler) Merge(current, requested any) (any, bool, error) {
	currentConfiguration, err := decodeWebhookConfiguration(current)
	if err != nil {
		return nil, false, fmt.Errorf("failed to decode current webhook configuration: %w", err)
	}

	requestedConfiguration, err := decodeWebhookConfiguration(requested)
	if err != nil {
		return nil, false, fmt.Errorf("failed to decode requested webhook configuration: %w", err)
	}

	if currentConfiguration.Strategy != requestedConfiguration.Strategy {
		return currentConfiguration, false, nil
	}

	if currentConfiguration.Strategy == webhookStrategyResourceType &&
		currentConfiguration.ResourceType != requestedConfiguration.ResourceType {
		return currentConfiguration, false, nil
	}

	mergedConfiguration := currentConfiguration
	mergedConfiguration.EventTypes = normalizeWebhookEventTypes(
		append(currentConfiguration.EventTypes, requestedConfiguration.EventTypes...),
	)

	if len(mergedConfiguration.EventTypes) == 0 {
		mergedConfiguration.EventTypes = defaultEventTypesForWebhook(currentConfiguration)
	}

	return mergedConfiguration, !webhookConfigurationsEqual(currentConfiguration, mergedConfiguration), nil
}

func (h *RenderWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	workspaceID, err := workspaceIDForIntegration(client, ctx.Integration)
	if err != nil {
		return nil, err
	}

	request, err := buildWebhookSetupRequest(ctx)
	if err != nil {
		return nil, err
	}

	selectedWebhook, err := h.findExistingWebhook(
		client,
		workspaceID,
		request.URL,
		request.Configuration,
		request.Name,
		request.EventFilter,
	)
	if err != nil {
		return nil, err
	}

	if selectedWebhook == nil {
		metadata, createErr := h.createWebhook(ctx, client, workspaceID, request)
		if createErr != nil {
			return nil, createErr
		}
		return metadata, nil
	}

	metadata, reuseErr := h.reuseWebhook(ctx, client, workspaceID, *selectedWebhook, request)
	if reuseErr != nil {
		return nil, reuseErr
	}

	return metadata, nil
}

func buildWebhookSetupRequest(ctx core.WebhookHandlerContext) (webhookSetupRequest, error) {
	webhookURL := ctx.Webhook.GetURL()
	if webhookURL == "" {
		return webhookSetupRequest{}, fmt.Errorf("webhook URL is required")
	}

	webhookConfiguration, err := decodeWebhookConfiguration(ctx.Webhook.GetConfiguration())
	if err != nil {
		return webhookSetupRequest{}, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	return webhookSetupRequest{
		URL:           webhookURL,
		Configuration: webhookConfiguration,
		Name:          webhookName(webhookConfiguration),
		EventFilter:   webhookEventFilter(webhookConfiguration),
	}, nil
}

func (h *RenderWebhookHandler) reuseWebhook(
	ctx core.WebhookHandlerContext,
	client *Client,
	workspaceID string,
	selectedWebhook Webhook,
	request webhookSetupRequest,
) (WebhookMetadata, error) {
	secret, err := h.updateWebhookIfNeeded(
		client,
		selectedWebhook,
		request.Name,
		request.URL,
		request.EventFilter,
	)
	if err != nil {
		return WebhookMetadata{}, err
	}

	return finalizeWebhookSetup(ctx, selectedWebhook.ID, workspaceID, secret)
}

func (h *RenderWebhookHandler) createWebhook(
	ctx core.WebhookHandlerContext,
	client *Client,
	workspaceID string,
	request webhookSetupRequest,
) (WebhookMetadata, error) {
	createdWebhook, err := client.CreateWebhook(CreateWebhookRequest{
		WorkspaceID: workspaceID,
		Name:        request.Name,
		URL:         request.URL,
		Enabled:     true,
		EventFilter: request.EventFilter,
	})
	if err != nil {
		return WebhookMetadata{}, fmt.Errorf("failed to create Render webhook: %w", err)
	}

	return finalizeWebhookSetup(ctx, createdWebhook.ID, workspaceID, createdWebhook.Secret)
}

func finalizeWebhookSetup(
	ctx core.WebhookHandlerContext,
	webhookID string,
	workspaceID string,
	secret string,
) (WebhookMetadata, error) {
	if err := setWebhookSecret(ctx, secret); err != nil {
		return WebhookMetadata{}, err
	}

	return WebhookMetadata{WebhookID: webhookID, WorkspaceID: workspaceID}, nil
}

func (h *RenderWebhookHandler) findExistingWebhook(
	client *Client,
	workspaceID string,
	webhookURL string,
	webhookConfiguration WebhookConfiguration,
	selectedWebhookName string,
	eventFilter []string,
) (*Webhook, error) {
	webhooks, err := client.ListWebhooks(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list Render webhooks: %w", err)
	}

	candidateWebhooks := filterWebhooksByURL(webhooks, webhookURL)

	if webhookConfiguration.Strategy == webhookStrategyIntegration {
		return pickExistingRenderWebhook(candidateWebhooks, selectedWebhookName), nil
	}

	webhook := pickExistingRenderWebhookByName(candidateWebhooks, selectedWebhookName)
	if webhook != nil {
		return webhook, nil
	}

	return pickExistingRenderWebhookByEventFilter(candidateWebhooks, eventFilter), nil
}

func (h *RenderWebhookHandler) updateWebhookIfNeeded(
	client *Client,
	selectedWebhook Webhook,
	selectedWebhookName string,
	webhookURL string,
	eventFilter []string,
) (string, error) {
	retrievedWebhook, err := client.GetWebhook(selectedWebhook.ID)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve existing Render webhook: %w", err)
	}
	if retrievedWebhook == nil {
		return "", fmt.Errorf("failed to retrieve existing Render webhook: empty response")
	}

	existingEventFilter := existingWebhookEventFilter(*retrievedWebhook, selectedWebhook)
	existingName := existingWebhookName(*retrievedWebhook, selectedWebhook)
	mergedEventFilter := mergeWebhookEventFilters(existingEventFilter, eventFilter)
	if shouldUpdateWebhook(retrievedWebhook.Enabled, existingName, selectedWebhookName, existingEventFilter, mergedEventFilter) {
		_, err = client.UpdateWebhook(selectedWebhook.ID, UpdateWebhookRequest{
			Name:        selectedWebhookName,
			URL:         webhookURL,
			Enabled:     true,
			EventFilter: mergedEventFilter,
		})
		if err != nil {
			return "", fmt.Errorf("failed to update existing Render webhook: %w", err)
		}
	}

	return webhookSecret(*retrievedWebhook, selectedWebhook), nil
}

func setWebhookSecret(ctx core.WebhookHandlerContext, secret string) error {
	if secret == "" {
		return fmt.Errorf("render webhook secret is empty")
	}
	if err := ctx.Webhook.SetSecret([]byte(secret)); err != nil {
		return fmt.Errorf("failed to store webhook secret: %w", err)
	}
	return nil
}

func decodeWebhookMetadata(value any) (WebhookMetadata, error) {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(value, &metadata); err != nil {
		return WebhookMetadata{}, err
	}

	return metadata, nil
}

func (h *RenderWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata, err := decodeWebhookMetadata(ctx.Webhook.GetMetadata())
	if err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}

	if metadata.WebhookID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	err = client.DeleteWebhook(metadata.WebhookID)
	if err == nil {
		return nil
	}

	apiErr, ok := err.(*APIError)
	if ok && apiErr.StatusCode == 404 {
		return nil
	}

	return err
}

func pickExistingRenderWebhook(webhooks []Webhook, webhookName string) *Webhook {
	if len(webhooks) == 0 {
		return nil
	}

	nameMatch := pickExistingRenderWebhookByName(webhooks, webhookName)
	if nameMatch != nil {
		return nameMatch
	}

	return &webhooks[0]
}

func pickExistingRenderWebhookByName(webhooks []Webhook, webhookName string) *Webhook {
	if webhookName == "" {
		return nil
	}

	for i := range webhooks {
		if webhooks[i].Name == webhookName {
			return &webhooks[i]
		}
	}

	return nil
}

func pickExistingRenderWebhookByEventFilter(webhooks []Webhook, eventFilter []string) *Webhook {
	for i := range webhooks {
		if slices.Equal(normalizeWebhookEventTypes(webhooks[i].EventFilter), eventFilter) {
			return &webhooks[i]
		}
	}

	return nil
}

func filterWebhooksByURL(webhooks []Webhook, webhookURL string) []Webhook {
	filteredWebhooks := make([]Webhook, 0, len(webhooks))
	for _, webhook := range webhooks {
		if webhook.URL == webhookURL {
			filteredWebhooks = append(filteredWebhooks, webhook)
		}
	}

	return filteredWebhooks
}

func existingWebhookEventFilter(retrievedWebhook, selectedWebhook Webhook) []string {
	existingEventFilter := normalizeWebhookEventTypes(retrievedWebhook.EventFilter)
	if len(existingEventFilter) != 0 {
		return existingEventFilter
	}

	return normalizeWebhookEventTypes(selectedWebhook.EventFilter)
}

func existingWebhookName(retrievedWebhook, selectedWebhook Webhook) string {
	existingName := retrievedWebhook.Name
	if existingName != "" {
		return existingName
	}

	return selectedWebhook.Name
}

func mergeWebhookEventFilters(existingEventFilter, requestEventFilter []string) []string {
	mergedEventFilter := normalizeWebhookEventTypes(append(existingEventFilter, requestEventFilter...))
	if len(mergedEventFilter) != 0 {
		return mergedEventFilter
	}

	return requestEventFilter
}

func shouldUpdateWebhook(
	enabled bool,
	existingName string,
	selectedWebhookName string,
	existingEventFilter []string,
	mergedEventFilter []string,
) bool {
	return existingName != selectedWebhookName || !slices.Equal(existingEventFilter, mergedEventFilter) || !enabled
}

func webhookSecret(retrievedWebhook, selectedWebhook Webhook) string {
	if retrievedWebhook.Secret != "" {
		return retrievedWebhook.Secret
	}

	return selectedWebhook.Secret
}
