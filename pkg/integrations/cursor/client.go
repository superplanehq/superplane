package cursor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const defaultBaseURL = "https://api.cursor.com"

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	var launchAgentKey, adminKey string
	if ctx.LegacySetup() {
		if b, err := ctx.GetConfig(SecretLaunchAgentKey); err == nil {
			launchAgentKey = string(b)
		}
		if b, err := ctx.GetConfig(SecretAdminKey); err == nil {
			adminKey = string(b)
		}
	} else {
		if v, err := ctx.Secrets().Get(SecretLaunchAgentKey); err == nil {
			launchAgentKey = v
		}
		if v, err := ctx.Secrets().Get(SecretAdminKey); err == nil {
			adminKey = v
		}
	}

	return &Client{
		LaunchAgentKey: launchAgentKey,
		AdminKey:       adminKey,
		BaseURL:        defaultBaseURL,
		http:           httpClient,
	}, nil
}

// verifyCursorCredentials checks Cursor API keys.
// verifyLaunch/verifyAdmin select which checks to run.
func verifyCursorCredentials(httpClient core.HTTPContext, launchAgentKey, adminKey string, verifyLaunch, verifyAdmin bool) error {
	c := &Client{
		LaunchAgentKey: strings.TrimSpace(launchAgentKey),
		AdminKey:       strings.TrimSpace(adminKey),
		BaseURL:        defaultBaseURL,
		http:           httpClient,
	}
	if verifyLaunch {
		if c.LaunchAgentKey == "" {
			return fmt.Errorf("cloud agent API key is required")
		}
		if err := c.VerifyLaunchAgent(); err != nil {
			return fmt.Errorf("cloud agent key verification failed: %w", err)
		}
	}
	if verifyAdmin {
		if c.AdminKey == "" {
			return fmt.Errorf("admin API key is required")
		}
		if err := c.VerifyAdmin(); err != nil {
			return fmt.Errorf("admin key verification failed: %w", err)
		}
	}
	return nil
}

type Client struct {
	LaunchAgentKey string
	AdminKey       string
	BaseURL        string
	http           core.HTTPContext
}

type cursorErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type UsageRequest struct {
	StartDate int64 `json:"startDate"`
	EndDate   int64 `json:"endDate"`
}

type UsageResponse map[string]any

type ModelsResponse struct {
	Models []string `json:"models"`
}

type ConversationMessage struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Text string `json:"text"`
}

type ConversationResponse struct {
	ID       string                `json:"id"`
	Messages []ConversationMessage `json:"messages"`
}

func (c *Client) ListModels() ([]string, error) {
	if c.LaunchAgentKey == "" {
		return nil, fmt.Errorf("Cloud Agent API key is not configured")
	}

	responseBody, err := c.execRequest(http.MethodGet, c.BaseURL+"/v0/models", nil, c.LaunchAgentKey)
	if err != nil {
		return nil, err
	}

	var response ModelsResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal models response: %w", err)
	}

	return response.Models, nil
}

func (c *Client) VerifyLaunchAgent() error {
	_, err := c.execRequest(http.MethodGet, c.BaseURL+"/v0/agents?limit=1", nil, c.LaunchAgentKey)
	return err
}

func (c *Client) VerifyAdmin() error {
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	req := UsageRequest{
		StartDate: startOfDay.Unix() * 1000,
		EndDate:   now.Unix() * 1000,
	}

	_, err := c.GetDailyUsage(req)
	return err
}

func (c *Client) GetDailyUsage(req UsageRequest) (*UsageResponse, error) {
	if c.AdminKey == "" {
		return nil, fmt.Errorf("Admin API key is not configured")
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal usage request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, c.BaseURL+"/teams/daily-usage-data", bytes.NewBuffer(reqBody), c.AdminKey)
	if err != nil {
		return nil, err
	}

	var response UsageResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal usage response: %v", err)
	}

	return &response, nil
}

func (c *Client) LaunchAgent(req launchAgentRequest) (*LaunchAgentResponse, error) {
	if c.LaunchAgentKey == "" {
		return nil, fmt.Errorf("Cloud Agent API key is not configured")
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal agent request: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, c.BaseURL+"/v0/agents", bytes.NewBuffer(reqBody), c.LaunchAgentKey)
	if err != nil {
		return nil, err
	}

	var response LaunchAgentResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent response: %w", err)
	}

	return &response, nil
}

func (c *Client) GetAgentStatus(agentID string) (*LaunchAgentResponse, error) {
	if c.LaunchAgentKey == "" {
		return nil, fmt.Errorf("Cloud Agent API key is not configured")
	}

	url := fmt.Sprintf("%s/v0/agents/%s", c.BaseURL, agentID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil, c.LaunchAgentKey)
	if err != nil {
		return nil, err
	}

	var response LaunchAgentResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent status response: %w", err)
	}

	return &response, nil
}

func (c *Client) CancelAgent(agentID string) error {
	if c.LaunchAgentKey == "" {
		return fmt.Errorf("Cloud Agent API key is not configured")
	}

	url := fmt.Sprintf("%s/v0/agents/%s/cancel", c.BaseURL, agentID)
	_, err := c.execRequest(http.MethodPost, url, nil, c.LaunchAgentKey)
	return err
}

func (c *Client) GetAgentConversation(agentID string) (*ConversationResponse, error) {
	if c.LaunchAgentKey == "" {
		return nil, fmt.Errorf("Cloud Agent API key is not configured")
	}

	url := fmt.Sprintf("%s/v0/agents/%s/conversation", c.BaseURL, agentID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil, c.LaunchAgentKey)
	if err != nil {
		return nil, err
	}

	var response ConversationResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conversation response: %w", err)
	}

	return &response, nil
}

func (c *Client) execRequest(method, URL string, body io.Reader, apiKey string) ([]byte, error) {
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

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
		var apiErr cursorErrorResponse
		var errorMessage string

		if err := json.Unmarshal(responseBody, &apiErr); err == nil {
			if apiErr.Message != "" {
				errorMessage = apiErr.Message
			} else if apiErr.Error != "" {
				errorMessage = apiErr.Error
			} else {
				errorMessage = string(responseBody)
			}
		} else {
			errorMessage = string(responseBody)
		}

		if res.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("Cursor credentials are invalid or expired: %s", errorMessage)
		}

		return nil, fmt.Errorf("request failed (%d): %s", res.StatusCode, errorMessage)
	}

	return responseBody, nil
}
