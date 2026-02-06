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
	APIKey  string `json:"apiKey" mapstructure:"apiKey"`
	OwnerID string `json:"ownerId" mapstructure:"ownerId"`
}

type Metadata struct {
	OwnerID string `json:"ownerId" mapstructure:"ownerId"`
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
	return "Deploy and manage Render services, and react to Render deploy/service events"
}

func (r *Render) Instructions() string {
	return `
1. **API Key:** Create it in [Render Account Settings -> API Keys](https://dashboard.render.com/u/settings#api-keys).
2. **Workspace ID (optional):** Use your Render workspace ID (` + "`usr-...`" + ` or ` + "`tea-...`" + `). Leave empty to use the first workspace available to the API key.
3. **Auth:** SuperPlane sends requests to [Render API v1](https://api.render.com/v1/) using ` + "`Authorization: Bearer <API_KEY>`" + `.


4. **Webhooks:** SuperPlane configures Render webhooks automatically via the [Render Webhooks API](https://render.com/docs/webhooks). No manual setup is required.
5. **Troubleshooting:** Check [Render Dashboard -> Integrations -> Webhooks](https://dashboard.render.com/) and the [Render webhook docs](https://render.com/docs/webhooks).

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
	}
}

func (r *Render) Components() []core.Component {
	return []core.Component{
		&TriggerDeploy{},
	}
}

func (r *Render) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnEvent{},
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

	ownerID, err := resolveOwnerID(client, strings.TrimSpace(config.OwnerID))
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	ctx.Integration.SetMetadata(Metadata{OwnerID: ownerID})
	ctx.Integration.Ready()
	return nil
}

func (r *Render) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (r *Render) CompareWebhookConfig(a, b any) (bool, error) {
	return true, nil
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

	webhooks, err := client.ListWebhooks(ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list Render webhooks: %w", err)
	}

	for _, webhook := range webhooks {
		if strings.TrimSpace(webhook.URL) != webhookURL {
			continue
		}

		retrievedWebhook, retrieveErr := client.GetWebhook(webhook.ID)
		if retrieveErr != nil {
			return nil, fmt.Errorf("failed to retrieve existing Render webhook: %w", retrieveErr)
		}

		secret := strings.TrimSpace(retrievedWebhook.Secret)
		if secret == "" {
			secret = strings.TrimSpace(webhook.Secret)
		}

		if secret == "" {
			return nil, fmt.Errorf("render webhook secret is empty")
		}

		if err := ctx.Webhook.SetSecret([]byte(secret)); err != nil {
			return nil, fmt.Errorf("failed to store webhook secret: %w", err)
		}

		return WebhookMetadata{WebhookID: webhook.ID, OwnerID: ownerID}, nil
	}

	createdWebhook, err := client.CreateWebhook(CreateWebhookRequest{
		OwnerID:     ownerID,
		Name:        "SuperPlane",
		URL:         webhookURL,
		Enabled:     true,
		EventFilter: []string{},
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
			return strings.TrimSpace(metadata.OwnerID), nil
		}
	}

	ownerIDConfig := ""
	ownerIDConfigValue, err := integration.GetConfig("ownerId")
	if err == nil {
		ownerIDConfig = strings.TrimSpace(string(ownerIDConfigValue))
	}

	ownerID, err := resolveOwnerID(client, ownerIDConfig)
	if err != nil {
		return "", err
	}

	integration.SetMetadata(Metadata{OwnerID: ownerID})
	return ownerID, nil
}

func resolveOwnerID(client *Client, ownerID string) (string, error) {
	owners, err := client.ListOwners()
	if err != nil {
		return "", err
	}

	if len(owners) == 0 {
		return "", fmt.Errorf("no workspaces found for this API key")
	}

	trimmedOwnerID := strings.TrimSpace(ownerID)
	if trimmedOwnerID == "" {
		return owners[0].ID, nil
	}

	if !slices.ContainsFunc(owners, func(owner Owner) bool {
		return owner.ID == trimmedOwnerID
	}) {
		return "", fmt.Errorf("workspace %s is not accessible with this API key", trimmedOwnerID)
	}

	return trimmedOwnerID, nil
}
