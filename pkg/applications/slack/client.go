package slack

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	BotToken string
}

func NewClient(botToken string) (*Client, error) {
	if botToken == "" {
		return nil, fmt.Errorf("bot token is required")
	}

	return &Client{
		BotToken: botToken,
	}, nil
}

type AuthTestResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func (c *Client) AuthTest() error {
	responseBody, err := c.execRequest(http.MethodPost, "https://slack.com/api/auth.test", nil)
	if err != nil {
		return err
	}

	var result AuthTestResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	if !result.OK {
		if result.Error != "" {
			return fmt.Errorf("slack auth test failed: %s", result.Error)
		}
		return fmt.Errorf("slack auth test failed")
	}

	return nil
}

func (c *Client) execRequest(method, URL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.BotToken))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	return responseBody, nil
}
