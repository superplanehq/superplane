package teams

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// SendTextMessage sends a text message to a Teams channel.
type SendTextMessage struct{}

// SendTextMessageConfiguration defines the action's configurable fields.
type SendTextMessageConfiguration struct {
	Channel string `json:"channel" mapstructure:"channel"`
	Text    string `json:"text" mapstructure:"text"`
}

// SendTextMessageMetadata stores metadata after action setup.
type SendTextMessageMetadata struct {
	Channel *ChannelMetadata `json:"channel" mapstructure:"channel"`
}

func (c *SendTextMessage) Name() string {
	return "teams.sendTextMessage"
}

func (c *SendTextMessage) Label() string {
	return "Send Text Message"
}

func (c *SendTextMessage) Description() string {
	return "Send a text message to a Microsoft Teams channel"
}

func (c *SendTextMessage) Documentation() string {
	return `The Send Text Message component sends a text message to a Microsoft Teams channel.

## Use Cases

- **Notifications**: Send notifications about workflow events or system status
- **Alerts**: Alert teams about important events or errors
- **Updates**: Provide status updates on long-running processes
- **Team communication**: Automate team communications from workflows

## Configuration

- **Channel**: Select the Teams channel to send the message to
- **Text**: The message text to send (supports expressions)

## Output

Returns metadata about the sent message including the message ID and timestamp.

## Notes

- The Teams bot must be installed in the team containing the target channel
- Messages are sent as the configured bot user
- The bot requires the appropriate permissions to post to the selected channel`
}

func (c *SendTextMessage) Icon() string {
	return "teams"
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
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "channel",
				},
			},
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

	// Look up channel name from the Graph API
	channelName := config.Channel
	client, err := NewClient(ctx.Integration)
	if err == nil {
		channelInfo, err := client.FindChannelByID(config.Channel)
		if err == nil {
			channelName = fmt.Sprintf("#%s (%s)", channelInfo.DisplayName, channelInfo.TeamName)
		}
	}

	metadata := SendTextMessageMetadata{
		Channel: &ChannelMetadata{
			ID:   config.Channel,
			Name: channelName,
		},
	}

	return ctx.Metadata.Set(metadata)
}

func (c *SendTextMessage) Execute(ctx core.ExecutionContext) error {
	var config SendTextMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Channel == "" {
		return errors.New("channel is required")
	}

	if config.Text == "" {
		return errors.New("text is required")
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Teams client: %w", err)
	}

	// The channel resource ID encodes the serviceUrl and conversationId.
	// We use the default Teams service URL and construct the conversation.
	serviceURL := "https://smba.trafficmanager.net/teams/"
	conversationID := config.Channel

	activity := Activity{
		Type: "message",
		Text: config.Text,
	}

	response, err := client.SendActivity(serviceURL, conversationID, activity)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"teams.message.sent",
		[]any{map[string]any{
			"id":             response.ID,
			"conversationId": conversationID,
			"text":           config.Text,
			"timestamp":      response.Timestamp,
		}},
	)
}

func (c *SendTextMessage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
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

func (c *SendTextMessage) Cleanup(ctx core.SetupContext) error {
	return nil
}
