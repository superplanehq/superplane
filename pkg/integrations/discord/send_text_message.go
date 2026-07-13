package discord

import (
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type SendTextMessage struct{}

type SendTextMessageConfiguration struct {
	Channel          string   `json:"channel" mapstructure:"channel"`
	Content          string   `json:"content" mapstructure:"content"`
	EmbedTitle       string   `json:"embedTitle" mapstructure:"embedTitle"`
	EmbedDescription string   `json:"embedDescription" mapstructure:"embedDescription"`
	EmbedColor       string   `json:"embedColor" mapstructure:"embedColor"`
	EmbedURL         string   `json:"embedUrl" mapstructure:"embedUrl"`
	Files            []string `json:"files" mapstructure:"files"`
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

func (c *SendTextMessage) Documentation() string {
	return `The Send Text Message component sends a message to a Discord channel.

## Use Cases

- **Notifications**: Send notifications about workflow events or system status
- **Alerts**: Alert teams about important events or errors
- **Updates**: Provide status updates on long-running processes

## Configuration

- **Channel**: Select the Discord channel to send the message to
- **Content**: Plain text message content (max 2000 characters)
- **Embed Title**: Optional title for a rich embed
- **Embed Description**: Optional description for a rich embed
- **Embed Color**: Hex color code for the embed (e.g., #5865F2)
- **Embed URL**: Optional URL to link from the embed title
- **Files**: Optional files to attach. Each entry is either a public http(s) URL (downloaded and uploaded to Discord) or a ` + "`data:`" + ` URI carrying the content inline — e.g. an AI component artifact: ` + "`data:image/png;base64,{{ $['Text Prompt'].data.artifacts[0].content }}`" + `.

## Output

Returns metadata about the sent message including message ID, channel ID, and author information.

## Notes

- Either content, embed (title/description), or files must be provided
- Up to 10 files per message, 8 MiB each (Discord limits)
- The Discord bot must be installed and have permission to post to the selected channel
- Supports Discord's rich embed formatting for visually appealing messages`
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
			Type:     configuration.FieldTypeIntegrationResource,
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
		{
			Name:        "files",
			Label:       "Files",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "URLs of files to attach (e.g. an artifact link from Cursor's Download Artifact)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "File URL",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
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

	// At least content, embed, or a non-empty file entry must be provided
	hasContent := config.Content != ""
	hasEmbed := config.EmbedTitle != "" || config.EmbedDescription != ""

	if !hasContent && !hasEmbed && !hasAttachableFile(config.Files) {
		return fmt.Errorf("either content, embed (title/description), or files is required")
	}

	// Validate content length
	if len(config.Content) > 2000 {
		return fmt.Errorf("content exceeds maximum length of 2000 characters")
	}

	if err := validateFiles(config.Files); err != nil {
		return err
	}

	// Validate color format if provided
	if config.EmbedColor != "" {
		if _, err := parseHexColor(config.EmbedColor); err != nil {
			return fmt.Errorf("invalid embed color: %w", err)
		}
	}

	// Get channel info to store in metadata
	client, err := NewClient(ctx.Integration)
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

	client, err := NewClient(ctx.Integration)
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

	response, err := sendMessage(client, ctx.HTTP, config, req)
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

func (c *SendTextMessage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *SendTextMessage) Cancel(ctx core.ExecutionContext) error {
	return nil
}

// validateFiles checks the attachment URL list at config time. Entries with
// unresolved expressions are skipped since they only resolve at execution.
func validateFiles(files []string) error {
	if len(files) > maxMessageFiles {
		return fmt.Errorf("at most %d files can be attached to a message", maxMessageFiles)
	}

	for _, file := range files {
		if file == "" || isExpressionValue(file) {
			continue
		}
		if strings.HasPrefix(file, "data:") {
			if _, _, err := parseDataURI(file); err != nil {
				return fmt.Errorf("invalid data URI: %v", err)
			}
			continue
		}
		parsed, err := url.Parse(file)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return fmt.Errorf("invalid file URL %q: must be an http(s) URL or a data: URI", file)
		}
	}

	return nil
}

// parseDataURI decodes a data: URI (data:[<mediatype>][;base64],<data>) into
// its media type and content bytes. It lets inline payload data — like the
// artifacts AI components emit — be attached without a publicly fetchable URL.
func parseDataURI(uri string) (string, []byte, error) {
	rest, ok := strings.CutPrefix(uri, "data:")
	if !ok {
		return "", nil, fmt.Errorf("missing data: prefix")
	}
	meta, data, ok := strings.Cut(rest, ",")
	if !ok {
		return "", nil, fmt.Errorf("missing comma separator")
	}

	mediaType := meta
	isBase64 := false
	if encoded, found := strings.CutSuffix(meta, ";base64"); found {
		mediaType = encoded
		isBase64 = true
	}

	if isBase64 {
		content, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return "", nil, fmt.Errorf("invalid base64 content: %v", err)
		}
		return mediaType, content, nil
	}

	content, err := url.PathUnescape(data)
	if err != nil {
		return "", nil, fmt.Errorf("invalid percent-encoded content: %v", err)
	}
	return mediaType, []byte(content), nil
}

// dataURIFileName derives an attachment filename from a data URI's media type
// (e.g. image/png -> file-1.png).
func dataURIFileName(mediaType string, index int) string {
	name := fmt.Sprintf("file-%d", index+1)
	if exts, err := mime.ExtensionsByType(mediaType); err == nil && len(exts) > 0 {
		return name + exts[0]
	}
	return name
}

// hasAttachableFile reports whether the list has at least one non-empty entry.
func hasAttachableFile(files []string) bool {
	for _, file := range files {
		if file != "" {
			return true
		}
	}
	return false
}

// fileNameFromURL derives an attachment filename from the URL path, falling
// back to a positional name for URLs without one (e.g. bare presigned links).
func fileNameFromURL(fileURL string, index int) string {
	fallback := fmt.Sprintf("file-%d", index+1)
	parsed, err := url.Parse(fileURL)
	if err != nil {
		return fallback
	}
	name := path.Base(parsed.Path)
	if name == "" || name == "." || name == "/" {
		return fallback
	}
	return name
}

// sendMessage sends the message, fetching and attaching files when configured.
func sendMessage(client *Client, httpCtx core.HTTPContext, config SendTextMessageConfiguration, req CreateMessageRequest) (*Message, error) {
	if len(config.Files) > maxMessageFiles {
		return nil, fmt.Errorf("at most %d files can be attached to a message", maxMessageFiles)
	}

	files := make([]MessageFile, 0, len(config.Files))
	for i, fileURL := range config.Files {
		if fileURL == "" {
			continue
		}
		if strings.HasPrefix(fileURL, "data:") {
			mediaType, content, err := parseDataURI(fileURL)
			if err != nil {
				return nil, fmt.Errorf("invalid data URI in files[%d]: %v", i, err)
			}
			files = append(files, MessageFile{Name: dataURIFileName(mediaType, i), Content: content})
			continue
		}
		content, err := client.FetchFile(httpCtx, fileURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch file %q: %w", fileURL, err)
		}
		files = append(files, MessageFile{Name: fileNameFromURL(fileURL, i), Content: content})
	}

	if len(files) == 0 {
		// File entries can resolve to empty at execution; without content or an
		// embed there is nothing left to send and Discord would reject it.
		if req.Content == "" && len(req.Embeds) == 0 {
			return nil, fmt.Errorf("nothing to send: content, embed, or a non-empty file URL is required")
		}
		return client.CreateMessage(config.Channel, req)
	}

	return client.CreateMessageWithFiles(config.Channel, req, files)
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

func (c *SendTextMessage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *SendTextMessage) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *SendTextMessage) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
