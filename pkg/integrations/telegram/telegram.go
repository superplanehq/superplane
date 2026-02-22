package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("telegram", &Telegram{})
}

type Telegram struct{}

type Configuration struct {
	BotToken string `json:"botToken" mapstructure:"botToken"`
}

type Metadata struct {
	BotID     int64  `json:"botId" mapstructure:"botId"`
	Username  string `json:"username" mapstructure:"username"`
	FirstName string `json:"firstName" mapstructure:"firstName"`
}

type SubscriptionConfiguration struct {
	EventTypes []string `json:"eventTypes" mapstructure:"eventTypes"`
}

func (t *Telegram) Name() string {
	return "telegram"
}

func (t *Telegram) Label() string {
	return "Telegram"
}

func (t *Telegram) Icon() string {
	return "telegram"
}

func (t *Telegram) Description() string {
	return "Send messages and react to events via Telegram bots"
}

func (t *Telegram) Instructions() string {
	return `To set up Telegram integration:

1. Get a bot token from @BotFather and paste it in the field below
2. Disable privacy mode so the bot can receive messages in groups: send /setprivacy to @BotFather, select your bot, and choose Disable
3. Add the bot to your group or channel

Note: if the bot was already in a group before disabling privacy mode, remove and re-add it for the change to take effect.`
}

func (t *Telegram) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "botToken",
			Label:       "Bot Token",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Telegram bot token from BotFather",
		},
	}
}

func (t *Telegram) Components() []core.Component {
	return []core.Component{
		&SendMessage{},
	}
}

func (t *Telegram) Triggers() []core.Trigger {
	return []core.Trigger{
		&OnMention{},
	}
}

func (t *Telegram) Sync(ctx core.SyncContext) error {
	// Get the decrypted bot token
	botTokenBytes, err := ctx.Integration.GetConfig("botToken")
	if err != nil {
		return fmt.Errorf("botToken is required")
	}

	botToken := string(botTokenBytes)
	if botToken == "" {
		return fmt.Errorf("botToken is required")
	}

	// Verify the bot token is valid by getting bot info
	client, err := NewClient(ctx.Integration)
	if err != nil {
		return err
	}

	botUser, err := client.GetMe()
	if err != nil {
		return fmt.Errorf("failed to verify bot token: %v", err)
	}

	// Set webhook URL for receiving updates
	webhookURL := ctx.WebhooksBaseURL
	if webhookURL == "" {
		webhookURL = ctx.BaseURL
	}
	webhookURL = fmt.Sprintf("%s/api/v1/integrations/%s/events", webhookURL, ctx.Integration.ID().String())

	err = client.SetWebhook(webhookURL)
	if err != nil {
		return fmt.Errorf("failed to set webhook: %v", err)
	}

	ctx.Integration.SetMetadata(Metadata{
		BotID:     botUser.ID,
		Username:  botUser.Username,
		FirstName: botUser.FirstName,
	})

	ctx.Integration.Ready()
	return nil
}

func (t *Telegram) HandleRequest(ctx core.HTTPRequestContext) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Logger.Errorf("error reading request body: %v", err)
		ctx.Response.WriteHeader(400)
		return
	}

	var update Update
	if err := json.Unmarshal(body, &update); err != nil {
		ctx.Logger.Errorf("error unmarshaling update: %v", err)
		ctx.Response.WriteHeader(400)
		return
	}

	// Only process messages
	if update.Message == nil {
		ctx.Response.WriteHeader(200)
		return
	}

	// Check if this is a bot mention
	metadata := Metadata{}
	m := ctx.Integration.GetMetadata()
	if m != nil {
		mMap, ok := m.(map[string]any)
		if ok {
			if botID, ok := mMap["botId"].(float64); ok {
				metadata.BotID = int64(botID)
			}
			if username, ok := mMap["username"].(string); ok {
				metadata.Username = username
			}
			if firstName, ok := mMap["firstName"].(string); ok {
				metadata.FirstName = firstName
			}
		}
	}

	isMention := false
	botUsername := "@" + metadata.Username

	// Check for mentions in entities
	for _, entity := range update.Message.Entities {
		if entity.Type == "mention" {
			// Extract the mentioned username from the message text
			mentionedText := update.Message.Text[entity.Offset : entity.Offset+entity.Length]
			if strings.EqualFold(mentionedText, botUsername) {
				isMention = true
				break
			}
		}
	}

	if !isMention {
		ctx.Response.WriteHeader(200)
		return
	}

	// Get subscriptions and dispatch
	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		ctx.Logger.Errorf("error listing subscriptions: %v", err)
		ctx.Response.WriteHeader(500)
		return
	}

	for _, subscription := range subscriptions {
		config := SubscriptionConfiguration{}
		if err := mapstructure.Decode(subscription.Configuration(), &config); err != nil {
			ctx.Logger.Errorf("error decoding subscription configuration: %v", err)
			continue
		}

		if !slices.ContainsFunc(config.EventTypes, func(t string) bool {
			return t == "message_mention"
		}) {
			continue
		}

		// Convert message to map for serialization
		messageMap := map[string]any{
			"message_id": update.Message.MessageID,
			"text":       update.Message.Text,
			"date":       update.Message.Date,
		}

		if update.Message.From != nil {
			messageMap["from"] = map[string]any{
				"id":         update.Message.From.ID,
				"is_bot":     update.Message.From.IsBot,
				"first_name": update.Message.From.FirstName,
				"username":   update.Message.From.Username,
			}
		}

		messageMap["chat"] = map[string]any{
			"id":    update.Message.Chat.ID,
			"type":  update.Message.Chat.Type,
			"title": update.Message.Chat.Title,
		}

		if len(update.Message.Entities) > 0 {
			entities := make([]map[string]any, len(update.Message.Entities))
			for i, entity := range update.Message.Entities {
				entities[i] = map[string]any{
					"type":   entity.Type,
					"offset": entity.Offset,
					"length": entity.Length,
				}
			}
			messageMap["entities"] = entities
		}

		err = subscription.SendMessage(messageMap)
		if err != nil {
			ctx.Logger.Errorf("error sending message from integration: %v", err)
		}
	}

	ctx.Response.WriteHeader(200)
}

func (t *Telegram) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (t *Telegram) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	return []core.IntegrationResource{}, nil
}

func (t *Telegram) Actions() []core.Action {
	return []core.Action{}
}

func (t *Telegram) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}

// Telegram API types for parsing webhook updates

type Update struct {
	UpdateID int64            `json:"update_id"`
	Message  *TelegramMessage `json:"message,omitempty"`
}

type TelegramMessage struct {
	MessageID int64           `json:"message_id"`
	From      *User           `json:"from,omitempty"`
	Chat      Chat            `json:"chat"`
	Text      string          `json:"text,omitempty"`
	Entities  []MessageEntity `json:"entities,omitempty"`
	Date      int64           `json:"date"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username,omitempty"`
}

type Chat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
}

type MessageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}
