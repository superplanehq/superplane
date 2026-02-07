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
	OwnerID       string `json:"ownerId" mapstructure:"ownerId"`
	WorkspacePlan string `json:"workspacePlan" mapstructure:"workspacePlan"`
}

type Metadata struct {
	OwnerID       string `json:"ownerId" mapstructure:"ownerId"`
	WorkspacePlan string `json:"workspacePlan" mapstructure:"workspacePlan"`
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
2. **Workspace ID (optional):** Use your Render workspace ID (` + "`usr-...`" + ` or ` + "`tea-...`" + `). Leave empty to use the first workspace available to the API key.
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
			Name:        "ownerId",
			Label:       "Workspace ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional Render workspace ID (usr-... or tea-...). Use this if your API key has access to multiple workspaces.",
		},
		{
			Name:     "workspacePlan",
			Label:    "Workspace Plan",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  renderWorkspacePlanProfessional,
			Description: "Render workspace plan used for webhook strategy. " +
				"Use Organization / Enterprise when available.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Professional", Value: renderWorkspacePlanProfessional},
						{Label: "Organization / Enterprise", Value: renderWorkspacePlanOrganization},
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

	owner, err := resolveOwner(client, strings.TrimSpace(config.OwnerID))
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	workspacePlan := normalizeWorkspacePlan(config.WorkspacePlan)
	ctx.Integration.SetMetadata(Metadata{
		OwnerID:       owner.ID,
		WorkspacePlan: workspacePlan,
	})
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

	if configA.Strategy == renderWebhookStrategyResourceType {
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

	if currentConfiguration.Strategy == renderWebhookStrategyResourceType &&
		currentConfiguration.ResourceType != requestedConfiguration.ResourceType {
		return currentConfiguration, false, nil
	}

	mergedConfiguration := currentConfiguration
	mergedConfiguration.EventTypes = normalizeWebhookEventTypes(
		append(currentConfiguration.EventTypes, requestedConfiguration.EventTypes...),
	)

	if len(mergedConfiguration.EventTypes) == 0 {
		mergedConfiguration.EventTypes = renderDefaultEventTypesForWebhook(currentConfiguration)
	}

	return mergedConfiguration, !renderWebhookConfigurationsEqual(currentConfiguration, mergedConfiguration), nil
}

func (r *Render) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "service" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	ownerID, err := r.ownerID(client, ctx.Integration)
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

	ownerID, err := r.ownerID(client, ctx.Integration)
	if err != nil {
		return nil, err
	}

	webhookURL := strings.TrimSpace(ctx.Webhook.GetURL())
	if webhookURL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	webhookConfiguration, err := decodeWebhookConfiguration(ctx.Webhook.GetConfiguration())
	if err != nil {
		return nil, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	webhookName := renderWebhookName(webhookConfiguration)
	eventFilter := renderWebhookEventFilter(webhookConfiguration)
	webhooks, err := client.ListWebhooks(ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list Render webhooks: %w", err)
	}

	candidateWebhooks := make([]Webhook, 0, len(webhooks))
	for _, webhook := range webhooks {
		if strings.TrimSpace(webhook.URL) == webhookURL {
			candidateWebhooks = append(candidateWebhooks, webhook)
		}
	}

	selectedWebhook := (*Webhook)(nil)
	if webhookConfiguration.Strategy == renderWebhookStrategyIntegration {
		selectedWebhook = pickExistingRenderWebhook(candidateWebhooks, webhookName)
	} else {
		selectedWebhook = pickExistingRenderWebhookByName(candidateWebhooks, webhookName)
		if selectedWebhook == nil {
			selectedWebhook = pickExistingRenderWebhookByEventFilter(candidateWebhooks, eventFilter)
		}
	}

	if selectedWebhook != nil {
		retrievedWebhook, retrieveErr := client.GetWebhook(selectedWebhook.ID)
		if retrieveErr != nil {
			return nil, fmt.Errorf("failed to retrieve existing Render webhook: %w", retrieveErr)
		}

		existingEventFilter := normalizeWebhookEventTypes(retrievedWebhook.EventFilter)
		if len(existingEventFilter) == 0 {
			existingEventFilter = normalizeWebhookEventTypes(selectedWebhook.EventFilter)
		}

		existingName := strings.TrimSpace(retrievedWebhook.Name)
		if existingName == "" {
			existingName = strings.TrimSpace(selectedWebhook.Name)
		}

		mergedEventFilter := normalizeWebhookEventTypes(append(existingEventFilter, eventFilter...))
		if len(mergedEventFilter) == 0 {
			mergedEventFilter = eventFilter
		}

		if existingName != webhookName || !slices.Equal(existingEventFilter, mergedEventFilter) || !retrievedWebhook.Enabled {
			_, updateErr := client.UpdateWebhook(selectedWebhook.ID, UpdateWebhookRequest{
				Name:        webhookName,
				URL:         webhookURL,
				Enabled:     true,
				EventFilter: mergedEventFilter,
			})
			if updateErr != nil {
				return nil, fmt.Errorf("failed to update existing Render webhook: %w", updateErr)
			}
		}

		secret := strings.TrimSpace(retrievedWebhook.Secret)
		if secret == "" {
			secret = strings.TrimSpace(selectedWebhook.Secret)
		}

		if secret == "" {
			return nil, fmt.Errorf("render webhook secret is empty")
		}

		if err := ctx.Webhook.SetSecret([]byte(secret)); err != nil {
			return nil, fmt.Errorf("failed to store webhook secret: %w", err)
		}

		return WebhookMetadata{WebhookID: selectedWebhook.ID, OwnerID: ownerID}, nil
	}

	createdWebhook, err := client.CreateWebhook(CreateWebhookRequest{
		OwnerID:     ownerID,
		Name:        webhookName,
		URL:         webhookURL,
		Enabled:     true,
		EventFilter: eventFilter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Render webhook: %w", err)
	}

	secret := strings.TrimSpace(createdWebhook.Secret)
	if secret == "" {
		return nil, fmt.Errorf("render webhook secret is empty")
	}

	if err := ctx.Webhook.SetSecret([]byte(secret)); err != nil {
		return nil, fmt.Errorf("failed to store webhook secret: %w", err)
	}

	return WebhookMetadata{WebhookID: createdWebhook.ID, OwnerID: ownerID}, nil
}

func (r *Render) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}

	if strings.TrimSpace(metadata.WebhookID) == "" {
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

func (r *Render) ownerID(client *Client, integration core.IntegrationContext) (string, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err == nil {
		if strings.TrimSpace(metadata.OwnerID) != "" {
			if strings.TrimSpace(metadata.WorkspacePlan) == "" {
				workspacePlan := renderWorkspacePlanProfessional
				workspacePlanValue, workspacePlanErr := integration.GetConfig("workspacePlan")
				if workspacePlanErr == nil {
					workspacePlan = normalizeWorkspacePlan(string(workspacePlanValue))
				}

				integration.SetMetadata(Metadata{
					OwnerID:       strings.TrimSpace(metadata.OwnerID),
					WorkspacePlan: workspacePlan,
				})
			}

			return strings.TrimSpace(metadata.OwnerID), nil
		}
	}

	ownerIDConfig := ""
	ownerIDConfigValue, err := integration.GetConfig("ownerId")
	if err == nil {
		ownerIDConfig = strings.TrimSpace(string(ownerIDConfigValue))
	}

	owner, err := resolveOwner(client, ownerIDConfig)
	if err != nil {
		return "", err
	}

	workspacePlan, workspacePlanErr := integration.GetConfig("workspacePlan")
	configuredWorkspacePlan := renderWorkspacePlanProfessional
	if workspacePlanErr == nil {
		configuredWorkspacePlan = normalizeWorkspacePlan(string(workspacePlan))
	}

	integration.SetMetadata(Metadata{
		OwnerID:       owner.ID,
		WorkspacePlan: configuredWorkspacePlan,
	})
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

	trimmedOwnerID := strings.TrimSpace(ownerID)
	if trimmedOwnerID == "" {
		return owners[0], nil
	}

	selectedOwner := slices.IndexFunc(owners, func(owner Owner) bool {
		return strings.TrimSpace(owner.ID) == trimmedOwnerID
	})
	if selectedOwner < 0 {
		return Owner{}, fmt.Errorf("workspace %s is not accessible with this API key", trimmedOwnerID)
	}

	return owners[selectedOwner], nil
}

func normalizeWorkspacePlan(workspacePlan string) string {
	switch strings.ToLower(strings.TrimSpace(workspacePlan)) {
	case renderWorkspacePlanOrganization, renderWorkspacePlanEnterpriseAlias:
		return renderWorkspacePlanOrganization
	default:
		return renderWorkspacePlanProfessional
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
