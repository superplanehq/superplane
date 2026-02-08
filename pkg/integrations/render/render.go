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
	WebhookID string `json:"webhookId" mapstructure:"webhookId"`
	OwnerID   string `json:"ownerId" mapstructure:"ownerId"`
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

	if strings.TrimSpace(config.APIKey) == "" {
		return fmt.Errorf("apiKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := client.Verify(); err != nil {
		return fmt.Errorf("failed to verify Render credentials: %w", err)
	}

	owner, err := resolveOwner(client, config.workspace())
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	workspacePlan := normalizeWorkspacePlan(config.WorkspacePlan)
	ctx.Integration.SetMetadata(buildMetadata(owner.ID, workspacePlan))
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

	ownerID, err := ownerIDForIntegration(client, ctx.Integration)
	if err != nil {
		return nil, err
	}

	services, err := client.ListServices(ownerID)
	if err != nil {
		return nil, err
	}

	resources := make([]core.IntegrationResource, 0, len(services))
	for _, service := range services {
		if strings.TrimSpace(service.ID) == "" {
			continue
		}

		name := strings.TrimSpace(service.Name)
		if name == "" {
			name = service.ID
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: name,
			ID:   service.ID,
		})
	}

	return resources, nil
}

func (r *Render) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	ownerID, err := ownerIDForIntegration(client, ctx.Integration)
	if err != nil {
		return nil, err
	}

	webhookURL := ctx.Webhook.GetURL()
	if webhookURL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	webhookConfiguration, err := decodeWebhookConfiguration(ctx.Webhook.GetConfiguration())
	if err != nil {
		return nil, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	selectedWebhookName := webhookName(webhookConfiguration)
	eventFilter := webhookEventFilter(webhookConfiguration)
	selectedWebhook, err := r.findExistingWebhook(
		client,
		ownerID,
		webhookURL,
		webhookConfiguration,
		selectedWebhookName,
		eventFilter,
	)
	if err != nil {
		return nil, err
	}

	if selectedWebhook != nil {
		secret, err := r.updateWebhookIfNeeded(
			client,
			*selectedWebhook,
			selectedWebhookName,
			webhookURL,
			eventFilter,
		)
		if err != nil {
			return nil, err
		}

		if err := setWebhookSecret(ctx, secret); err != nil {
			return nil, err
		}

		return WebhookMetadata{WebhookID: selectedWebhook.ID, OwnerID: ownerID}, nil
	}

	createdWebhook, err := client.CreateWebhook(CreateWebhookRequest{
		OwnerID:     ownerID,
		Name:        selectedWebhookName,
		URL:         webhookURL,
		Enabled:     true,
		EventFilter: eventFilter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Render webhook: %w", err)
	}
	if err := setWebhookSecret(ctx, createdWebhook.Secret); err != nil {
		return nil, err
	}
	return WebhookMetadata{WebhookID: createdWebhook.ID, OwnerID: ownerID}, nil
}

func (r *Render) findExistingWebhook(
	client *Client,
	ownerID string,
	webhookURL string,
	webhookConfiguration WebhookConfiguration,
	selectedWebhookName string,
	eventFilter []string,
) (*Webhook, error) {
	webhooks, err := client.ListWebhooks(ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list Render webhooks: %w", err)
	}

	candidateWebhooks := make([]Webhook, 0, len(webhooks))
	for _, webhook := range webhooks {
		if webhook.URL == webhookURL {
			candidateWebhooks = append(candidateWebhooks, webhook)
		}
	}

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

	existingEventFilter := normalizeWebhookEventTypes(retrievedWebhook.EventFilter)
	if len(existingEventFilter) == 0 {
		existingEventFilter = normalizeWebhookEventTypes(selectedWebhook.EventFilter)
	}

	existingName := retrievedWebhook.Name
	if existingName == "" {
		existingName = selectedWebhook.Name
	}

	mergedEventFilter := normalizeWebhookEventTypes(append(existingEventFilter, eventFilter...))
	if len(mergedEventFilter) == 0 {
		mergedEventFilter = eventFilter
	}

	if existingName != selectedWebhookName || !slices.Equal(existingEventFilter, mergedEventFilter) || !retrievedWebhook.Enabled {
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

	if retrievedWebhook.Secret != "" {
		return retrievedWebhook.Secret, nil
	}

	return selectedWebhook.Secret, nil
}

func setWebhookSecret(ctx core.SetupWebhookContext, secret string) error {
	if strings.TrimSpace(secret) == "" {
		return fmt.Errorf("render webhook secret is empty")
	}
	if err := ctx.Webhook.SetSecret([]byte(secret)); err != nil {
		return fmt.Errorf("failed to store webhook secret: %w", err)
	}
	return nil
}

func (r *Render) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
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

func ownerIDForIntegration(client *Client, integration core.IntegrationContext) (string, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err == nil && metadata.Workspace != nil && metadata.Workspace.ID != "" {
		return metadata.Workspace.ID, nil
	}

	workspace := ""
	workspaceValue, workspaceErr := integration.GetConfig("workspace")
	if workspaceErr == nil {
		workspace = strings.TrimSpace(string(workspaceValue))
	}

	owner, err := resolveOwner(client, workspace)
	if err != nil {
		return "", err
	}

	integration.SetMetadata(buildMetadata(owner.ID, workspacePlanFromConfig(integration)))
	return owner.ID, nil
}

func resolveOwner(client *Client, ownerID string) (Owner, error) {
	owners, err := client.ListOwners()
	if err != nil {
		return Owner{}, err
	}

	if len(owners) == 0 {
		return Owner{}, fmt.Errorf("no workspaces found for this API key")
	}

	trimmedWorkspace := strings.TrimSpace(ownerID)
	if trimmedWorkspace == "" {
		return owners[0], nil
	}

	selectedOwner := slices.IndexFunc(owners, func(owner Owner) bool {
		return strings.TrimSpace(owner.ID) == trimmedWorkspace
	})
	if selectedOwner < 0 {
		selectedOwner = slices.IndexFunc(owners, func(owner Owner) bool {
			return strings.EqualFold(strings.TrimSpace(owner.Name), trimmedWorkspace)
		})
	}

	if selectedOwner < 0 {
		return Owner{}, fmt.Errorf("workspace %s is not accessible with this API key", trimmedWorkspace)
	}

	return owners[selectedOwner], nil
}

func (c Configuration) workspace() string {
	return strings.TrimSpace(c.Workspace)
}

func (m Metadata) workspacePlan() string {
	if m.Workspace == nil {
		return workspacePlanProfessional
	}

	return normalizeWorkspacePlan(m.Workspace.Plan)
}

func buildMetadata(workspaceID, workspacePlan string) Metadata {
	return Metadata{
		Workspace: &WorkspaceMetadata{
			ID:   workspaceID,
			Plan: normalizeWorkspacePlan(workspacePlan),
		},
	}
}

func workspacePlanFromConfig(integration core.IntegrationContext) string {
	configuredWorkspacePlan := workspacePlanProfessional
	workspacePlanValue, workspacePlanErr := integration.GetConfig("workspacePlan")
	if workspacePlanErr == nil {
		configuredWorkspacePlan = normalizeWorkspacePlan(string(workspacePlanValue))
	}

	return configuredWorkspacePlan
}

func normalizeWorkspacePlan(workspacePlan string) string {
	switch strings.ToLower(strings.TrimSpace(workspacePlan)) {
	case workspacePlanOrganization, workspacePlanEnterpriseAlias:
		return workspacePlanOrganization
	default:
		return workspacePlanProfessional
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
	normalizedWebhookName := strings.TrimSpace(webhookName)
	if normalizedWebhookName == "" {
		return nil
	}

	for i := range webhooks {
		if strings.TrimSpace(webhooks[i].Name) == normalizedWebhookName {
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
