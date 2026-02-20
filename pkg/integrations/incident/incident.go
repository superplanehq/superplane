package incident

import (
	"context"
	"encoding/base64"
	"fmt"
	"maps"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("incident", &IncidentIO{}, &IncidentIOWebhookHandler{})
}

type IncidentIO struct{}

type Configuration struct {
	APIKey               string `json:"apiKey"`
	WebhookSigningSecret string `json:"webhookSigningSecret"`
}

func (i *IncidentIO) Name() string {
	return "incident"
}

func (i *IncidentIO) Label() string {
	return "Incident"
}

func (i *IncidentIO) Icon() string {
	return "alert-triangle"
}

func (i *IncidentIO) Description() string {
	return "Manage and react to incidents in incident.io"
}

func (i *IncidentIO) Instructions() string {
	return `## API integration

1. In [incident.io Settings > API keys](https://app.incident.io/settings/api-keys), click **Create API key** and give it a name.
2. Under **Add permissions**, select exactly these (use "Find a permission" if needed):
   - **View data, like public incidents and organisation settings** (needed to read severities)
   - **Create incidents** (needed for the Create Incident action)
   - **View all incident data, including private incidents** (only if you use private incidents)
3. Create the key and **paste the API key** in the Configuration section below.

## Webhook integration

Required for the **On Incident** trigger. Until this is done, the trigger will not receive events.

1. Copy the **webhook URL** from the Webhook section below.
2. In incident.io go to **Settings → Webhooks**, create a new endpoint, and paste that URL. Subscribe to **Public incident created (v2)** and **Public incident updated (v2)**.
3. Copy the **Signing secret** from the endpoint and paste it in **Webhook signing secret** in the Configuration section above, then save.

The On Incident trigger becomes operational once the URL is registered in incident.io and the signing secret is saved here.`
}

func (i *IncidentIO) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "apiKey",
			Label:       "API Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "API key from incident.io. Create one in Settings > API keys with permissions: View data (public incidents and organisation settings), Create incidents; optionally View all incident data (private incidents).",
		},
		{
			Name:        "webhookSigningSecret",
			Label:       "Webhook signing secret",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Sensitive:   true,
			Description: "From your incident.io webhook endpoint (Settings → Webhooks). Paste the signing secret here so the On Incident trigger can verify requests.",
			Placeholder: "whsec_...",
		},
	}
}

func (i *IncidentIO) Components() []core.Component {
	return []core.Component{
		&CreateIncident{},
	}
}

func (i *IncidentIO) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnIncident{},
	}
}

func (i *IncidentIO) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (i *IncidentIO) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	// Validate API key by listing severities
	_, err = client.ListSeverities()
	if err != nil {
		return fmt.Errorf("error validating API key (listing severities): %w", err)
	}

	if ctx.Encryptor != nil {
		tx := ctx.Tx
		if tx == nil {
			tx = database.Conn()
		}
		if ensureErr := EnsureWebhookExists(tx, ctx.Integration.ID()); ensureErr != nil {
			ctx.Logger.WithError(ensureErr).Warn("failed to ensure incident webhook")
		}
	}

	setIncidentIntegrationMetadata(ctx)

	ctx.Integration.Ready()
	return nil
}

// setIncidentIntegrationMetadata sets webhookUrl and webhookSigningSecretConfigured on the integration metadata
// so the UI can show the webhook URL and secret status without organization-layer logic.
func setIncidentIntegrationMetadata(ctx core.SyncContext) {
	var m map[string]any
	if meta := ctx.Integration.GetMetadata(); meta != nil {
		if mm, ok := meta.(map[string]any); ok {
			m = maps.Clone(mm)
		}
	}
	if m == nil {
		m = make(map[string]any)
	}

	if ctx.WebhooksBaseURL != "" {
		tx := ctx.Tx
		if tx == nil {
			tx = database.Conn()
		}
		webhooks, err := models.ListIntegrationWebhooks(tx, ctx.Integration.ID())
		if err == nil && len(webhooks) > 0 {
			m["webhookUrl"] = ctx.WebhooksBaseURL + "/api/v1/webhooks/" + webhooks[0].ID.String()
		}
	}

	configured := isWebhookSigningSecretConfigured(ctx)
	m["webhookSigningSecretConfigured"] = configured

	ctx.Integration.SetMetadata(m)
}

// isWebhookSigningSecretConfigured returns true only if the integration has a non-empty webhook signing secret
// (decrypted and trimmed). Encrypting an empty string yields non-empty ciphertext, so we must decrypt to check.
func isWebhookSigningSecretConfigured(ctx core.SyncContext) bool {
	if ctx.Encryptor == nil {
		return false
	}
	configMap, ok := ctx.Configuration.(map[string]any)
	if !ok {
		return false
	}
	raw, _ := configMap["webhookSigningSecret"].(string)
	if raw == "" || raw == "<redacted>" {
		return false
	}
	ciphertext, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(ciphertext) == 0 {
		return false
	}
	plaintext, err := ctx.Encryptor.Decrypt(context.Background(), ciphertext, []byte(ctx.Integration.ID().String()))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(plaintext)) != ""
}

func (i *IncidentIO) HandleRequest(ctx core.HTTPRequestContext) {}

func (i *IncidentIO) Actions() []core.Action {
	return nil
}

func (i *IncidentIO) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

func (i *IncidentIO) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	if resourceType != "severity" {
		return []core.IntegrationResource{}, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	severities, err := client.ListSeverities()
	if err != nil {
		return nil, fmt.Errorf("failed to list severities: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(severities))
	for _, s := range severities {
		resources = append(resources, core.IntegrationResource{
			Type: "severity",
			Name: s.Name,
			ID:   s.ID,
		})
	}
	return resources, nil
}
