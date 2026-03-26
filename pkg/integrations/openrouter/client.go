package openrouter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const defaultBaseURL = "https://openrouter.ai/api/v1"

type Client struct {
	APIKey  string
	BaseURL string
	http    core.HTTPContext
}

// Chat message (OpenRouter/OpenAI Chat API style).
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionsRequest matches OpenRouter POST /chat/completions body.
type ChatCompletionsRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// ChatCompletionsResponse matches OpenRouter response (OpenAI-style).
type ChatCompletionsResponse struct {
	ID      string         `json:"id"`
	Model   string         `json:"model"`
	Choices []Choice       `json:"choices"`
	Usage   *ResponseUsage `json:"usage,omitempty"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type ResponseUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ModelsResponse for GET /models (OpenAI-style list).
type ModelsResponse struct {
	Data []Model `json:"data"`
}

type Model struct {
	ID string `json:"id"`
}

// CreditsResponse for GET /credits (requires management API key).
type CreditsResponse struct {
	Data struct {
		TotalCredits float64 `json:"total_credits"`
		TotalUsage   float64 `json:"total_usage"`
	} `json:"data"`
}

// KeyRateLimit is legacy rate limit info for a key (will always return -1).
type KeyRateLimit struct {
	Requests float64 `json:"requests"`
	Interval string  `json:"interval"`
	Note     string  `json:"note"`
}

// KeyResponse for GET /key - current API key details.
// See https://openrouter.ai/docs/api/api-reference/api-keys/get-current-key
type KeyResponse struct {
	Data struct {
		Label              string       `json:"label"`
		Limit              *float64     `json:"limit"`
		Usage              float64      `json:"usage"`
		UsageDaily         float64      `json:"usage_daily"`
		UsageWeekly        float64      `json:"usage_weekly"`
		UsageMonthly       float64      `json:"usage_monthly"`
		ByokUsage          float64      `json:"byok_usage"`
		ByokUsageDaily     float64      `json:"byok_usage_daily"`
		ByokUsageWeekly    float64      `json:"byok_usage_weekly"`
		ByokUsageMonthly   float64      `json:"byok_usage_monthly"`
		IsFreeTier         bool         `json:"is_free_tier"`
		IsManagementKey    bool         `json:"is_management_key"`
		IsProvisioningKey  bool         `json:"is_provisioning_key"`
		LimitRemaining     *float64     `json:"limit_remaining"`
		LimitReset         *string      `json:"limit_reset"`
		IncludeByokInLimit bool         `json:"include_byok_in_limit"`
		ExpiresAt          *string      `json:"expires_at"`
		RateLimit          KeyRateLimit `json:"rate_limit"`
	} `json:"data"`
}

type openRouterError struct {
	Error struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
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

	return &Client{
		APIKey:  string(apiKey),
		BaseURL: defaultBaseURL,
		http:    httpClient,
	}, nil
}

func (c *Client) Verify() error {
	_, err := c.doRequest(http.MethodGet, c.BaseURL+"/models", nil)
	return err
}

func (c *Client) ListModels() ([]Model, error) {
	body, err := c.doRequest(http.MethodGet, c.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}

	var resp ModelsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal models response: %w", err)
	}

	return resp.Data, nil
}

func (c *Client) ChatCompletions(req ChatCompletionsRequest) (*ChatCompletionsResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	body, err := c.doRequest(http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	var resp ChatCompletionsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chat completions response: %w", err)
	}

	return &resp, nil
}

// GetRemainingCredits returns total credits purchased and used. Requires a management API key.
// See https://openrouter.ai/docs/api/api-reference/credits/get-credits
func (c *Client) GetRemainingCredits() (*CreditsResponse, error) {
	body, err := c.doRequest(http.MethodGet, c.BaseURL+"/credits", nil)
	if err != nil {
		return nil, err
	}

	var resp CreditsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credits response: %w", err)
	}

	return &resp, nil
}

// GetCurrentKeyDetails returns information on the API key associated with the current authentication.
// See https://openrouter.ai/docs/api/api-reference/api-keys/get-current-key
func (c *Client) GetCurrentKeyDetails() (*KeyResponse, error) {
	body, err := c.doRequest(http.MethodGet, c.BaseURL+"/key", nil)
	if err != nil {
		return nil, err
	}

	var resp KeyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal key response: %w", err)
	}

	return &resp, nil
}

func (c *Client) doRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		var apiErr openRouterError
		msg := string(responseBody)
		if err := json.Unmarshal(responseBody, &apiErr); err == nil && apiErr.Error.Message != "" {
			msg = apiErr.Error.Message
		}
		if res.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("OpenRouter API key is invalid or expired: %s", msg)
		}
		return nil, fmt.Errorf("request failed (%d): %s", res.StatusCode, msg)
	}

	return responseBody, nil
}
