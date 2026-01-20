package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	WebhookURL string
	WebhookID  string
	Token      string
}

func NewClient(ctx core.AppInstallationContext) (*Client, error) {
	webhookURL, err := ctx.GetConfig("webhookUrl")
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook URL: %w", err)
	}

	url := string(webhookURL)
	if url == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	id, token, err := parseWebhookURL(url)
	if err != nil {
		return nil, err
	}

	return &Client{
		WebhookURL: url,
		WebhookID:  id,
		Token:      token,
	}, nil
}

// parseWebhookURL extracts webhook ID and token from Discord webhook URL
// Format: https://discord.com/api/webhooks/{id}/{token}
func parseWebhookURL(webhookURL string) (string, string, error) {
	pattern := regexp.MustCompile(`/webhooks/(\d+)/([\w-]+)$`)
	matches := pattern.FindStringSubmatch(webhookURL)
	if len(matches) < 3 {
		return "", "", fmt.Errorf("invalid webhook URL format")
	}
	return matches[1], matches[2], nil
}

// WebhookInfo represents the Discord webhook object
type WebhookInfo struct {
	ID        string `json:"id"`
	Type      int    `json:"type"`
	GuildID   string `json:"guild_id,omitempty"`
	ChannelID string `json:"channel_id"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar,omitempty"`
	Token     string `json:"token,omitempty"`
}

// GetWebhook retrieves information about the webhook
func (c *Client) GetWebhook() (*WebhookInfo, error) {
	resp, err := http.Get(c.WebhookURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get webhook info: status %d, body: %s", resp.StatusCode, string(body))
	}

	var info WebhookInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode webhook info: %w", err)
	}

	return &info, nil
}

// Embed represents a Discord message embed
type Embed struct {
	Title       string        `json:"title,omitempty"`
	Description string        `json:"description,omitempty"`
	URL         string        `json:"url,omitempty"`
	Color       int           `json:"color,omitempty"`
	Timestamp   string        `json:"timestamp,omitempty"`
	Footer      *EmbedFooter  `json:"footer,omitempty"`
	Author      *EmbedAuthor  `json:"author,omitempty"`
	Fields      []EmbedField  `json:"fields,omitempty"`
	Thumbnail   *EmbedMedia   `json:"thumbnail,omitempty"`
	Image       *EmbedMedia   `json:"image,omitempty"`
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

// ExecuteWebhookRequest represents the request body for executing a webhook
type ExecuteWebhookRequest struct {
	Content   string  `json:"content,omitempty"`
	Username  string  `json:"username,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	Embeds    []Embed `json:"embeds,omitempty"`
}

// ExecuteWebhookResponse represents the response from executing a webhook with wait=true
type ExecuteWebhookResponse struct {
	ID        string `json:"id"`
	Type      int    `json:"type"`
	Content   string `json:"content"`
	ChannelID string `json:"channel_id"`
	Author    struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Bot      bool   `json:"bot"`
	} `json:"author"`
	Timestamp string  `json:"timestamp"`
	Embeds    []Embed `json:"embeds,omitempty"`
}

// ExecuteWebhook sends a message through the webhook
func (c *Client) ExecuteWebhook(req ExecuteWebhookRequest) (*ExecuteWebhookResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use ?wait=true to get the message object in response
	url := c.WebhookURL + "?wait=true"

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute webhook: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("webhook execution failed: status %d, body: %s", resp.StatusCode, string(responseBody))
	}

	var result ExecuteWebhookResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}
