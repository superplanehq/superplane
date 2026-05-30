package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
		serverURL: string(serverURL),
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

func (c *Client) ExecuteAction(actionName string, params map[string]any, input any) (*ExecuteResponse, error) {
	reqBody := ExecuteRequest{
		Parameters: params,
		Input:      input,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.serverURL+"/actions/"+actionName+"/execute", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)

	resp, err := c.httpDo(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute action: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return &ExecuteResponse{
			Success: false,
			Error:   fmt.Sprintf("plugin server returned status %d: %s", resp.StatusCode, string(respBody)),
		}, nil
	}

	var result ExecuteResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (c *Client) setAuth(req *http.Request) {
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
}
