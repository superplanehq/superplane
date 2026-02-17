package teams

import (
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// OnMessage triggers when any message is posted in a Teams channel.
type OnMessage struct{}

// OnMessageConfiguration defines the trigger's configurable fields.
type OnMessageConfiguration struct {
	Channel string `json:"channel" mapstructure:"channel"`
}

// OnMessageMetadata stores metadata after trigger setup.
type OnMessageMetadata struct {
	Channel           *ChannelMetadata `json:"channel,omitempty" mapstructure:"channel,omitempty"`
	AppSubscriptionID *string          `json:"appSubscriptionID,omitempty" mapstructure:"appSubscriptionID,omitempty"`
}

func (t *OnMessage) Name() string {
	return "teams.onMessage"
}

func (t *OnMessage) Label() string {
	return "On Message"
}

func (t *OnMessage) Description() string {
	return "Listen to all messages in a Teams channel"
}

func (t *OnMessage) Documentation() string {
	return `The On Message trigger starts a workflow execution when any message is posted in a Teams channel.

## Use Cases

- **Message monitoring**: React to any message in a channel
- **Keyword detection**: Process messages looking for specific content
- **Activity tracking**: Track channel activity for analytics
- **Auto-responses**: Automatically respond to specific message patterns

## Configuration

- **Channel**: Optional channel filter — if specified, only messages in this channel will trigger (leave empty to listen to all channels)

## Event Data

Each message event includes:
- **text**: The message text
- **from**: User who sent the message (ID and name)
- **conversation**: Channel and team information
- **timestamp**: When the message was sent
- **serviceUrl**: Bot Framework service URL for sending replies

## Setup

This trigger automatically sets up a subscription for channel message events when configured.

## Important

This trigger requires **Resource-Specific Consent (RSC)** permissions in the Teams app manifest. Specifically, the app must include the ` + "`ChannelMessage.Read.Group`" + ` permission. Without this permission, the bot will only receive messages where it is @mentioned.

The generated Teams app manifest includes this permission by default. If you created the manifest manually, ensure this RSC permission is included.`
}

func (t *OnMessage) Icon() string {
	return "teams"
}

func (t *OnMessage) Color() string {
	return "gray"
}

func (t *OnMessage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "channel",
			Label:    "Channel",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "channel",
				},
			},
		},
	}
}

func (t *OnMessage) Setup(ctx core.TriggerContext) error {
	var metadata OnMessageMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	var config OnMessageConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	channel, err := t.validateChannel(config, metadata)
	if err != nil {
		return fmt.Errorf("failed to validate channel: %w", err)
	}

	subscriptionID, err := t.subscribe(ctx, metadata)
	if err != nil {
		return fmt.Errorf("failed to subscribe to message events: %w", err)
	}

	return ctx.Metadata.Set(OnMessageMetadata{
		AppSubscriptionID: subscriptionID,
		Channel:           channel,
	})
}

func (t *OnMessage) subscribe(ctx core.TriggerContext, metadata OnMessageMetadata) (*string, error) {
	if metadata.AppSubscriptionID != nil {
		logrus.Infof("using existing subscription %s", *metadata.AppSubscriptionID)
		return metadata.AppSubscriptionID, nil
	}

	logrus.Infof("creating new subscription")

	subscriptionID, err := ctx.Integration.Subscribe(SubscriptionConfiguration{
		EventTypes: []string{"message"},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to message events: %w", err)
	}

	s := subscriptionID.String()
	return &s, nil
}

func (t *OnMessage) validateChannel(config OnMessageConfiguration, metadata OnMessageMetadata) (*ChannelMetadata, error) {
	if config.Channel == "" {
		return nil, nil
	}

	if metadata.Channel != nil && config.Channel == metadata.Channel.ID {
		return metadata.Channel, nil
	}

	return &ChannelMetadata{
		ID:   config.Channel,
		Name: config.Channel,
	}, nil
}

func (t *OnMessage) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	config := OnMessageConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	message, ok := ctx.Message.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected message type: %T", ctx.Message)
	}

	// Filter by channel if configured
	if config.Channel != "" {
		conversation, ok := message["conversation"].(map[string]any)
		if !ok {
			ctx.Logger.Infof("message has no conversation info, ignoring")
			return nil
		}

		conversationID, _ := conversation["id"].(string)
		if conversationID != "" && !channelMatches(config.Channel, conversationID) {
			ctx.Logger.Infof("message channel %s does not match configured channel %s, ignoring", conversationID, config.Channel)
			return nil
		}
	}

	return ctx.Events.Emit("teams.channel.message", ctx.Message)
}

func (t *OnMessage) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnMessage) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnMessage) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (t *OnMessage) Cleanup(ctx core.TriggerContext) error {
	return nil
}
