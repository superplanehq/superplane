package render

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("render", &Render{})
}

type Render struct{}

type Configuration struct {
	APIKey        string `json:"apiKey" mapstructure:"apiKey"`
	Workspace     string `json:"workspace" mapstructure:"workspace"`
	WorkspacePlan string `json:"workspacePlan" mapstructure:"workspacePlan"`
}

type Metadata struct {
	Workspace *WorkspaceMetadata `json:"workspace,omitempty" mapstructure:"workspace"`
}

type WorkspaceMetadata struct {
	ID   string `json:"id" mapstructure:"id"`
	Plan string `json:"plan" mapstructure:"plan"`
}

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

func (r *Render) Name() string {
	return "render"
}

func (r *Render) Label() string {
	return "Render"
}

func (r *Render) Icon() string {
	return "server"
}

func (r *Render) Description() string {
	return "Deploy and manage Render services, and react to Render deploy/build events"
}

func (r *Render) Instructions() string {
	return `
1. **API Key:** Create it in [Render Account Settings -> API Keys](https://dashboard.render.com/u/settings#api-keys).
2. **Workspace (optional):** Use your Render workspace ID (` + "`usr-...`" + ` or ` + "`tea-...`" + `) or workspace name. Leave empty to use the first workspace available to the API key.
3. **Workspace Plan:** Select **Professional** or **Organization / Enterprise** (used to choose webhook strategy).
4. **Auth:** SuperPlane sends requests to [Render API v1](https://api.render.com/v1/) using ` + "`Authorization: Bearer <API_KEY>`" + `.
5. **Webhooks:** SuperPlane configures Render webhooks automatically via the [Render Webhooks API](https://render.com/docs/webhooks). No manual setup is required.
6. **Troubleshooting:** Check [Render Dashboard -> Integrations -> Webhooks](https://dashboard.render.com/) and the [Render webhook docs](https://render.com/docs/webhooks).

Note: **Plan requirement:** Render webhooks require a Professional plan or higher.`
}

func (r *Render) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Render API key",
		},
		{
			Name:        "workspace",
			Label:       "Workspace",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional Render workspace ID/name. Use this if your API key has access to multiple workspaces.",
		},
		{
			Name:     "workspacePlan",
			Label:    "Workspace Plan",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  workspacePlanProfessional,
			Description: "Render workspace plan used for webhook strategy. " +
				"Use Organization / Enterprise when available.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Professional", Value: workspacePlanProfessional},
						{Label: "Organization / Enterprise", Value: workspacePlanOrganization},
					},
				},
			},
		},
	}
}

func (r *Render) Components() []core.Component {
	return []core.Component{
		&TriggerDeploy{},
	}
}

func (r *Render) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnDeploy{},
		&OnBuild{},
	}
}

func (r *Render) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (r *Render) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return fmt.Errorf("failed to verify Render credentials: %w", err)
	}

	workspace, err := resolveWorkspace(client, config.Workspace)
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	ctx.Integration.SetMetadata(buildMetadata(workspace.ID, config.WorkspacePlan))
	ctx.Integration.Ready()
	return nil
}

func (r *Render) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (r *Render) CompareWebhookConfig(a, b any) (bool, error) {
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

func (r *Render) MergeWebhookConfig(current, requested any) (any, bool, error) {
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

func (r *Render) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "service" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	workspaceID, err := workspaceIDForIntegration(client, ctx.Integration)
	if err != nil {
		return nil, err
	}

	services, err := client.ListServices(workspaceID)
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(services))
	for _, service := range services {
		if service.ID == "" || service.Name == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{Type: resourceType, Name: service.Name, ID: service.ID})
	}

	return resources, nil
}

func (r *Render) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
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

	selectedWebhook, err := r.findExistingWebhook(
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
		metadata, createErr := r.createWebhook(ctx, client, workspaceID, request)
		if createErr != nil {
			return nil, createErr
		}
		return metadata, nil
	}

	metadata, reuseErr := r.reuseWebhook(ctx, client, workspaceID, *selectedWebhook, request)
	if reuseErr != nil {
		return nil, reuseErr
	}

	return metadata, nil
}

func buildWebhookSetupRequest(ctx core.SetupWebhookContext) (webhookSetupRequest, error) {
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

func (r *Render) reuseWebhook(
	ctx core.SetupWebhookContext,
	client *Client,
	workspaceID string,
	selectedWebhook Webhook,
	request webhookSetupRequest,
) (WebhookMetadata, error) {
	secret, err := r.updateWebhookIfNeeded(
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

func (r *Render) createWebhook(
	ctx core.SetupWebhookContext,
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
	ctx core.SetupWebhookContext,
	webhookID string,
	workspaceID string,
	secret string,
) (WebhookMetadata, error) {
	if err := setWebhookSecret(ctx, secret); err != nil {
		return WebhookMetadata{}, err
	}

	return WebhookMetadata{WebhookID: webhookID, WorkspaceID: workspaceID}, nil
}

func (r *Render) findExistingWebhook(
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

func (r *Render) updateWebhookIfNeeded(
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

func setWebhookSecret(ctx core.SetupWebhookContext, secret string) error {
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

	if metadata.WorkspaceID != "" {
		return metadata, nil
	}

	legacyMetadata := map[string]any{}
	if err := mapstructure.Decode(value, &legacyMetadata); err != nil {
		return metadata, nil
	}

	legacyWorkspaceID, ok := legacyMetadata["ownerId"].(string)
	if !ok {
		return metadata, nil
	}

	metadata.WorkspaceID = strings.TrimSpace(legacyWorkspaceID)
	return metadata, nil
}

func (r *Render) CleanupWebhook(ctx core.CleanupWebhookContext) error {
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

func (r *Render) Actions() []core.Action {
	return []core.Action{}
}

func (r *Render) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func workspaceIDForIntegration(client *Client, integration core.IntegrationContext) (string, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err == nil && metadata.Workspace != nil && metadata.Workspace.ID != "" {
		return metadata.Workspace.ID, nil
	}

	workspace := ""
	workspaceValue, workspaceErr := integration.GetConfig("workspace")
	if workspaceErr == nil {
		workspace = string(workspaceValue)
	}

	selectedWorkspace, err := resolveWorkspace(client, workspace)
	if err != nil {
		return "", err
	}

	workspacePlan := workspacePlanProfessional
	workspacePlanValue, workspacePlanErr := integration.GetConfig("workspacePlan")
	if workspacePlanErr == nil {
		workspacePlan = string(workspacePlanValue)
	}

	integration.SetMetadata(buildMetadata(selectedWorkspace.ID, workspacePlan))
	return selectedWorkspace.ID, nil
}

func resolveWorkspace(client *Client, workspace string) (Workspace, error) {
	workspaces, err := client.ListWorkspaces()
	if err != nil {
		return Workspace{}, err
	}

	if len(workspaces) == 0 {
		return Workspace{}, fmt.Errorf("no workspaces found for this API key")
	}

	if workspace == "" {
		return workspaces[0], nil
	}

	selectedWorkspace := slices.IndexFunc(workspaces, func(item Workspace) bool {
		return item.ID == workspace
	})
	if selectedWorkspace < 0 {
		selectedWorkspace = slices.IndexFunc(workspaces, func(item Workspace) bool {
			return strings.EqualFold(item.Name, workspace)
		})
	}

	if selectedWorkspace < 0 {
		return Workspace{}, fmt.Errorf("workspace %s is not accessible with this API key", workspace)
	}

	return workspaces[selectedWorkspace], nil
}

func (m Metadata) workspacePlan() string {
	if m.Workspace == nil {
		return workspacePlanProfessional
	}

	return m.Workspace.Plan
}

func buildMetadata(workspaceID, workspacePlan string) Metadata {
	return Metadata{
		Workspace: &WorkspaceMetadata{
			ID:   strings.TrimSpace(workspaceID),
			Plan: strings.TrimSpace(workspacePlan),
		},
	}
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
