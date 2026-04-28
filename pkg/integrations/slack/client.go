package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/httpx"
)

// slackHTTPClient is a package-level shared client. Transport is
// left nil so it falls back to http.DefaultTransport, which keeps
// the existing test helper (withDefaultTransport) working.
var slackHTTPClient = &http.Client{Timeout: 30 * time.Second}

// slackRetryConfig governs retry behavior for outbound Slack API
// calls. Tests may override this via the package-internal helpers
// in test_helpers_test.go.
var slackRetryConfig = httpx.DefaultConfig()

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

type AuthTestResponse struct {
	OK     bool   `json:"ok"`
	URL    string `json:"url"`
	Team   string `json:"team"`
	TeamID string `json:"team_id"`
	User   string `json:"user"`
	UserID string `json:"user_id"`
	BotID  string `json:"bot_id"`
	Error  string `json:"error,omitempty"`
}

func (c *Client) AuthTest() (*AuthTestResponse, error) {
	responseBody, err := c.execRequest(http.MethodPost, "https://slack.com/api/auth.test", nil)
	if err != nil {
		return nil, err
	}

	var result AuthTestResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if !result.OK {
		if result.Error != "" {
			return nil, fmt.Errorf("slack auth test failed: %s", result.Error)
		}
		return nil, fmt.Errorf("slack auth test failed")
	}

	return &result, nil
}

type ChannelInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ConversationsInfoResponse struct {
	OK      bool         `json:"ok"`
	Error   string       `json:"error,omitempty"`
	Channel *ChannelInfo `json:"channel,omitempty"`
}

func (c *Client) GetChannelInfo(channel string) (*ChannelInfo, error) {
	apiURL := "https://slack.com/api/conversations.info"
	params := url.Values{}
	params.Set("channel", channel)

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())
	responseBody, err := c.execRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}

	var result ConversationsInfoResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if !result.OK {
		if result.Error != "" {
			return nil, fmt.Errorf("failed to get channel info: %s", result.Error)
		}
		return nil, fmt.Errorf("failed to get channel info")
	}

	return result.Channel, nil
}

func (c *Client) ListChannels() ([]ChannelInfo, error) {
	responseBody, err := c.execRequest(http.MethodGet, "https://slack.com/api/conversations.list", nil)
	if err != nil {
		return nil, err
	}

	var result ConversationsListResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if !result.OK {
		if result.Error != "" {
			return nil, fmt.Errorf("failed to list channels: %s", result.Error)
		}
		return nil, fmt.Errorf("failed to list channels")
	}

	return result.Channels, nil
}

type ConversationsListResponse struct {
	OK       bool          `json:"ok"`
	Error    string        `json:"error,omitempty"`
	Channels []ChannelInfo `json:"channels,omitempty"`
}

type ChatPostMessageRequest struct {
	Channel         string        `json:"channel"`
	Text            string        `json:"text,omitempty"`
	Blocks          []interface{} `json:"blocks,omitempty"`
	ThreadTimestamp string        `json:"thread_ts,omitempty"`
}

type ChatPostMessageResponse struct {
	OK      bool           `json:"ok"`
	Error   string         `json:"error,omitempty"`
	TS      string         `json:"ts,omitempty"`
	Message map[string]any `json:"message,omitempty"`
}

func (c *Client) PostMessage(req ChatPostMessageRequest) (*ChatPostMessageResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, "https://slack.com/api/chat.postMessage", body)
	if err != nil {
		return nil, err
	}

	var result ChatPostMessageResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if !result.OK {
		if result.Error != "" {
			return nil, fmt.Errorf("failed to post message: %s", result.Error)
		}
		return nil, fmt.Errorf("failed to post message")
	}

	return &result, nil
}

// execRequest sends an authenticated request to the Slack API with
// retries on transient failures (5xx / 429 / network errors). The
// body argument is a marshaled payload; pass nil for GET requests.
//
// Callers do not currently have a context.Context to thread through
// (Slack-side callers are component Setup/Execute/etc., which do
// not expose one). Once the integration framework propagates ctx,
// switch this to take it as an argument.
func (c *Client) execRequest(method, URL string, body []byte) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.BotToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpx.Do(context.Background(), slackHTTPClient, req, slackRetryConfig)
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
