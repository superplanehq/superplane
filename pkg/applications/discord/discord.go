package discord

import (
	"fmt"
	"regexp"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterApplication("discord", &Discord{})
}

// webhookURLPattern matches Discord webhook URLs
// Format: https://discord.com/api/webhooks/{webhook.id}/{webhook.token}
// or: https://discordapp.com/api/webhooks/{webhook.id}/{webhook.token}
var webhookURLPattern = regexp.MustCompile(`^https://(discord\.com|discordapp\.com)/api/webhooks/\d+/[\w-]+$`)

type Discord struct{}

type Configuration struct {
	WebhookURL string `json:"webhookUrl" mapstructure:"webhookUrl"`
}

type Metadata struct {
	WebhookID string `json:"webhookId" mapstructure:"webhookId"`
}

func (d *Discord) Name() string {
	return "discord"
}

func (d *Discord) Label() string {
	return "Discord"
}

func (d *Discord) Icon() string {
	return "discord"
}

func (d *Discord) Description() string {
	return "Send messages to Discord channels"
}

func (d *Discord) InstallationInstructions() string {
	return `To set up Discord integration:

1. Open your Discord server settings
2. Go to **Integrations** → **Webhooks**
3. Click **New Webhook**
4. Select the channel where messages should be sent
5. Copy the **Webhook URL** and paste it below`
}

func (d *Discord) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "webhookUrl",
			Label:       "Webhook URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   false,
			Description: "Discord webhook URL (https://discord.com/api/webhooks/{id}/{token})",
		},
	}
}

func (d *Discord) Components() []core.Component {
	return []core.Component{
		&SendTextMessage{},
	}
}

func (d *Discord) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (d *Discord) Sync(ctx core.SyncContext) error {
	// Get the decrypted webhook URL using GetConfig (sensitive fields are encrypted)
	webhookURLBytes, err := ctx.AppInstallation.GetConfig("webhookUrl")
	if err != nil {
		return fmt.Errorf("webhookUrl is required")
	}

	webhookURL := string(webhookURLBytes)
	if webhookURL == "" {
		return fmt.Errorf("webhookUrl is required")
	}

	if !webhookURLPattern.MatchString(webhookURL) {
		return fmt.Errorf("invalid webhook URL format. Expected: https://discord.com/api/webhooks/{id}/{token}")
	}

	// Parse webhook ID from URL for metadata
	webhookID, err := parseWebhookID(webhookURL)
	if err != nil {
		return fmt.Errorf("failed to parse webhook URL: %v", err)
	}

	// Verify the webhook is valid by making a GET request
	client, err := NewClient(ctx.AppInstallation)
	if err != nil {
		return err
	}

	webhookInfo, err := client.GetWebhook()
	if err != nil {
		return fmt.Errorf("failed to verify webhook: %v", err)
	}

	ctx.AppInstallation.SetMetadata(Metadata{
		WebhookID: webhookID,
	})

	ctx.AppInstallation.SetState("ready", fmt.Sprintf("Connected to channel: %s", webhookInfo.ChannelID))
	return nil
}

func (d *Discord) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op: Discord webhook-based integration doesn't receive incoming requests
}

func (d *Discord) CompareWebhookConfig(a, b any) (bool, error) {
	return true, nil
}

func (d *Discord) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.ApplicationResource, error) {
	// No resources for webhook-based integration
	return []core.ApplicationResource{}, nil
}

func (d *Discord) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (d *Discord) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}

// parseWebhookID extracts the webhook ID from a Discord webhook URL
func parseWebhookID(webhookURL string) (string, error) {
	pattern := regexp.MustCompile(`/webhooks/(\d+)/`)
	matches := pattern.FindStringSubmatch(webhookURL)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract webhook ID from URL")
	}
	return matches[1], nil
}
