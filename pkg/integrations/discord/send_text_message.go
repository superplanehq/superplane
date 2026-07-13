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
	Channel          string `json:"channel" mapstructure:"channel"`
	Content          string `json:"content" mapstructure:"content"`
	EmbedTitle       string `json:"embedTitle" mapstructure:"embedTitle"`
	EmbedDescription string `json:"embedDescription" mapstructure:"embedDescription"`
	EmbedColor       string `json:"embedColor" mapstructure:"embedColor"`
	EmbedURL         string `json:"embedUrl" mapstructure:"embedUrl"`
	// Files mixes the structured object form and the legacy string form
	// (URL or data: URI), so it is decoded separately from the raw config.
	Files []FileAttachment `json:"files" mapstructure:"-"`
}

const (
	fileSourceURL     = "url"
	fileSourceContent = "content"

	fileEncodingText   = "text"
	fileEncodingBase64 = "base64"
)

// FileAttachment is one entry of the Files list: either a URL to download or
// inline content (e.g. an AI component artifact). Legacy string entries (URL
// or data: URI) are carried in Raw.
type FileAttachment struct {
	Source   string `json:"source" mapstructure:"source"`
	URL      string `json:"url" mapstructure:"url"`
	Content  string `json:"content" mapstructure:"content"`
	Encoding string `json:"encoding" mapstructure:"encoding"`
	MimeType string `json:"mimeType" mapstructure:"mimeType"`
	Filename string `json:"filename" mapstructure:"filename"`
	Raw      string `json:"-" mapstructure:"-"`
}

func (f FileAttachment) isEmpty() bool {
	return f.Raw == "" && f.URL == "" && f.Content == ""
}

// decodeFileAttachments accepts both entry shapes: structured objects and
// legacy strings (http(s) URL or data: URI).
func decodeFileAttachments(raw any) ([]FileAttachment, error) {
	items, ok := raw.([]any)
	if !ok {
		if raw == nil {
			return nil, nil
		}
		return nil, fmt.Errorf("files must be a list")
	}

	entries := make([]FileAttachment, 0, len(items))
	for i, item := range items {
		switch v := item.(type) {
		case string:
			entries = append(entries, FileAttachment{Raw: strings.TrimSpace(v)})
		case map[string]any:
			var entry FileAttachment
			if err := mapstructure.Decode(v, &entry); err != nil {
				return nil, fmt.Errorf("files[%d]: %v", i, err)
			}
			entries = append(entries, entry)
		default:
			return nil, fmt.Errorf("files[%d] must be a file entry", i)
		}
	}
	return entries, nil
}

func decodeSendTextMessageConfiguration(raw any) (SendTextMessageConfiguration, error) {
	var config SendTextMessageConfiguration
	if err := mapstructure.Decode(raw, &config); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}
	if m, ok := raw.(map[string]any); ok {
		files, err := decodeFileAttachments(m["files"])
		if err != nil {
			return config, err
		}
		config.Files = files
	}
	return config, nil
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
- **Files**: Optional files to attach. Each entry picks a **Source**:
  - **URL** — a public http(s) link; the file is downloaded and attached.
  - **Inline content** — the file content itself, e.g. an AI component artifact: set **Content** to ` + "`{{ $['Text Prompt'].data.artifacts[0].content }}`" + `, pick the **MIME Type**, and set **Encoding** to match the artifact's ` + "`encoding`" + ` field (text for plain text, base64 for binary files like images).

  An optional **Filename** names the attachment; legacy plain-string entries (URL or ` + "`data:`" + ` URI) keep working.

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
			Description: "Files to attach: download from a URL, or attach inline content such as an AI component artifact",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "File",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "source",
								Label:       "Source",
								Type:        configuration.FieldTypeSelect,
								Required:    true,
								Default:     fileSourceURL,
								Description: "Where the file comes from",
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "URL", Value: fileSourceURL},
											{Label: "Inline content", Value: fileSourceContent},
										},
									},
								},
							},
							{
								Name:        "url",
								Label:       "URL",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Placeholder: "https://example.com/report.pdf",
								Description: "Public http(s) URL; the file is downloaded and attached",
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "source", Values: []string{fileSourceURL}},
								},
							},
							{
								Name:        "content",
								Label:       "Content",
								Type:        configuration.FieldTypeText,
								Required:    false,
								Description: "File content, e.g. {{ $['Text Prompt'].data.artifacts[0].content }}",
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "source", Values: []string{fileSourceContent}},
								},
							},
							{
								Name:        "encoding",
								Label:       "Encoding",
								Type:        configuration.FieldTypeSelect,
								Required:    false,
								Default:     fileEncodingText,
								Description: "Match the artifact's encoding field: text for plain text, base64 for binary files",
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Text", Value: fileEncodingText},
											{Label: "Base64", Value: fileEncodingBase64},
										},
									},
								},
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "source", Values: []string{fileSourceContent}},
								},
							},
							{
								Name:        "mimeType",
								Label:       "MIME Type",
								Type:        configuration.FieldTypeSelect,
								Required:    false,
								Default:     "application/octet-stream",
								Description: "Used to name the attachment when no filename is set",
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "PNG image", Value: "image/png"},
											{Label: "JPEG image", Value: "image/jpeg"},
											{Label: "GIF image", Value: "image/gif"},
											{Label: "Plain text", Value: "text/plain"},
											{Label: "CSV", Value: "text/csv"},
											{Label: "Markdown", Value: "text/markdown"},
											{Label: "HTML", Value: "text/html"},
											{Label: "JSON", Value: "application/json"},
											{Label: "PDF", Value: "application/pdf"},
											{Label: "ZIP", Value: "application/zip"},
											{Label: "Binary", Value: "application/octet-stream"},
										},
									},
								},
								VisibilityConditions: []configuration.VisibilityCondition{
									{Field: "source", Values: []string{fileSourceContent}},
								},
							},
							{
								Name:        "filename",
								Label:       "Filename",
								Type:        configuration.FieldTypeString,
								Required:    false,
								Placeholder: "report.csv",
								Description: "Optional attachment name; defaults to a name derived from the URL or MIME type",
							},
						},
					},
				},
			},
		},
	}
}

func (c *SendTextMessage) Setup(ctx core.SetupContext) error {
	config, err := decodeSendTextMessageConfiguration(ctx.Configuration)
	if err != nil {
		return err
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
	config, err := decodeSendTextMessageConfiguration(ctx.Configuration)
	if err != nil {
		return err
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

// validateFiles checks the attachment list at config time. Values with
// unresolved expressions are skipped since they only resolve at execution.
func validateFiles(files []FileAttachment) error {
	if len(files) > maxMessageFiles {
		return fmt.Errorf("at most %d files can be attached to a message", maxMessageFiles)
	}

	for i, file := range files {
		if file.Raw != "" {
			if err := validateLegacyFileEntry(file.Raw); err != nil {
				return err
			}
			continue
		}

		switch file.Source {
		case "", fileSourceURL:
			if file.URL == "" || isExpressionValue(file.URL) {
				continue
			}
			parsed, err := url.Parse(file.URL)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
				return fmt.Errorf("files[%d]: invalid URL %q: must be an http(s) URL", i, file.URL)
			}
		case fileSourceContent:
			if file.Encoding != "" && file.Encoding != fileEncodingText && file.Encoding != fileEncodingBase64 {
				return fmt.Errorf("files[%d]: encoding must be %q or %q", i, fileEncodingText, fileEncodingBase64)
			}
		default:
			return fmt.Errorf("files[%d]: source must be %q or %q", i, fileSourceURL, fileSourceContent)
		}
	}

	return nil
}

func validateLegacyFileEntry(value string) error {
	if isExpressionValue(value) {
		return nil
	}
	if strings.HasPrefix(value, "data:") {
		if _, _, err := parseDataURI(value); err != nil {
			return fmt.Errorf("invalid data URI: %v", err)
		}
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("invalid file URL %q: must be an http(s) URL or a data: URI", value)
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
		// Tolerate whitespace around the payload, e.g. "base64, {{ expr }}".
		content, err := base64.StdEncoding.DecodeString(strings.TrimSpace(data))
		if err != nil {
			return "", nil, fmt.Errorf("invalid base64 content: %v", err)
		}
		return mediaType, content, nil
	}

	// Plain data is percent-decoded per the data: URI scheme, but expression
	// values paste raw content in, so undecodable data is kept as-is rather
	// than rejected (e.g. a CSV containing a literal "%").
	if content, err := url.PathUnescape(data); err == nil {
		return mediaType, []byte(content), nil
	}
	return mediaType, []byte(data), nil
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
func hasAttachableFile(files []FileAttachment) bool {
	for _, file := range files {
		if !file.isEmpty() {
			return true
		}
	}
	return false
}

// resolveFileAttachment turns one Files entry into attachment bytes: URL
// entries are downloaded, content entries are decoded inline, and legacy
// string entries keep their URL/data: URI behavior.
func resolveFileAttachment(client *Client, httpCtx core.HTTPContext, entry FileAttachment, index int) (MessageFile, error) {
	if entry.Raw != "" {
		return resolveLegacyFileEntry(client, httpCtx, entry.Raw, index)
	}

	if entry.Source == fileSourceContent {
		// Expression values often carry stray whitespace around the content.
		data := strings.TrimSpace(entry.Content)
		content := []byte(data)
		if entry.Encoding == fileEncodingBase64 {
			decoded, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				return MessageFile{}, fmt.Errorf("files[%d]: invalid base64 content: %v", index, err)
			}
			content = decoded
		}
		name := entry.Filename
		if name == "" {
			name = dataURIFileName(entry.MimeType, index)
		}
		return MessageFile{Name: name, Content: content}, nil
	}

	fileURL := strings.TrimSpace(entry.URL)
	if !strings.HasPrefix(fileURL, "http://") && !strings.HasPrefix(fileURL, "https://") {
		return MessageFile{}, fmt.Errorf("files[%d]: URL %q must be an http(s) URL; to attach inline content, set the entry's source to content", index, fileURL)
	}
	content, err := client.FetchFile(httpCtx, fileURL)
	if err != nil {
		return MessageFile{}, fmt.Errorf("failed to fetch file %q: %w", fileURL, err)
	}
	name := entry.Filename
	if name == "" {
		name = fileNameFromURL(fileURL, index)
	}
	return MessageFile{Name: name, Content: content}, nil
}

// resolveLegacyFileEntry handles the original string entry form: an http(s)
// URL to download or a data: URI carrying the content inline.
func resolveLegacyFileEntry(client *Client, httpCtx core.HTTPContext, value string, index int) (MessageFile, error) {
	if strings.HasPrefix(value, "data:") {
		mediaType, content, err := parseDataURI(value)
		if err != nil {
			return MessageFile{}, fmt.Errorf("invalid data URI in files[%d]: %v", index, err)
		}
		return MessageFile{Name: dataURIFileName(mediaType, index), Content: content}, nil
	}
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		// A schemeless entry is almost always raw file content pasted via
		// an expression; a fetch would only fail with a cryptic error.
		return MessageFile{}, fmt.Errorf("files[%d] is neither an http(s) URL nor a data: URI; to attach inline content, use a file entry with source set to content", index)
	}
	content, err := client.FetchFile(httpCtx, value)
	if err != nil {
		return MessageFile{}, fmt.Errorf("failed to fetch file %q: %w", value, err)
	}
	return MessageFile{Name: fileNameFromURL(value, index), Content: content}, nil
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
	for i, entry := range config.Files {
		if entry.isEmpty() {
			continue
		}
		file, err := resolveFileAttachment(client, httpCtx, entry, i)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
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
