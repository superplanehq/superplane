package teams

import (
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// OnMention triggers when the bot is @mentioned in a Teams channel.
type OnMention struct{}

// OnMentionConfiguration defines the trigger's configurable fields.
type OnMentionConfiguration struct {
	Channel string `json:"channel" mapstructure:"channel"`
}

// OnMentionMetadata stores metadata after trigger setup.
type OnMentionMetadata struct {
	Channel           *ChannelMetadata `json:"channel,omitempty" mapstructure:"channel,omitempty"`
	AppSubscriptionID *string          `json:"appSubscriptionID,omitempty" mapstructure:"appSubscriptionID,omitempty"`
}

// ChannelMetadata stores channel identification information.
type ChannelMetadata struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
}

func (t *OnMention) Name() string {
	return "teams.onMention"
}

func (t *OnMention) Label() string {
	return "On Mention"
}

func (t *OnMention) Description() string {
	return "Listen to messages mentioning the Teams bot"
}

func (t *OnMention) Documentation() string {
	return `The On Mention trigger starts a workflow execution when the Teams bot is @mentioned in a channel message.

## Use Cases

- **Bot commands**: Process commands from Teams messages
- **Bot interactions**: Create interactive Teams bots
- **Team workflows**: Trigger workflows from Teams conversations
- **Notification processing**: Process and respond to mentions

## Configuration

- **Channel**: Optional channel filter — if specified, only mentions in this channel will trigger (leave empty to listen to all channels)

## Event Data

Each mention event includes:
- **text**: The message text containing the mention
- **from**: User who mentioned the bot (ID and name)
- **conversation**: Channel and team information
- **timestamp**: When the mention occurred
- **serviceUrl**: Bot Framework service URL for sending replies

## Setup

This trigger automatically sets up a subscription for bot mention events when configured. The subscription is managed by SuperPlane and will be cleaned up when the trigger is removed.

## Note

This trigger works with the default Bot Framework behavior — the bot receives messages where it is @mentioned without any additional permissions.`
}

func (t *OnMention) Icon() string {
	return "teams"
}

func (t *OnMention) Color() string {
	return "gray"
}

func (t *OnMention) Configuration() []configuration.Field {
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

func (t *OnMention) Setup(ctx core.TriggerContext) error {
	var metadata OnMentionMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	var config OnMentionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	channel, err := t.validateChannel(ctx, config, metadata)
	if err != nil {
		return fmt.Errorf("failed to validate channel: %w", err)
	}

	subscriptionID, err := t.subscribe(ctx, metadata)
	if err != nil {
		return fmt.Errorf("failed to subscribe to bot events: %w", err)
	}

	return ctx.Metadata.Set(OnMentionMetadata{
		AppSubscriptionID: subscriptionID,
		Channel:           channel,
	})
}

func (t *OnMention) subscribe(ctx core.TriggerContext, metadata OnMentionMetadata) (*string, error) {
	if metadata.AppSubscriptionID != nil {
		logrus.Infof("using existing subscription %s", *metadata.AppSubscriptionID)
		return metadata.AppSubscriptionID, nil
	}

	logrus.Infof("creating new subscription")

	subscriptionID, err := ctx.Integration.Subscribe(SubscriptionConfiguration{
		EventTypes: []string{"mention"},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to bot events: %w", err)
	}

	s := subscriptionID.String()
	return &s, nil
}

func (t *OnMention) validateChannel(ctx core.TriggerContext, config OnMentionConfiguration, metadata OnMentionMetadata) (*ChannelMetadata, error) {
	if config.Channel == "" {
		return nil, nil
	}

	if metadata.Channel != nil && config.Channel == metadata.Channel.ID {
		return metadata.Channel, nil
	}

	// For Teams, channel IDs are opaque strings from the Graph API.
	// We store the ID as-is since validation requires Graph API access
	// which may not be available at trigger setup time.
	return &ChannelMetadata{
		ID:   config.Channel,
		Name: config.Channel,
	}, nil
}

func (t *OnMention) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	config := OnMentionConfiguration{}
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

	return ctx.Events.Emit("teams.bot.mention", ctx.Message)
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

func (t *OnMention) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// channelMatches checks if a conversation ID matches a configured channel.
// Teams conversation IDs can have complex formats, so we check for containment.
func channelMatches(configured, actual string) bool {
	return configured == actual || fmt.Sprintf("19:%s", configured) == actual
}
