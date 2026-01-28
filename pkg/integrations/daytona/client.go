package daytona

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const defaultBaseURL = "https://app.daytona.io/api"

type Client struct {
	APIKey  string
	BaseURL string
	http    core.HTTPContext
}

func NewClient(httpClient core.HTTPContext, ctx core.AppInstallationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no app installation context")
	}

	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, err
	}

	baseURL := defaultBaseURL
	if customURL, err := ctx.GetConfig("baseURL"); err == nil && string(customURL) != "" {
		baseURL = string(customURL)
	}

	return &Client{
		APIKey:  string(apiKey),
		BaseURL: baseURL,
		http:    httpClient,
	}, nil
}

// Sandbox represents a Daytona sandbox environment
type Sandbox struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

// CreateSandboxRequest represents the request to create a sandbox
type CreateSandboxRequest struct {
	Snapshot         string            `json:"snapshot,omitempty"`
	Target           string            `json:"target,omitempty"`
	Env              map[string]string `json:"env,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	AutoStopInterval int               `json:"autoStopInterval,omitempty"`
}

// ExecuteCodeRequest represents the request to execute code in a sandbox
type ExecuteCodeRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
	Timeout  int    `json:"timeout,omitempty"`
}

// ExecuteCodeResponse represents the response from code execution
type ExecuteCodeResponse struct {
	ExitCode int    `json:"exitCode"`
	Result   string `json:"result"`
}

// ExecuteCommandRequest represents the request to execute a command in a sandbox
type ExecuteCommandRequest struct {
	Command string `json:"command"`
	Cwd     string `json:"cwd,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

// ExecuteCommandResponse represents the response from command execution
type ExecuteCommandResponse struct {
	ExitCode int    `json:"exitCode"`
	Result   string `json:"result"`
}

// Verify checks if the API key is valid by listing sandboxes
func (c *Client) Verify() error {
	_, err := c.execRequest(http.MethodGet, c.BaseURL+"/sandbox", nil)
	return err
}

// CreateSandbox creates a new sandbox environment
func (c *Client) CreateSandbox(req *CreateSandboxRequest) (*Sandbox, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, c.BaseURL+"/sandbox", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var sandbox Sandbox
	if err := json.Unmarshal(responseBody, &sandbox); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sandbox response: %v", err)
	}

	return &sandbox, nil
}

// ExecuteCode executes code in a sandbox (uses the execute command endpoint)
func (c *Client) ExecuteCode(sandboxID string, req *ExecuteCodeRequest) (*ExecuteCodeResponse, error) {
	// Convert code execution to a command based on language
	var command string
	switch req.Language {
	case "python":
		command = fmt.Sprintf("python3 -c %q", req.Code)
	case "javascript":
		command = fmt.Sprintf("node -e %q", req.Code)
	case "typescript":
		command = fmt.Sprintf("npx ts-node -e %q", req.Code)
	default:
		command = fmt.Sprintf("python3 -c %q", req.Code)
	}

	// Convert ms to seconds, rounding up to ensure sub-second timeouts get at least 1 second
	var timeoutSeconds int
	if req.Timeout > 0 {
		timeoutSeconds = (req.Timeout + 999) / 1000
	}

	cmdReq := &ExecuteCommandRequest{
		Command: command,
		Timeout: timeoutSeconds,
	}

	body, err := json.Marshal(cmdReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	url := fmt.Sprintf("%s/toolbox/%s/toolbox/process/execute", c.BaseURL, sandboxID)
	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response ExecuteCodeResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal execute code response: %v", err)
	}

	return &response, nil
}

// ExecuteCommand executes a shell command in a sandbox
func (c *Client) ExecuteCommand(sandboxID string, req *ExecuteCommandRequest) (*ExecuteCommandResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	url := fmt.Sprintf("%s/toolbox/%s/toolbox/process/execute", c.BaseURL, sandboxID)
	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response ExecuteCommandResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal execute command response: %v", err)
	}

	return &response, nil
}

// DeleteSandbox deletes a sandbox
func (c *Client) DeleteSandbox(sandboxID string, force bool) error {
	url := fmt.Sprintf("%s/sandbox/%s?force=%t", c.BaseURL, sandboxID, force)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	return err
}

// APIError represents an error response from the Daytona API
type APIError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// 204 No Content is valid for DELETE
	if res.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		// Try to parse error response for a cleaner message
		var apiErr APIError
		if json.Unmarshal(responseBody, &apiErr) == nil && apiErr.Message != "" {
			return nil, fmt.Errorf("API error (%d): %s", res.StatusCode, apiErr.Message)
		}
		return nil, fmt.Errorf("API error (%d)", res.StatusCode)
	}

	return responseBody, nil
}
