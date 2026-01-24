package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const discordAPIBase = "https://discord.com/api/v10"

type Client struct {
	BotToken string
}

func NewClient(ctx core.AppInstallationContext) (*Client, error) {
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
	Author    User    `json:"author"`
	Timestamp string  `json:"timestamp"`
	Embeds    []Embed `json:"embeds,omitempty"`
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
