package claude

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
)

// anthropicManagedAgentsBeta is the beta header required by the Managed Agents
// API (agents, environments, and their versions).
const anthropicManagedAgentsBeta = "managed-agents-2026-04-01"

// maxManagedAgentsPages caps forward pagination so a runaway next_page loop can
// never hang a resource listing.
const maxManagedAgentsPages = 20

// ManagedAgent is a subset of the agent resource returned by GET /v1/agents.
type ManagedAgent struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}

// ManagedEnvironment is a subset of the environment resource returned by
// GET /v1/environments.
type ManagedEnvironment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ManagedAgentVersion is a single historical version of an agent, returned by
// GET /v1/agents/{id}/versions.
type ManagedAgentVersion struct {
	Version int `json:"version"`
}

type managedAgentsResponse struct {
	Data     []ManagedAgent `json:"data"`
	NextPage string         `json:"next_page"`
}

type managedEnvironmentsResponse struct {
	Data     []ManagedEnvironment `json:"data"`
	NextPage string               `json:"next_page"`
}

type managedAgentVersionsResponse struct {
	Data     []ManagedAgentVersion `json:"data"`
	NextPage string                `json:"next_page"`
}

// ListManagedAgents lists the Managed Agents in the workspace, following
// next_page cursors.
func (c *Client) ListManagedAgents() ([]ManagedAgent, error) {
	var agents []ManagedAgent
	page := ""
	for range maxManagedAgentsPages {
		params := url.Values{}
		params.Set("limit", "100")
		if page != "" {
			params.Set("page", page)
		}

		body, err := c.execRequestWithBeta(http.MethodGet, c.BaseURL+"/agents?"+params.Encode(), nil, anthropicManagedAgentsBeta)
		if err != nil {
			return nil, err
		}

		var response managedAgentsResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal agents response: %v", err)
		}

		agents = append(agents, response.Data...)
		if response.NextPage == "" {
			break
		}
		page = response.NextPage
	}
	return agents, nil
}

// ListManagedEnvironments lists the Managed Agent environments in the
// workspace, following next_page cursors.
func (c *Client) ListManagedEnvironments() ([]ManagedEnvironment, error) {
	var environments []ManagedEnvironment
	page := ""
	for range maxManagedAgentsPages {
		params := url.Values{}
		params.Set("limit", "100")
		if page != "" {
			params.Set("page", page)
		}

		body, err := c.execRequestWithBeta(http.MethodGet, c.BaseURL+"/environments?"+params.Encode(), nil, anthropicManagedAgentsBeta)
		if err != nil {
			return nil, err
		}

		var response managedEnvironmentsResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal environments response: %v", err)
		}

		environments = append(environments, response.Data...)
		if response.NextPage == "" {
			break
		}
		page = response.NextPage
	}
	return environments, nil
}

// ListManagedAgentVersions lists an agent's versions, newest first.
func (c *Client) ListManagedAgentVersions(agentID string) ([]ManagedAgentVersion, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent id is required")
	}

	var versions []ManagedAgentVersion
	page := ""
	for range maxManagedAgentsPages {
		params := url.Values{}
		params.Set("limit", "100")
		if page != "" {
			params.Set("page", page)
		}

		body, err := c.execRequestWithBeta(http.MethodGet, c.BaseURL+"/agents/"+url.PathEscape(agentID)+"/versions?"+params.Encode(), nil, anthropicManagedAgentsBeta)
		if err != nil {
			return nil, err
		}

		var response managedAgentVersionsResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal agent versions response: %v", err)
		}

		versions = append(versions, response.Data...)
		if response.NextPage == "" {
			break
		}
		page = response.NextPage
	}

	// The API documents newest-first ordering, but sort defensively so the
	// caller can always treat index 0 as the latest version.
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}
