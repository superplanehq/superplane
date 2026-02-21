package telegram

import (
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnMention struct{}

type OnMentionConfiguration struct {
	ChatID string `json:"chatId" mapstructure:"chatId"`
}

type OnMentionMetadata struct {
	AppSubscriptionID *string `json:"appSubscriptionID,omitempty" mapstructure:"appSubscriptionID,omitempty"`
	ChatID            *string `json:"chatId,omitempty" mapstructure:"chatId,omitempty"`
	ChatName          *string `json:"chatName,omitempty" mapstructure:"chatName,omitempty"`
}

func (t *OnMention) Name() string {
	return "telegram.onMention"
}

func (t *OnMention) Label() string {
	return "On Mention"
}

func (t *OnMention) Description() string {
	return "Fires when the bot is mentioned in a message"
}

func (t *OnMention) Documentation() string {
	return `The On Mention trigger starts a workflow execution when the Telegram bot is mentioned in a message.

## Use Cases

- **Bot commands**: Process commands from Telegram messages
- **Bot interactions**: Create interactive Telegram bots
- **Team workflows**: Trigger workflows from Telegram conversations
- **Notification processing**: Process and respond to mentions

## Configuration

- **Chat ID**: Optional chat filter - if specified, only mentions in this chat will trigger (leave empty to listen to all chats)

## Event Data

Each mention event includes:
- **message_id**: The unique message identifier
- **from**: User who mentioned the bot
- **chat**: Chat where the mention occurred
- **text**: The message text containing the mention
- **date**: Unix timestamp of the message

## Setup

This trigger automatically sets up a webhook subscription when configured. The subscription is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnMention) Icon() string {
	return "telegram"
}

func (t *OnMention) Color() string {
	return "gray"
}

func (t *OnMention) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "chatId",
			Label:       "Chat ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional: filter mentions by specific chat ID",
		},
	}
}

func (t *OnMention) Setup(ctx core.TriggerContext) error {
	// Decode existing metadata
	var metadata OnMentionMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	// Decode configuration
	var config OnMentionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Subscribe to message_mention events if not already subscribed
	subscriptionID, err := t.subscribe(ctx, metadata)
	if err != nil {
		return fmt.Errorf("failed to subscribe to message events: %w", err)
	}

	newMetadata := OnMentionMetadata{
		AppSubscriptionID: subscriptionID,
	}

	if config.ChatID != "" {
		newMetadata.ChatID = &config.ChatID

		client, err := NewClient(ctx.Integration)
		if err != nil {
			return fmt.Errorf("failed to create Telegram client: %w", err)
		}

		chatInfo, err := client.GetChat(config.ChatID)
		if err != nil {
			return fmt.Errorf("chat validation failed: %w", err)
		}

		name := ChatDisplayName(chatInfo)
		newMetadata.ChatName = &name
	}

	return ctx.Metadata.Set(newMetadata)
}

func (t *OnMention) subscribe(ctx core.TriggerContext, metadata OnMentionMetadata) (*string, error) {
	if metadata.AppSubscriptionID != nil {
		logrus.Infof("using existing subscription %s", *metadata.AppSubscriptionID)
		return metadata.AppSubscriptionID, nil
	}

	logrus.Infof("creating new subscription")

	subscriptionID, err := ctx.Integration.Subscribe(SubscriptionConfiguration{
		EventTypes: []string{"message_mention"},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to message events: %w", err)
	}

	s := subscriptionID.String()
	return &s, nil
}

func (t *OnMention) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnMention) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnMention) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (t *OnMention) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	var config OnMentionConfiguration
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Get message from context as map
	message, ok := ctx.Message.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid message type")
	}

	// If chat ID configuration is set and does not match the message chat, ignore
	if config.ChatID != "" {
		chatData, ok := message["chat"].(map[string]any)
		if !ok {
			return fmt.Errorf("invalid chat data in message")
		}

		// Handle both int64 and float64 types since JSON unmarshaling may represent numbers as float64
		var chatID int64
		switch v := chatData["id"].(type) {
		case int64:
			chatID = v
		case float64:
			chatID = int64(v)
		default:
			return fmt.Errorf("invalid chat ID type")
		}

		messageChatID := fmt.Sprintf("%d", chatID)
		if config.ChatID != messageChatID {
			ctx.Logger.Infof("message chat %s does not match configuration chat %s, ignoring", messageChatID, config.ChatID)
			return nil
		}
	}

	// Emit the mention event
	return ctx.Events.Emit("telegram.message.mention", message)
}

func (t *OnMention) Cleanup(ctx core.TriggerContext) error {
	return nil
}
