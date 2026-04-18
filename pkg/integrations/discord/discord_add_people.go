package discord

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DiscordAddPeople struct{}

type DiscordAddPeopleConfiguration struct {
	Channel string `json:"channel" mapstructure:"channel"`
	UserIDs string `json:"userIds" mapstructure:"userIds"`
}

type DiscordAddPeopleMetadata struct {
	Channel *ChannelMetadata `json:"channel" mapstructure:"channel"`
}

func (c *DiscordAddPeople) Name() string {
	return "discord.addPeople"
}

func (c *DiscordAddPeople) Label() string {
	return "Add People"
}

func (c *DiscordAddPeople) Description() string {
	return "Grant selected Discord users access to a channel"
}

func (c *DiscordAddPeople) Documentation() string {
	return `The Add People component grants selected Discord users access to a Discord channel.

## Use Cases

- Add incident responders to a newly created incident channel
- Grant selected users access to a private operations channel
- Dynamically control access to workflow-created channels

## Configuration

- **Channel**: The Discord channel to update
- **User IDs**: Comma-separated Discord user IDs to grant access to

## Output

Returns the channel and the list of parsed user IDs.

## Notes

- Users should already be members of the Discord server
- This component is intended for guild text channels`
}

func (c *DiscordAddPeople) Icon() string {
	return "discord"
}

func (c *DiscordAddPeople) Color() string {
	return "gray"
}

func (c *DiscordAddPeople) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DiscordAddPeople) Actions() []core.Action {
	return []core.Action{}
}

func (c *DiscordAddPeople) Configuration() []configuration.Field {
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
			Description: "Discord channel to update",
		},
		{
			Name:        "userIds",
			Label:       "User IDs",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "Comma-separated Discord user IDs to grant access to",
		},
	}
}

func (c *DiscordAddPeople) Setup(ctx core.SetupContext) error {
	var config DiscordAddPeopleConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Channel == "" {
		return errors.New("channel is required")
	}

	if strings.TrimSpace(config.UserIDs) == "" {
		return errors.New("userIds is required")
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Discord client: %w", err)
	}

	channelInfo, err := client.GetChannel(config.Channel)
	if err != nil {
		return fmt.Errorf("channel validation failed: %w", err)
	}

	metadata := DiscordAddPeopleMetadata{
		Channel: &ChannelMetadata{
			ID:   channelInfo.ID,
			Name: channelInfo.Name,
		},
	}

	return ctx.Metadata.Set(metadata)
}

func (c *DiscordAddPeople) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DiscordAddPeople) Execute(ctx core.ExecutionContext) error {
	var config DiscordAddPeopleConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Channel == "" {
		return errors.New("channel is required")
	}

	if strings.TrimSpace(config.UserIDs) == "" {
		return errors.New("userIds is required")
	}

	userIDs := parseCommaSeparatedIDs(config.UserIDs)
	if len(userIDs) == 0 {
		return errors.New("no valid user IDs provided")
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Discord client: %w", err)
	}

	for _, userID := range userIDs {
		if err := client.AddMemberToChannel(config.Channel, userID); err != nil {
			return fmt.Errorf("failed to add user %s to channel %s: %w", userID, config.Channel, err)
		}
	}

	return nil
}

func parseCommaSeparatedIDs(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		id := strings.TrimSpace(p)
		if id == "" {
			continue
		}
		out = append(out, id)
	}

	return out
}

func (c *DiscordAddPeople) ExampleOutput() map[string]any {
	return map[string]any{
		"channelId": "123456789012345678",
		"userIds": []string{
			"111111111111111111",
			"222222222222222222",
		},
		"status": "ok",
	}
}

func (c *DiscordAddPeople) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DiscordAddPeople) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 0, nil, nil
}

func (c *DiscordAddPeople) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DiscordAddPeople) Cleanup(ctx core.SetupContext) error {
	return nil
}