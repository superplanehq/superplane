package terraform

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Workspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) ListOrganizations() ([]Organization, error) {
	req, err := c.newRequest(http.MethodGet, "/api/v2/organizations", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create organizations request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organizations: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list organizations, expected 200 OK got %d", resp.StatusCode)
	}

	var payload struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Name string `json:"name"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode organizations: %w", err)
	}

	var orgs []Organization
	for _, item := range payload.Data {
		orgs = append(orgs, Organization{
			ID:   item.ID,
			Name: item.Attributes.Name,
		})
	}
	return orgs, nil
}

func (c *Client) ListWorkspaces(organizationName string) ([]Workspace, error) {
	path := fmt.Sprintf("/api/v2/organizations/%s/workspaces", url.PathEscape(organizationName))
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspaces request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch workspaces for %s: %w", organizationName, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list workspaces, expected 200 OK got %d", resp.StatusCode)
	}

	var payload struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Name string `json:"name"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode workspaces: %w", err)
	}

	var workspaces []Workspace
	for _, item := range payload.Data {
		workspaces = append(workspaces, Workspace{
			ID:   item.ID,
			Name: item.Attributes.Name,
		})
	}
	return workspaces, nil
}

func (c *Client) ReadWorkspace(id string) (*WorkspacePayload, error) {
	path := fmt.Sprintf("/api/v2/workspaces/%s", url.PathEscape(id))
	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create read workspace request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspace %s: %w", id, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to read workspace %s: expected 200 OK got %d", id, resp.StatusCode)
	}

	var payload struct {
		Data WorkspacePayload `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode workspace: %w", err)
	}

	return &payload.Data, nil
}
