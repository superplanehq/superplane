package discord

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type SendTextMessage struct{}

type SendTextMessageConfiguration struct {
	Channel          string `json:"channel" mapstructure:"channel"`
	Content          string `json:"content" mapstructure:"content"`
	EmbedTitle       string `json:"embedTitle" mapstructure:"embedTitle"`
	EmbedDescription string `json:"embedDescription" mapstructure:"embedDescription"`
	EmbedColor       string `json:"embedColor" mapstructure:"embedColor"`
	EmbedURL         string `json:"embedUrl" mapstructure:"embedUrl"`
}

type SendTextMessageMetadata struct {
	HasEmbed bool             `json:"hasEmbed" mapstructure:"hasEmbed"`
	Channel  *ChannelMetadata `json:"channel" mapstructure:"channel"`
}

type ChannelMetadata struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
}

func (c *SendTextMessage) Name() string {
	return "discord.sendTextMessage"
}

func (c *SendTextMessage) Label() string {
	return "Send Text Message"
}

func (c *SendTextMessage) Description() string {
	return "Send a text message to a Discord channel"
}

func (c *SendTextMessage) Icon() string {
	return "discord"
}

func (c *SendTextMessage) Color() string {
	return "gray"
}

func (c *SendTextMessage) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *SendTextMessage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "channel",
			Label:    "Channel",
			Type:     configuration.FieldTypeAppInstallationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "channel",
				},
			},
			Description: "Discord channel to send the message to",
		},
		{
			Name:        "content",
			Label:       "Content",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Plain text message content (max 2000 characters)",
		},
		{
			Name:        "embedTitle",
			Label:       "Embed Title",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Title for the rich embed",
		},
		{
			Name:        "embedDescription",
			Label:       "Embed Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Description text for the rich embed",
		},
		{
			Name:        "embedColor",
			Label:       "Embed Color",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Hex color code for the embed (e.g., #5865F2)",
		},
		{
			Name:        "embedUrl",
			Label:       "Embed URL",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "URL to link from the embed title",
		},
	}
}

func (c *SendTextMessage) Setup(ctx core.SetupContext) error {
	var config SendTextMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Channel == "" {
		return errors.New("channel is required")
	}

	// At least content or embed must be provided
	hasContent := config.Content != ""
	hasEmbed := config.EmbedTitle != "" || config.EmbedDescription != ""

	if !hasContent && !hasEmbed {
		return fmt.Errorf("either content or embed (title/description) is required")
	}

	// Validate content length
	if len(config.Content) > 2000 {
		return fmt.Errorf("content exceeds maximum length of 2000 characters")
	}

	// Validate color format if provided
	if config.EmbedColor != "" {
		if _, err := parseHexColor(config.EmbedColor); err != nil {
			return fmt.Errorf("invalid embed color: %w", err)
		}
	}

	// Get channel info to store in metadata
	client, err := NewClient(ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("failed to create Discord client: %w", err)
	}

	channelInfo, err := client.GetChannel(config.Channel)
	if err != nil {
		return fmt.Errorf("channel validation failed: %w", err)
	}

	metadata := SendTextMessageMetadata{
		HasEmbed: hasEmbed,
		Channel: &ChannelMetadata{
			ID:   channelInfo.ID,
			Name: channelInfo.Name,
		},
	}

	return ctx.Metadata.Set(metadata)
}

func (c *SendTextMessage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *SendTextMessage) Execute(ctx core.ExecutionContext) error {
	var config SendTextMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Channel == "" {
		return errors.New("channel is required")
	}

	client, err := NewClient(ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("failed to create Discord client: %w", err)
	}

	// Build the message request
	req := CreateMessageRequest{
		Content: config.Content,
	}

	// Add embed if title or description is provided
	if config.EmbedTitle != "" || config.EmbedDescription != "" {
		embed := Embed{
			Title:       config.EmbedTitle,
			Description: config.EmbedDescription,
			URL:         config.EmbedURL,
		}

		if config.EmbedColor != "" {
			color, err := parseHexColor(config.EmbedColor)
			if err == nil {
				embed.Color = color
			}
		}

		req.Embeds = []Embed{embed}
	}

	response, err := client.CreateMessage(config.Channel, req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"discord.message.sent",
		[]any{map[string]any{
			"id":         response.ID,
			"channel_id": response.ChannelID,
			"content":    response.Content,
			"timestamp":  response.Timestamp,
			"author": map[string]any{
				"id":       response.Author.ID,
				"username": response.Author.Username,
				"bot":      response.Author.Bot,
			},
		}},
	)
}

func (c *SendTextMessage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *SendTextMessage) Actions() []core.Action {
	return []core.Action{}
}

func (c *SendTextMessage) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *SendTextMessage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

// parseHexColor converts a hex color string to decimal integer
// Supports formats: #RGB, #RRGGBB, RGB, RRGGBB
func parseHexColor(hex string) (int, error) {
	hex = strings.TrimPrefix(hex, "#")

	// Expand shorthand notation
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}

	if len(hex) != 6 {
		return 0, fmt.Errorf("invalid color format: expected 6 hex characters")
	}

	value, err := strconv.ParseInt(hex, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid hex value: %w", err)
	}

	return int(value), nil
}
