package discord

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetLastMentionPayloadType = "discord.getLastMention.result"

const (
	GetLastMentionOutputChannelFound    = "found"
	GetLastMentionOutputChannelNotFound = "notFound"
)

type GetLastMention struct{}

type GetLastMentionConfiguration struct {
	Channel string `json:"channel" mapstructure:"channel"`
	Since   string `json:"since,omitempty" mapstructure:"since,omitempty"`
}

type GetLastMentionMetadata struct {
	Channel *ChannelMetadata `json:"channel,omitempty" mapstructure:"channel,omitempty"`
}

func (c *GetLastMention) Name() string {
	return "discord.getLastMention"
}

func (c *GetLastMention) Label() string {
	return "Get Last Mention"
}

func (c *GetLastMention) Description() string {
	return "Get the most recent message that mentions the Discord bot in a channel"
}

func (c *GetLastMention) Documentation() string {
	return `The Get Last Mention component fetches recent messages from a Discord channel and returns the latest one that mentions your bot.

## Use Cases

- **Command polling**: Retrieve the most recent mention command in a channel
- **Manual workflows**: Pull latest bot mention on demand in a workflow step
- **Mention auditing**: Inspect the latest mention payload before replying

## Configuration

- **Channel**: Discord channel to search for mentions
- **Since**: Optional date string lower-bound for mentions (only mentions at or after this time are considered). Supports expressions.
  Accepted formats include ISO 8601 (recommended) and Go's default timestamp format (e.g. 2026-03-16 04:17:08.750328135 +0000 UTC).

## Output

The payload includes:
- **channel_id**: Channel queried
- **mention**: Full message payload for the latest bot mention (when found)

Output channels:
- **found**: Emitted when a matching mention is found
- **notFound**: Emitted when no matching mention is found

## Notes

- Requires the bot permission **Read Message History** in the selected channel
- Only non-bot-authored messages are considered
- Datetimes without timezone in **Since** are interpreted as UTC`
}

func (c *GetLastMention) Icon() string {
	return "discord"
}

func (c *GetLastMention) Color() string {
	return "gray"
}

func (c *GetLastMention) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  GetLastMentionOutputChannelFound,
			Label: "Found",
		},
		{
			Name:  GetLastMentionOutputChannelNotFound,
			Label: "Not Found",
		},
	}
}

func (c *GetLastMention) Configuration() []configuration.Field {
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
			Description: "Discord channel to search for the latest bot mention",
		},
		{
			Name:        "since",
			Label:       "Since",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Only consider mentions created at or after this date string. Supports ISO 8601, timestamps and expressions.",
		},
	}
}

func (c *GetLastMention) Setup(ctx core.SetupContext) error {
	var config GetLastMentionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Channel == "" {
		return errors.New("channel is required")
	}

	if !isExpressionValue(config.Since) {
		_, err := parseOptionalSince(config.Since)
		if err != nil {
			return err
		}
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Discord client: %w", err)
	}

	channelInfo, err := client.GetChannel(config.Channel)
	if err != nil {
		return fmt.Errorf("channel validation failed: %w", err)
	}

	name := channelInfo.Name
	if name == "" {
		name = config.Channel
	}

	return ctx.Metadata.Set(GetLastMentionMetadata{
		Channel: &ChannelMetadata{
			ID:   channelInfo.ID,
			Name: name,
		},
	})
}

func (c *GetLastMention) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetLastMention) Execute(ctx core.ExecutionContext) error {
	var config GetLastMentionConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Channel == "" {
		return errors.New("channel is required")
	}

	since, err := parseOptionalSince(config.Since)
	if err != nil {
		return err
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	if metadata.BotID == "" && metadata.Username == "" {
		return fmt.Errorf("bot identity metadata is missing; reconnect the integration")
	}

	client, err := NewClient(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Discord client: %w", err)
	}

	messages, err := client.GetChannelMessages(config.Channel, 100)
	if err != nil {
		return fmt.Errorf("failed to fetch channel messages: %w", err)
	}

	output := map[string]any{
		"channel_id": config.Channel,
	}

	for _, message := range messages {
		if since != nil {
			messageTimestamp, err := time.Parse(time.RFC3339Nano, message.Timestamp)
			if err != nil {
				return fmt.Errorf("invalid Discord message timestamp %q: %w", message.Timestamp, err)
			}

			if messageTimestamp.Before(*since) {
				continue
			}
		}

		messageMap := discordMessageToMap(message)
		if isBotAuthor(messageMap) {
			continue
		}

		if messageMentionsBot(messageMap, metadata.BotID, metadata.Username) {
			output["mention"] = messageMap
			return ctx.ExecutionState.Emit(
				GetLastMentionOutputChannelFound,
				GetLastMentionPayloadType,
				[]any{output},
			)
		}
	}

	return ctx.ExecutionState.Emit(
		GetLastMentionOutputChannelNotFound,
		GetLastMentionPayloadType,
		[]any{output},
	)
}

func (c *GetLastMention) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetLastMention) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetLastMention) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetLastMention) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetLastMention) Cleanup(ctx core.SetupContext) error {
	return nil
}

func parseOptionalSince(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := parseDateTimeWithUTCDefault(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid since: %w", err)
	}

	return &parsed, nil
}

func isExpressionValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	return strings.Contains(trimmed, "{{") && strings.Contains(trimmed, "}}")
}

func parseDateTimeWithUTCDefault(value string) (time.Time, error) {
	formatsWithTimezone := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
	}

	for _, format := range formatsWithTimezone {
		parsed, err := time.Parse(format, value)
		if err == nil {
			return parsed, nil
		}
	}

	formatsWithoutTimezone := []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	}

	for _, format := range formatsWithoutTimezone {
		parsed, err := time.ParseInLocation(format, value, time.UTC)
		if err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf(
		"expected datetime in ISO 8601 (e.g. 2026-03-10T16:00:00Z) or Go timestamp format (e.g. 2026-03-16 04:17:08.750328135 +0000 UTC), got %q",
		value,
	)
}
