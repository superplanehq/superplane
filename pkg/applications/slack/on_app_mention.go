package slack

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnAppMention struct{}

type OnAppMentionConfiguration struct {
	Channel string `json:"channel" mapstructure:"channel"`
}

type AppMentionMetadata struct {
	Channel           *ChannelMetadata `json:"channel,omitempty" mapstructure:"channel,omitempty"`
	AppSubscriptionID *uuid.UUID       `json:"appSubscriptionID,omitempty" mapstructure:"appSubscriptionID,omitempty"`
}

func (t *OnAppMention) Name() string {
	return "slack.onAppMention"
}

func (t *OnAppMention) Label() string {
	return "On App Mention"
}

func (t *OnAppMention) Description() string {
	return "Listen to messages mentioning the Slack App"
}

func (t *OnAppMention) Icon() string {
	return "slack"
}

func (t *OnAppMention) Color() string {
	return "gray"
}

func (t *OnAppMention) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "channel",
			Label:    "Channel",
			Type:     configuration.FieldTypeString,
			Required: false,
		},
	}
}

func (t *OnAppMention) Setup(ctx core.TriggerContext) error {
	//
	// If subscription ID is already set, nothing to do.
	//
	var metadata AppMentionMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.AppSubscriptionID != nil && metadata.Channel != nil {
		return nil
	}

	//
	// Validate channel configuration
	//
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

	subscriptionID, err := ctx.AppInstallation.Subscribe(SubscriptionConfiguration{
		EventTypes: []string{"app_mention"},
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe to app events: %w", err)
	}

	return ctx.Metadata.Set(AppMentionMetadata{
		AppSubscriptionID: subscriptionID,
		Channel: &ChannelMetadata{
			ID:   channelInfo.ID,
			Name: channelInfo.Name,
		},
	})
}

func (t *OnAppMention) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnAppMention) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnAppMention) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (t *OnAppMention) OnAppMessage(ctx core.AppMessageContext) error {
	config := OnAppMentionConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	ctx.Logger.Infof("configuration: %+v", config)

	message := ctx.Message.(map[string]any)
	channel := message["channel"].(string)

	//
	// If channel configuration is set and does not match the message channel, ignore the message.
	//
	if config.Channel != "" && config.Channel != channel {
		ctx.Logger.Infof("message channel %s does not match configuration channel %s, ignoring", channel, config.Channel)
		return nil
	}

	//
	// Othewise, emit message.
	//
	return ctx.Events.Emit("slack.app.mention", ctx.Message)
}
