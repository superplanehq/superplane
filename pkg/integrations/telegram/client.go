package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const telegramAPIBase = "https://api.telegram.org/bot"

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

type apiResponse struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
}

// GetMe retrieves information about the bot
func (c *Client) GetMe() (*User, error) {
	responseBody, err := c.doRequest(http.MethodGet, "/getMe", nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(responseBody, &user); err != nil {
		return nil, fmt.Errorf("failed to decode bot info: %w", err)
	}

	return &user, nil
}

// SetWebhook sets the webhook URL for receiving updates
func (c *Client) SetWebhook(url string) error {
	payload := map[string]string{
		"url": url,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	_, err = c.doRequest(http.MethodPost, "/setWebhook", bytes.NewReader(body))
	if err != nil {
		return err
	}

	return nil
}

type ChatDetail struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

func (c *Client) GetChat(chatID string) (*ChatDetail, error) {
	payload := map[string]string{
		"chat_id": chatID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	responseBody, err := c.doRequest(http.MethodPost, "/getChat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var chat ChatDetail
	if err := json.Unmarshal(responseBody, &chat); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &chat, nil
}

func ChatDisplayName(chat *ChatDetail) string {
	if chat.Title != "" {
		return chat.Title
	}
	if chat.Username != "" {
		return "@" + chat.Username
	}
	name := chat.FirstName
	if chat.LastName != "" {
		name += " " + chat.LastName
	}
	return name
}

// SendMessage sends a text message to a chat
func (c *Client) SendMessage(chatID string, text string, parseMode string) (*TelegramMessage, error) {
	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}

	if parseMode != "" {
		payload["parse_mode"] = parseMode
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	responseBody, err := c.doRequest(http.MethodPost, "/sendMessage", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var message TelegramMessage
	if err := json.Unmarshal(responseBody, &message); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &message, nil
}

// doRequest executes an HTTP request to the Telegram Bot API
func (c *Client) doRequest(method, endpoint string, body io.Reader) ([]byte, error) {
	url := telegramAPIBase + c.BotToken + endpoint

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

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

	var apiResp apiResponse
	if err := json.Unmarshal(responseBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("Telegram API error: %s", string(responseBody))
	}

	return apiResp.Result, nil
}
