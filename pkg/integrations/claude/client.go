package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/superplanehq/superplane/pkg/core"
	"io"
	"net/http"
)

const defaultBaseURL = "https://api.anthropic.com/v1"

type Client struct {
	APIKey           string
	AnthropicVersion string
	BaseURL          string
	http             core.HTTPContext
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CreateMessageRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	System      string    `json:"system,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
}

type MessageContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type MessageUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type CreateMessageResponse struct {
	ID           string           `json:"id"`
	Type         string           `json:"type"`
	Role         string           `json:"role"`
	Content      []MessageContent `json:"content"`
	Model        string           `json:"model"`
	StopReason   string           `json:"stop_reason"`
	StopSequence string           `json:"stop_sequence,omitempty"`
	Usage        MessageUsage     `json:"usage"`
}

type ModelsResponse struct {
	Data []Model `json:"data"`
}

type Model struct {
	ID string `json:"id"`
}

type claudeErrorResponse struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, err
	}

	anthropicVersion, err := ctx.GetConfig("anthropicVersion")
	if err != nil {
		return nil, err
	}

	return &Client{
		APIKey:           string(apiKey),
		AnthropicVersion: string(anthropicVersion),
		BaseURL:          defaultBaseURL,
		http:             httpClient,
	}, nil
}

func (c *Client) Verify() error {
	_, err := c.execRequest(http.MethodGet, c.BaseURL+"/models", nil)
	return err
}

func (c *Client) ListModels() ([]Model, error) {
	responseBody, err := c.execRequest(http.MethodGet, c.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}

	var response ModelsResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal models response: %v", err)
	}

	return response.Data, nil
}

func (c *Client) CreateMessage(req CreateMessageRequest) (*CreateMessageResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, c.BaseURL+"/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	var response CreateMessageResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message response: %v", err)
	}

	return &response, nil
}

func (c *Client) execRequest(method, URL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", c.AnthropicVersion)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		var apiErr claudeErrorResponse
		var errorMessage string

		// Try to parse the official Anthropic error message
		if err := json.Unmarshal(responseBody, &apiErr); err == nil && apiErr.Error.Message != "" {
			errorMessage = apiErr.Error.Message
		} else {
			errorMessage = string(responseBody)
		}

		// Handle 401 specifically
		if res.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("Claude credentials are invalid or expired: %s", errorMessage)
		}

		return nil, fmt.Errorf("request failed (%d): %s", res.StatusCode, errorMessage)
	}
	return responseBody, nil
}
