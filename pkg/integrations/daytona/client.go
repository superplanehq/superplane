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

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
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

// Snapshot represents a Daytona snapshot
type Snapshot struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PaginatedSnapshots represents a paginated list of snapshots
type PaginatedSnapshots struct {
	Items []Snapshot `json:"items"`
}

// SessionExecuteRequest represents the request to execute a command in a session
type SessionExecuteRequest struct {
	Command  string `json:"command"`
	RunAsync bool   `json:"runAsync"`
}

// SessionExecuteResponse represents the response from async session command execution
type SessionExecuteResponse struct {
	CmdID string `json:"cmdId"`
}

// SessionCommand represents a command within a session
type SessionCommand struct {
	CmdID    string `json:"cmdId"`
	Command  string `json:"command"`
	ExitCode *int   `json:"exitCode"`
}

// Session represents a Daytona session with its executed commands
type Session struct {
	SessionID string           `json:"sessionId"`
	Commands  []SessionCommand `json:"commands"`
}

// FindCommand returns the command with the given ID, or nil if not found.
func (s *Session) FindCommand(cmdID string) *SessionCommand {
	for i := range s.Commands {
		if s.Commands[i].CmdID == cmdID {
			return &s.Commands[i]
		}
	}
	return nil
}

// ExecuteCommandResponse represents the response from command execution
type ExecuteCommandResponse struct {
	ExitCode int    `json:"exitCode"`
	Result   string `json:"result"`
}

// ExecuteCodeResponse represents the response from code execution
type ExecuteCodeResponse struct {
	ExitCode int    `json:"exitCode"`
	Result   string `json:"result"`
}

// ListSnapshots lists available snapshots
func (c *Client) ListSnapshots() ([]Snapshot, error) {
	responseBody, err := c.execRequest(http.MethodGet, c.BaseURL+"/snapshots", nil)
	if err != nil {
		return nil, err
	}

	var result PaginatedSnapshots
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshots response: %v", err)
	}

	return result.Items, nil
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

// APIConfig represents the relevant fields from the /api/config endpoint
type APIConfig struct {
	ProxyToolboxURL string `json:"proxyToolboxUrl"`
}

// FetchConfig fetches the API configuration from the /api/config endpoint
func (c *Client) FetchConfig() (*APIConfig, error) {
	responseBody, err := c.execRequest(http.MethodGet, c.BaseURL+"/config", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config: %v", err)
	}

	var config APIConfig
	if err := json.Unmarshal(responseBody, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config response: %v", err)
	}

	return &config, nil
}

// toolboxBaseURL returns the base toolbox URL for a given sandbox.
func (c *Client) toolboxBaseURL(sandboxID string) (string, error) {
	config, err := c.FetchConfig()
	if err != nil {
		return "", err
	}

	if config.ProxyToolboxURL == "" {
		return "", fmt.Errorf("proxyToolboxUrl not found in config")
	}

	return fmt.Sprintf("%s/%s", config.ProxyToolboxURL, sandboxID), nil
}

// CreateSession creates a new session in the sandbox.
func (c *Client) CreateSession(sandboxID, sessionID string) error {
	baseURL, err := c.toolboxBaseURL(sandboxID)
	if err != nil {
		return fmt.Errorf("failed to resolve toolbox URL: %v", err)
	}

	body, err := json.Marshal(map[string]string{"sessionId": sessionID})
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	_, err = c.execRequest(http.MethodPost, baseURL+"/process/session", bytes.NewReader(body))
	return err
}

// ExecuteSessionCommand executes a command asynchronously in a session.
// Returns the command ID which can be used to poll for completion.
func (c *Client) ExecuteSessionCommand(sandboxID, sessionID, command string) (*SessionExecuteResponse, error) {
	baseURL, err := c.toolboxBaseURL(sandboxID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve toolbox URL: %v", err)
	}

	reqBody := &SessionExecuteRequest{
		Command:  command,
		RunAsync: true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	url := fmt.Sprintf("%s/process/session/%s/exec", baseURL, sessionID)
	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response SessionExecuteResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session execute response: %v", err)
	}

	return &response, nil
}

// GetSession retrieves a session and its command statuses.
func (c *Client) GetSession(sandboxID, sessionID string) (*Session, error) {
	baseURL, err := c.toolboxBaseURL(sandboxID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve toolbox URL: %v", err)
	}

	url := fmt.Sprintf("%s/process/session/%s", baseURL, sessionID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(responseBody, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session response: %v", err)
	}

	return &session, nil
}

// GetSessionCommandLogs retrieves the logs for a specific command in a session.
func (c *Client) GetSessionCommandLogs(sandboxID, sessionID, commandID string) (string, error) {
	baseURL, err := c.toolboxBaseURL(sandboxID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve toolbox URL: %v", err)
	}

	url := fmt.Sprintf("%s/process/session/%s/command/%s/logs", baseURL, sessionID, commandID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	return string(responseBody), nil
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
		var apiErr APIError
		if json.Unmarshal(responseBody, &apiErr) == nil && apiErr.Message != "" {
			return nil, fmt.Errorf("API error (%d): %s", res.StatusCode, apiErr.Message)
		}
		return nil, fmt.Errorf("API error (%d)", res.StatusCode)
	}

	return responseBody, nil
}
