package slack

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type SendTextMessage struct{}

type SendTextMessageConfiguration struct {
	Channel string `json:"channel" mapstructure:"channel"`
	Text    string `json:"text" mapstructure:"text"`
}

type SendTextMessageMetadata struct {
	Channel *ChannelMetadata `json:"channel" mapstructure:"channel"`
}

type ChannelMetadata struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
}

func (c *SendTextMessage) Name() string {
	return "slack.sendTextMessage"
}

func (c *SendTextMessage) Label() string {
	return "Send Text Message"
}

func (c *SendTextMessage) Description() string {
	return "Send a text message to a Slack channel"
}

func (c *SendTextMessage) Icon() string {
	return "slack"
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
			Name:        "channel",
			Label:       "Channel",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Channel name (e.g., #general) or channel ID (e.g., C1234567890)",
		},
		{
			Name:     "text",
			Label:    "Text",
			Type:     configuration.FieldTypeText,
			Required: true,
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

	client, err := NewClient(ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("failed to create Slack client: %w", err)
	}

	channelInfo, err := client.GetChannelInfo(config.Channel)
	if err != nil {
		return fmt.Errorf("channel validation failed: %w", err)
	}

	metadata := SendTextMessageMetadata{
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
		return fmt.Errorf("failed to create Slack client: %w", err)
	}

	response, err := client.PostMessage(ChatPostMessageRequest{
		Channel: config.Channel,
		Text:    config.Text,
	})

	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"slack.message.sent",
		[]any{response.Message},
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
