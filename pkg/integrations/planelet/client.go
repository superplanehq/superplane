package planelet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	serverURL string
	authToken string
	httpDo    func(*http.Request) (*http.Response, error)
}

func NewClient(integration core.IntegrationContext) (*Client, error) {
	serverURL, err := integration.GetConfig("serverUrl")
	if err != nil || serverURL == nil {
		return nil, fmt.Errorf("serverUrl is required")
	}

	var authToken string
	token, err := integration.GetConfig("authToken")
	if err == nil && token != nil {
		authToken = string(token)
	}

	return &Client{
		serverURL: strings.TrimRight(string(serverURL), "/"),
		authToken: authToken,
		httpDo:    http.DefaultClient.Do,
	}, nil
}

func NewClientWithHTTP(integration core.IntegrationContext, httpCtx core.HTTPContext) (*Client, error) {
	client, err := NewClient(integration)
	if err != nil {
		return nil, err
	}
	client.httpDo = httpCtx.Do
	return client, nil
}

type ExecuteRequest struct {
	Parameters map[string]any `json:"parameters"`
	Input      any            `json:"input,omitempty"`
}

type ExecuteResponse struct {
	Success bool           `json:"success"`
	Data    map[string]any `json:"data,omitempty"`
	Error   string         `json:"error,omitempty"`
}

type SetupTriggerRequest struct {
	Parameters map[string]any       `json:"parameters"`
	Webhook    TriggerWebhookConfig `json:"webhook"`
}

type TriggerWebhookConfig struct {
	URL    string `json:"url"`
	Secret string `json:"secret,omitempty"`
}

type SetupTriggerResponse struct {
	Success  bool           `json:"success"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Error    string         `json:"error,omitempty"`
}

type CleanupTriggerRequest struct {
	Parameters map[string]any `json:"parameters"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type CleanupTriggerResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type HandleTriggerWebhookRequest struct {
	Parameters map[string]any          `json:"parameters"`
	Metadata   map[string]any          `json:"metadata,omitempty"`
	Request    ForwardedWebhookRequest `json:"request"`
}

type ForwardedWebhookRequest struct {
	Method        string              `json:"method"`
	Headers       map[string][]string `json:"headers"`
	Query         map[string][]string `json:"query,omitempty"`
	RawBodyBase64 string              `json:"rawBodyBase64"`
}

type HandleTriggerWebhookResponse struct {
	Success   bool                 `json:"success"`
	Emit      bool                 `json:"emit"`
	EventType string               `json:"eventType,omitempty"`
	Payload   any                  `json:"payload,omitempty"`
	Reason    string               `json:"reason,omitempty"`
	Response  *WebhookHTTPResponse `json:"response,omitempty"`
	Error     string               `json:"error,omitempty"`
	Status    int                  `json:"status,omitempty"`
}

type WebhookHTTPResponse struct {
	Status  int               `json:"status,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

func (c *Client) FetchManifest() (*Manifest, error) {
	req, err := http.NewRequest("GET", c.serverURL+"/manifest", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.httpDo(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

func (c *Client) ExecuteAction(actionID string, params map[string]any, input any) (*ExecuteResponse, error) {
	reqBody := ExecuteRequest{
		Parameters: params,
		Input:      input,
	}

	var result ExecuteResponse
	status, body, err := c.postJSON("/actions/"+url.PathEscape(actionID)+"/execute", reqBody, &result)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return &ExecuteResponse{
			Success: false,
			Error:   failureMessage(status, body, result.Error),
		}, nil
	}

	return &result, nil
}

func (c *Client) SetupTrigger(triggerID string, params map[string]any, webhookURL string, secret string) (*SetupTriggerResponse, error) {
	reqBody := SetupTriggerRequest{
		Parameters: params,
		Webhook: TriggerWebhookConfig{
			URL:    webhookURL,
			Secret: secret,
		},
	}

	var result SetupTriggerResponse
	status, body, err := c.postJSON("/triggers/"+url.PathEscape(triggerID)+"/setup", reqBody, &result)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return &SetupTriggerResponse{
			Success: false,
			Error:   failureMessage(status, body, result.Error),
		}, nil
	}

	return &result, nil
}

func (c *Client) CleanupTrigger(triggerID string, params map[string]any, metadata map[string]any) (*CleanupTriggerResponse, error) {
	reqBody := CleanupTriggerRequest{
		Parameters: params,
		Metadata:   metadata,
	}

	var result CleanupTriggerResponse
	status, body, err := c.postJSON("/triggers/"+url.PathEscape(triggerID)+"/cleanup", reqBody, &result)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return &CleanupTriggerResponse{
			Success: false,
			Error:   failureMessage(status, body, result.Error),
		}, nil
	}

	return &result, nil
}

func (c *Client) HandleTriggerWebhook(triggerID string, reqBody HandleTriggerWebhookRequest) (*HandleTriggerWebhookResponse, error) {
	var result HandleTriggerWebhookResponse
	status, body, err := c.postJSON("/triggers/"+url.PathEscape(triggerID)+"/webhook", reqBody, &result)
	if err != nil {
		return nil, err
	}

	if status >= 400 {
		return &HandleTriggerWebhookResponse{
			Success: false,
			Error:   failureMessage(status, body, result.Error),
			Status:  status,
		}, nil
	}

	return &result, nil
}

func (c *Client) postJSON(path string, reqBody any, result any) (int, []byte, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.serverURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)

	resp, err := c.httpDo(req)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to call Planelet server: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		_ = json.Unmarshal(respBody, result)
		return resp.StatusCode, respBody, nil
	}

	if len(respBody) == 0 {
		return resp.StatusCode, respBody, nil
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return resp.StatusCode, respBody, fmt.Errorf("failed to parse response: %w", err)
	}

	return resp.StatusCode, respBody, nil
}

func failureMessage(status int, body []byte, parsedError string) string {
	if parsedError != "" {
		return parsedError
	}

	return fmt.Sprintf("Planelet server returned status %d: %s", status, string(body))
}

func (c *Client) setAuth(req *http.Request) {
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
}
