package runagent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// EnvironmentNetworking configures a cloud environment's outbound network policy.
type EnvironmentNetworking struct {
	Type                 string   `json:"type"` // "unrestricted" or "limited"
	AllowedHosts         []string `json:"allowed_hosts,omitempty"`
	AllowPackageManagers *bool    `json:"allow_package_managers,omitempty"`
	AllowMCPServers      *bool    `json:"allow_mcp_servers,omitempty"`
}

// EnvironmentConfig is the config object for a cloud environment.
type EnvironmentConfig struct {
	Type       string                `json:"type"` // "cloud"
	Networking EnvironmentNetworking `json:"networking"`
	Packages   map[string][]string   `json:"packages,omitempty"`
}

type createEnvironmentBody struct {
	Name   string            `json:"name"`
	Config EnvironmentConfig `json:"config"`
}

// CreateEnvironment provisions a cloud environment and returns its ID.
func (c *Client) CreateEnvironment(name string, config EnvironmentConfig) (string, error) {
	body := createEnvironmentBody{Name: name, Config: config}
	b, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal environment request: %w", err)
	}
	responseBody, err := c.execRequestWithBeta(http.MethodPost, c.BaseURL+"/environments", bytes.NewBuffer(b), anthropicBetaManagedAgents)
	if err != nil {
		return "", err
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", fmt.Errorf("decode environment response: %w", err)
	}
	if result.ID == "" {
		return "", fmt.Errorf("environment creation returned empty ID")
	}
	return result.ID, nil
}

// DeleteEnvironment removes an environment (only succeeds if no session references it).
func (c *Client) DeleteEnvironment(environmentID string) error {
	if environmentID == "" {
		return nil
	}
	_, err := c.execRequestWithBeta(http.MethodDelete, c.BaseURL+"/environments/"+url.PathEscape(environmentID), nil, anthropicBetaManagedAgents)
	return err
}

// CreateAgentRequest describes a managed agent to create.
type CreateAgentRequest struct {
	Name        string
	Model       string
	System      string
	ToolsetType string // defaults to the agent toolset when empty
}

type createAgentBody struct {
	Name   string           `json:"name"`
	Model  string           `json:"model"`
	System string           `json:"system,omitempty"`
	Tools  []map[string]any `json:"tools"`
}

const defaultAgentToolset = "agent_toolset_20260401"

// CreateAgent creates a managed agent (model + system prompt + built-in toolset)
// and returns its ID.
func (c *Client) CreateAgent(req CreateAgentRequest) (string, error) {
	toolset := req.ToolsetType
	if toolset == "" {
		toolset = defaultAgentToolset
	}
	body := createAgentBody{
		Name:   req.Name,
		Model:  req.Model,
		System: req.System,
		Tools:  []map[string]any{{"type": toolset}},
	}
	b, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal agent request: %w", err)
	}
	responseBody, err := c.execRequestWithBeta(http.MethodPost, c.BaseURL+"/agents", bytes.NewBuffer(b), anthropicBetaManagedAgents)
	if err != nil {
		return "", err
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", fmt.Errorf("decode agent response: %w", err)
	}
	if result.ID == "" {
		return "", fmt.Errorf("agent creation returned empty ID")
	}
	return result.ID, nil
}

// ArchiveAgent archives an agent so it is no longer usable for new sessions.
func (c *Client) ArchiveAgent(agentID string) error {
	if agentID == "" {
		return nil
	}
	_, err := c.execRequestWithBeta(http.MethodPost, c.BaseURL+"/agents/"+url.PathEscape(agentID)+"/archive", nil, anthropicBetaManagedAgents)
	return err
}
