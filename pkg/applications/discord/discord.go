package discord

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterApplication("discord", &Discord{})
}

type Discord struct{}

type Configuration struct {
	BotToken string `json:"botToken" mapstructure:"botToken"`
}

type Metadata struct {
	BotID    string `json:"botId" mapstructure:"botId"`
	Username string `json:"username" mapstructure:"username"`
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

1. Go to the **Discord Developer Portal** (https://discord.com/developers/applications)
2. Click **New Application** and give it a name
3. Go to the **Bot** section and click **Add Bot**
4. Under **Token**, click **Reset Token** then **Copy** to copy the bot token
5. Go to **OAuth2** → **URL Generator**:
   - Under **Scopes**, select **bot**
   - Under **Bot Permissions**, select: **View Channels**, **Send Messages**
6. Copy the generated URL and open it to invite the bot to your server
7. Paste the **Bot Token** below`
}

func (d *Discord) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "botToken",
			Label:       "Bot Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Discord bot token from the Developer Portal",
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
	// Get the decrypted bot token using GetConfig (sensitive fields are encrypted)
	botTokenBytes, err := ctx.AppInstallation.GetConfig("botToken")
	if err != nil {
		return fmt.Errorf("botToken is required")
	}

	botToken := string(botTokenBytes)
	if botToken == "" {
		return fmt.Errorf("botToken is required")
	}

	// Verify the bot token is valid by getting the current user
	client, err := NewClient(ctx.AppInstallation)
	if err != nil {
		return err
	}

	botUser, err := client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("failed to verify bot token: %v", err)
	}

	ctx.AppInstallation.SetMetadata(Metadata{
		BotID:    botUser.ID,
		Username: botUser.Username,
	})

	ctx.AppInstallation.SetState("ready", fmt.Sprintf("Connected as: %s", botUser.Username))
	return nil
}

func (d *Discord) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op: Discord bot integration doesn't receive incoming HTTP requests
}

func (d *Discord) CompareWebhookConfig(a, b any) (bool, error) {
	return true, nil
}

func (d *Discord) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.ApplicationResource, error) {
	if resourceType != "channel" {
		return []core.ApplicationResource{}, nil
	}

	client, err := NewClient(ctx.AppInstallation)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Errorf("Discord: failed to create client: %v", err)
		}
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	guilds, err := client.GetCurrentUserGuilds()
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Errorf("Discord: failed to get guilds: %v", err)
		}
		return nil, fmt.Errorf("failed to get guilds: %w", err)
	}

	if ctx.Logger != nil {
		ctx.Logger.Infof("Discord: found %d guilds", len(guilds))
	}

	var resources []core.ApplicationResource
	for _, guild := range guilds {
		if ctx.Logger != nil {
			ctx.Logger.Infof("Discord: fetching channels for guild %s (%s)", guild.ID, guild.Name)
		}
		channels, err := client.GetGuildChannels(guild.ID)
		if err != nil {
			if ctx.Logger != nil {
				ctx.Logger.Warnf("Discord: failed to get channels for guild %s: %v", guild.ID, err)
			}
			continue // Skip guilds where we can't list channels
		}

		if ctx.Logger != nil {
			ctx.Logger.Infof("Discord: found %d channels in guild %s", len(channels), guild.Name)
		}
		for _, channel := range channels {
			// Only include text channels (type 0)
			if channel.Type == 0 {
				resources = append(resources, core.ApplicationResource{
					Type: "channel",
					ID:   channel.ID,
					Name: fmt.Sprintf("#%s (%s)", channel.Name, guild.Name),
				})
			}
		}
	}

	if ctx.Logger != nil {
		ctx.Logger.Infof("Discord: returning %d channel resources", len(resources))
	}
	return resources, nil
}

func (d *Discord) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (d *Discord) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}
