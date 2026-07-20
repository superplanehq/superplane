package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const discordAPIBase = "https://discord.com/api/v10"

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

// createFormFile is like multipart.Writer.CreateFormFile but lets the caller
// set the part's Content-Type. The stdlib helper hardcodes
// application/octet-stream, which loses the real media type of the attachment.
func createFormFile(w *multipart.Writer, field, filename, contentType string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
		quoteEscaper.Replace(field), quoteEscaper.Replace(filename)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	h.Set("Content-Type", contentType)
	return w.CreatePart(h)
}

type Client struct {
	BotToken string
}

func NewClient(ctx core.IntegrationContext) (*Client, error) {
	botToken, err := ctx.GetConfig("botToken")
	if err != nil {
		return nil, fmt.Errorf("failed to get bot token: %w", err)
	}

	token := string(botToken)
	if token == "" {
		return nil, fmt.Errorf("bot token is required")
	}

	return &Client{
		BotToken: token,
	}, nil
}

// User represents a Discord user object
type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Bot           bool   `json:"bot,omitempty"`
}

// Guild represents a Discord guild (server) object
type Guild struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Channel represents a Discord channel object
type Channel struct {
	ID      string `json:"id"`
	Type    int    `json:"type"`
	GuildID string `json:"guild_id,omitempty"`
	Name    string `json:"name,omitempty"`
}

// GetCurrentUser retrieves the current bot user
func (c *Client) GetCurrentUser() (*User, error) {
	responseBody, err := c.doRequest(http.MethodGet, "/users/@me", nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(responseBody, &user); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &user, nil
}

// GetCurrentUserGuilds retrieves the guilds the bot is a member of
func (c *Client) GetCurrentUserGuilds() ([]Guild, error) {
	responseBody, err := c.doRequest(http.MethodGet, "/users/@me/guilds", nil)
	if err != nil {
		return nil, err
	}

	var guilds []Guild
	if err := json.Unmarshal(responseBody, &guilds); err != nil {
		return nil, fmt.Errorf("failed to decode guilds: %w", err)
	}

	return guilds, nil
}

// GetGuildChannels retrieves the channels in a guild
func (c *Client) GetGuildChannels(guildID string) ([]Channel, error) {
	responseBody, err := c.doRequest(http.MethodGet, fmt.Sprintf("/guilds/%s/channels", guildID), nil)
	if err != nil {
		return nil, err
	}

	var channels []Channel
	if err := json.Unmarshal(responseBody, &channels); err != nil {
		return nil, fmt.Errorf("failed to decode channels: %w", err)
	}

	return channels, nil
}

// GetChannel retrieves a channel by ID
func (c *Client) GetChannel(channelID string) (*Channel, error) {
	responseBody, err := c.doRequest(http.MethodGet, fmt.Sprintf("/channels/%s", channelID), nil)
	if err != nil {
		return nil, err
	}

	var channel Channel
	if err := json.Unmarshal(responseBody, &channel); err != nil {
		return nil, fmt.Errorf("failed to decode channel: %w", err)
	}

	return &channel, nil
}

// Embed represents a Discord message embed
type Embed struct {
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	URL         string       `json:"url,omitempty"`
	Color       int          `json:"color,omitempty"`
	Timestamp   string       `json:"timestamp,omitempty"`
	Footer      *EmbedFooter `json:"footer,omitempty"`
	Author      *EmbedAuthor `json:"author,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
	Thumbnail   *EmbedMedia  `json:"thumbnail,omitempty"`
	Image       *EmbedMedia  `json:"image,omitempty"`
}

type EmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

type EmbedAuthor struct {
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type EmbedMedia struct {
	URL string `json:"url"`
}

// CreateMessageRequest represents the request body for creating a message
type CreateMessageRequest struct {
	Content string  `json:"content,omitempty"`
	Embeds  []Embed `json:"embeds,omitempty"`
}

// Message represents a Discord message object
type Message struct {
	ID        string  `json:"id"`
	Type      int     `json:"type"`
	Content   string  `json:"content"`
	ChannelID string  `json:"channel_id"`
	GuildID   string  `json:"guild_id,omitempty"`
	Author    User    `json:"author"`
	Timestamp string  `json:"timestamp"`
	Embeds    []Embed `json:"embeds,omitempty"`
	Mentions  []User  `json:"mentions,omitempty"`
}

const (
	// maxMessageFiles is Discord's attachment cap per message.
	maxMessageFiles = 10
	// maxMessageFileSize is Discord's default upload limit per file.
	maxMessageFileSize = 8 * 1024 * 1024
)

// MessageFile is a file attachment for a channel message.
type MessageFile struct {
	Name    string
	Content []byte
	// ContentType is the attachment's media type. When empty the multipart
	// helper falls back to application/octet-stream.
	ContentType string
}

// FetchFile downloads a file from a URL (e.g. a presigned artifact link from
// a Cursor agent run) so it can be attached to a message. The request goes
// through the workflow HTTP context so the platform's SSRF policy (blocked
// hosts, private IP ranges, redirect checks, response limits) applies to
// user-supplied URLs. The per-file size limit is enforced while reading.
func (c *Client) FetchFile(httpCtx core.HTTPContext, fileURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := httpCtx.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("file fetch failed: status %d", resp.StatusCode)
	}

	content, err := io.ReadAll(io.LimitReader(resp.Body, maxMessageFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if len(content) > maxMessageFileSize {
		return nil, fmt.Errorf("file exceeds the %d byte attachment limit", maxMessageFileSize)
	}

	return content, nil
}

// CreateMessageWithFiles sends a channel message with file attachments using
// multipart/form-data: a payload_json part with the message body plus one
// files[i] part per attachment, as required by the Discord API.
func (c *Client) CreateMessageWithFiles(channelID string, req CreateMessageRequest, files []MessageFile) (*Message, error) {
	payloadJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("payload_json", string(payloadJSON)); err != nil {
		return nil, fmt.Errorf("failed to write payload_json: %w", err)
	}

	for i, file := range files {
		part, err := createFormFile(writer, fmt.Sprintf("files[%d]", i), file.Name, file.ContentType)
		if err != nil {
			return nil, fmt.Errorf("failed to create file part: %w", err)
		}
		if _, err := part.Write(file.Content); err != nil {
			return nil, fmt.Errorf("failed to write file content: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/channels/%s/messages", discordAPIBase, channelID), &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bot %s", c.BotToken))
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed: status %d, body: %s", resp.StatusCode, string(responseBody))
	}

	var message Message
	if err := json.Unmarshal(responseBody, &message); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &message, nil
}

// CreateMessage sends a message to a channel
func (c *Client) CreateMessage(channelID string, req CreateMessageRequest) (*Message, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	responseBody, err := c.doRequest(http.MethodPost, fmt.Sprintf("/channels/%s/messages", channelID), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var message Message
	if err := json.Unmarshal(responseBody, &message); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &message, nil
}

// GetChannelMessages retrieves recent messages from a channel.
func (c *Client) GetChannelMessages(channelID string, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	endpoint := fmt.Sprintf("/channels/%s/messages?%s", channelID, url.Values{
		"limit": []string{fmt.Sprintf("%d", limit)},
	}.Encode())

	responseBody, err := c.doRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var messages []Message
	if err := json.Unmarshal(responseBody, &messages); err != nil {
		return nil, fmt.Errorf("failed to decode channel messages: %w", err)
	}

	return messages, nil
}

// doRequest executes an HTTP request to the Discord API
func (c *Client) doRequest(method, endpoint string, body io.Reader) ([]byte, error) {
	url := discordAPIBase + endpoint

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", c.BotToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed: status %d, body: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}
