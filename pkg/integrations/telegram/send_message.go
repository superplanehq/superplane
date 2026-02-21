package telegram

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type SendMessage struct{}

type SendMessageConfiguration struct {
	ChatID    string `json:"chatId" mapstructure:"chatId"`
	Text      string `json:"text" mapstructure:"text"`
	ParseMode string `json:"parseMode" mapstructure:"parseMode"`
}

type SendMessageMetadata struct {
	Chat *SendMessageChatMetadata `json:"chat" mapstructure:"chat"`
}

type SendMessageChatMetadata struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
}

func (c *SendMessage) Name() string {
	return "telegram.sendMessage"
}

func (c *SendMessage) Label() string {
	return "Send Message"
}

func (c *SendMessage) Description() string {
	return "Send a text message to a Telegram chat"
}

func (c *SendMessage) Documentation() string {
	return `The Send Message component sends a text message to a Telegram chat.

## Use Cases

- **Notifications**: Send notifications about workflow events or system status
- **Alerts**: Alert teams about important events or errors
- **Updates**: Provide status updates on long-running processes
- **Bot interactions**: Send automated responses to users

## Configuration

- **Chat ID**: The Telegram chat ID (can be a user, group, or channel)
- **Text**: The message text to send
- **Parse Mode**: Optional formatting mode (Markdown)

## Output

Returns metadata about the sent message including message ID, chat ID, text, and timestamp.

## Notes

- The bot must have permission to post messages in the specified chat
- For groups and channels, add the bot as a member first
- Use parse mode for rich text formatting in your messages
- Chat ID can be negative for groups and channels`
}

func (c *SendMessage) Icon() string {
	return "telegram"
}

func (c *SendMessage) Color() string {
	return "gray"
}

func (c *SendMessage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *SendMessage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "chatId",
			Label:       "Chat ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Telegram chat ID (user, group, or channel). Message @userinfobot on Telegram to find your chat ID, or add it to a group to get the group's ID.",
		},
		{
			Name:        "text",
			Label:       "Text",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "Message text to send",
		},
		{
			Name:        "parseMode",
			Label:       "Parse Mode",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Message formatting mode",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "None", Value: "none"},
						{Label: "Markdown", Value: "Markdown"},
					},
				},
			},
		},
	}
}

func (c *SendMessage) Setup(ctx core.SetupContext) error {
	var config SendMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ChatID == "" {
		return errors.New("chatId is required")
	}

	if config.Text == "" {
		return errors.New("text is required")
	}

	if config.ParseMode != "" && config.ParseMode != "none" && config.ParseMode != "Markdown" {
		return fmt.Errorf("invalid parseMode %q: must be none or Markdown", config.ParseMode)
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Telegram client: %w", err)
	}

	chatInfo, err := client.GetChat(config.ChatID)
	if err != nil {
		return fmt.Errorf("chat validation failed: %w", err)
	}

	metadata := SendMessageMetadata{
		Chat: &SendMessageChatMetadata{
			ID:   config.ChatID,
			Name: ChatDisplayName(chatInfo),
		},
	}

	return ctx.Metadata.Set(metadata)
}

func (c *SendMessage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SendMessage) Execute(ctx core.ExecutionContext) error {
	var config SendMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ChatID == "" {
		return errors.New("chatId is required")
	}

	if config.Text == "" {
		return errors.New("text is required")
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Telegram client: %w", err)
	}

	parseMode := config.ParseMode
	if parseMode == "none" {
		parseMode = ""
	}

	response, err := client.SendMessage(config.ChatID, config.Text, parseMode)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"telegram.message.sent",
		[]any{map[string]any{
			"message_id": response.MessageID,
			"chat_id":    response.Chat.ID,
			"text":       response.Text,
			"date":       response.Date,
		}},
	)
}

func (c *SendMessage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *SendMessage) Actions() []core.Action {
	return []core.Action{}
}

func (c *SendMessage) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *SendMessage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *SendMessage) Cleanup(ctx core.SetupContext) error {
	return nil
}
