package terraform

import (
	"encoding/json"
	"fmt"
	"io"
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
	var orgs []Organization
	page := 1

	for {
		path := fmt.Sprintf("/api/v2/organizations?page[number]=%d&page[size]=100", page)
		req, err := c.newRequest(http.MethodGet, path, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create organizations request: %w", err)
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch organizations: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to list organizations, expected 200 OK got %d", resp.StatusCode)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read organizations response: %w", err)
		}

		var payload struct {
			Data []struct {
				ID         string `json:"id"`
				Attributes struct {
					Name string `json:"name"`
				} `json:"attributes"`
			} `json:"data"`
			Meta struct {
				Pagination struct {
					NextPage *int `json:"next-page"`
				} `json:"pagination"`
			} `json:"meta"`
		}

		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			return nil, fmt.Errorf("failed to decode organizations: %w", err)
		}

		for _, item := range payload.Data {
			orgs = append(orgs, Organization{
				ID:   item.ID,
				Name: item.Attributes.Name,
			})
		}

		if payload.Meta.Pagination.NextPage == nil {
			break
		}
		page = *payload.Meta.Pagination.NextPage
	}

	return orgs, nil
}

func (c *Client) ListWorkspaces(organizationName string) ([]Workspace, error) {
	var workspaces []Workspace
	page := 1

	for {
		path := fmt.Sprintf("/api/v2/organizations/%s/workspaces?page[number]=%d&page[size]=100", url.PathEscape(organizationName), page)
		req, err := c.newRequest(http.MethodGet, path, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create workspaces request: %w", err)
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch workspaces for %s: %w", organizationName, err)
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to list workspaces, expected 200 OK got %d", resp.StatusCode)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read workspaces response: %w", err)
		}

		var payload struct {
			Data []struct {
				ID         string `json:"id"`
				Attributes struct {
					Name string `json:"name"`
				} `json:"attributes"`
			} `json:"data"`
			Meta struct {
				Pagination struct {
					NextPage *int `json:"next-page"`
				} `json:"pagination"`
			} `json:"meta"`
		}

		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			return nil, fmt.Errorf("failed to decode workspaces: %w", err)
		}

		for _, item := range payload.Data {
			workspaces = append(workspaces, Workspace{
				ID:   item.ID,
				Name: item.Attributes.Name,
			})
		}

		if payload.Meta.Pagination.NextPage == nil {
			break
		}
		page = *payload.Meta.Pagination.NextPage
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
