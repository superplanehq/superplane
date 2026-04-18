package slack

import (
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnReactionAdded struct{}

type OnReactionAddedConfiguration struct {
	Channel  string `json:"channel" mapstructure:"channel"`
	Reaction string `json:"reaction" mapstructure:"reaction"`
}

type OnReactionAddedMetadata struct {
	Channel           *ChannelMetadata `json:"channel,omitempty" mapstructure:"channel,omitempty"`
	AppSubscriptionID *string          `json:"appSubscriptionID,omitempty" mapstructure:"appSubscriptionID,omitempty"`
}

func (t *OnReactionAdded) Name() string {
	return "slack.onReactionAdded"
}

func (t *OnReactionAdded) Label() string {
	return "On Reaction Added"
}

func (t *OnReactionAdded) Description() string {
	return "Listen to reactions added to messages"
}

func (t *OnReactionAdded) Documentation() string {
	return `The On Reaction Added trigger starts a workflow execution when a reaction is added to a message.

## Use Cases

- **Approval workflows**: Trigger workflows when a message gets a ✅ reaction
- **Team interactions**: React to message reactions
- **Automation**: Respond to specific reactions as triggers

## Configuration

- **Channel**: Optional channel filter - if specified, only reactions in this channel will trigger (leave empty to listen to all channels)
- **Reaction**: Optional reaction filter - if specified, only the specified reaction will trigger (leave empty to listen to all reactions)

## Event Data

Each reaction event includes:
- **event**: Event information including reaction, channel, and message timestamp
- **user**: User who added the reaction
- **channel**: Channel where the reaction occurred
- **reaction**: The reaction that was added

## Setup

This trigger automatically sets up a Slack event subscription when configured. The subscription is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnReactionAdded) Icon() string {
	return "slack"
}

func (t *OnReactionAdded) Color() string {
	return "gray"
}

func (t *OnReactionAdded) ExampleData() map[string]any {
	return map[string]any{
		"type":      "reaction_added",
		"channel":   "C01234567",
		"timestamp": "1234567890.123456",
		"reaction":  "thumbsup",
		"item": map[string]any{
			"type":    "message",
			"channel": "C01234567",
			"ts":      "1234567890.123456",
		},
		"user": "U01234567",
	}
}

func (t *OnReactionAdded) Configuration() []configuration.Field {
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
		{
			Name:     "reaction",
			Label:    "Reaction",
			Type:     configuration.FieldTypeString,
			Required: false,
		},
	}
}

func (t *OnReactionAdded) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (t *OnReactionAdded) Setup(ctx core.TriggerContext) error {
	var metadata OnReactionAddedMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	var config OnReactionAddedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	channel, err := t.validateChannel(ctx, config, metadata)
	if err != nil {
		return fmt.Errorf("failed to validate channel: %w", err)
	}

	subscriptionID, err := t.subscribe(ctx, metadata)
	if err != nil {
		return fmt.Errorf("failed to subscribe to reaction events: %w", err)
	}

	return ctx.Metadata.Set(OnReactionAddedMetadata{
		AppSubscriptionID: subscriptionID,
		Channel:           channel,
	})
}

func (t *OnReactionAdded) subscribe(ctx core.TriggerContext, metadata OnReactionAddedMetadata) (*string, error) {
	if metadata.AppSubscriptionID != nil {
		logrus.Infof("using existing subscription %s", *metadata.AppSubscriptionID)
		return metadata.AppSubscriptionID, nil
	}

	logrus.Infof("creating new subscription")

	subscriptionID, err := ctx.Integration.Subscribe(SubscriptionConfiguration{
		EventTypes: []string{"reaction_added"},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to reaction events: %w", err)
	}

	s := subscriptionID.String()
	return &s, nil
}

func (t *OnReactionAdded) validateChannel(ctx core.TriggerContext, config OnReactionAddedConfiguration, metadata OnReactionAddedMetadata) (*ChannelMetadata, error) {
	var channelInfo *ChannelInfo
	if config.Channel == "" {
		return nil, nil
	}

	if metadata.Channel != nil && config.Channel == metadata.Channel.ID {
		return metadata.Channel, nil
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create Slack client: %w", err)
	}

	channelInfo, err = client.GetChannelInfo(config.Channel)
	if err != nil {
		return nil, fmt.Errorf("channel validation failed: %w", err)
	}

	return &ChannelMetadata{
		ID:   channelInfo.ID,
		Name: channelInfo.Name,
	}, nil
}

func (t *OnReactionAdded) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnReactionAdded) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnReactionAdded) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	config := OnReactionAddedConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	event := ctx.Message.(map[string]any)

	reactionItem, ok := event["item"].(map[string]any)
	if !ok {
		return fmt.Errorf("item not found in reaction event")
	}

	channel, ok := reactionItem["channel"].(string)
	if !ok {
		return fmt.Errorf("channel not found in reaction event")
	}

	reaction, ok := event["reaction"].(string)
	if !ok {
		return fmt.Errorf("reaction not found in reaction event")
	}

	if config.Channel != "" && config.Channel != channel {
		ctx.Logger.Infof("reaction channel %s does not match configuration channel %s, ignoring", channel, config.Channel)
		return nil
	}

	if config.Reaction != "" && config.Reaction != reaction {
		ctx.Logger.Infof("reaction %s does not match configuration reaction %s, ignoring", reaction, config.Reaction)
		return nil
	}

	return ctx.Events.Emit("slack.reaction.added", ctx.Message)
}

func (t *OnReactionAdded) Cleanup(ctx core.TriggerContext) error {
	return nil
}
