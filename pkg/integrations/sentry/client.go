package sentry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	BaseURL = "https://sentry.io"
)

type Client struct {
	httpCtx      core.HTTPContext
	organization string
	authToken    string
}

func NewClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	if integration == nil {
		return nil, fmt.Errorf("no integration context")
	}

	organization, err := integration.GetConfig("organization")
	if err != nil {
		return nil, err
	}

	authToken, err := integration.GetConfig("authToken")
	if err != nil {
		return nil, err
	}

	return &Client{
		httpCtx:      httpCtx,
		organization: string(organization),
		authToken:    string(authToken),
	}, nil
}

func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", BaseURL, path)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
	req.Header.Set("Content-Type", "application/json")

	return c.httpCtx.Do(req)
}

// Organization represents a Sentry organization
type Organization struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// GetOrganization retrieves the organization details to verify the connection
func (c *Client) GetOrganization() (*Organization, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/0/organizations/%s/", c.organization), nil)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error getting organization: status=%d body=%s", resp.StatusCode, string(body))
	}

	var org Organization
	err = json.NewDecoder(resp.Body).Decode(&org)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &org, nil
}

// APIProject represents a Sentry project from the API
type APIProject struct {
	ID           string `json:"id"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Organization struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
		Name string `json:"name"`
	} `json:"organization"`
}

// ListProjects retrieves all projects in the organization
func (c *Client) ListProjects() ([]APIProject, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/0/organizations/%s/projects/", c.organization), nil)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing projects: status=%d body=%s", resp.StatusCode, string(body))
	}

	var projects []APIProject
	err = json.NewDecoder(resp.Body).Decode(&projects)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return projects, nil
}

// GetProject retrieves a project by slug
func (c *Client) GetProject(projectSlug string) (*APIProject, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/0/projects/%s/%s/", c.organization, projectSlug), nil)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error getting project: status=%d body=%s", resp.StatusCode, string(body))
	}

	var project APIProject
	err = json.NewDecoder(resp.Body).Decode(&project)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &project, nil
}

// UpdateIssueRequest represents the request body for updating an issue
type UpdateIssueRequest struct {
	Status       string `json:"status,omitempty"`
	AssignedTo   string `json:"assignedTo,omitempty"`
	HasSeen      *bool  `json:"hasSeen,omitempty"`
	IsBookmarked *bool  `json:"isBookmarked,omitempty"`
}

// Issue represents a Sentry issue
type Issue struct {
	ID           string `json:"id"`
	ShortID      string `json:"shortId"`
	Title        string `json:"title"`
	Culprit      string `json:"culprit"`
	Level        string `json:"level"`
	Status       string `json:"status"`
	IsPublic     bool   `json:"isPublic"`
	IsBookmarked bool   `json:"isBookmarked"`
	HasSeen      bool   `json:"hasSeen"`
	Project      struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
		Name string `json:"name"`
	} `json:"project"`
	AssignedTo *struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"assignedTo,omitempty"`
	FirstSeen string `json:"firstSeen"`
	LastSeen  string `json:"lastSeen"`
	Count     string `json:"count"`
	UserCount int    `json:"userCount"`
	Permalink string `json:"permalink"`
}

// UpdateIssue updates an issue's status, assignment, etc.
func (c *Client) UpdateIssue(issueID string, request UpdateIssueRequest) (*Issue, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	resp, err := c.doRequest("PUT", fmt.Sprintf("/api/0/organizations/%s/issues/%s/", c.organization, issueID), strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error updating issue: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var issue Issue
	err = json.NewDecoder(resp.Body).Decode(&issue)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &issue, nil
}

// ServiceHook represents a Sentry service hook (webhook)
type ServiceHook struct {
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	Secret      string   `json:"secret"`
	Status      string   `json:"status"`
	Events      []string `json:"events"`
	DateCreated string   `json:"dateCreated"`
}

// CreateServiceHookRequest represents the request body for creating a service hook
type CreateServiceHookRequest struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

// CreateServiceHook creates a new service hook (webhook) for a project
// POST /api/0/projects/{organization_slug}/{project_slug}/hooks/
func (c *Client) CreateServiceHook(projectSlug string, webhookURL string, events []string) (*ServiceHook, error) {
	request := CreateServiceHookRequest{
		URL:    webhookURL,
		Events: events,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	resp, err := c.doRequest("POST", fmt.Sprintf("/api/0/projects/%s/%s/hooks/", c.organization, projectSlug), strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error creating service hook: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var hook ServiceHook
	err = json.NewDecoder(resp.Body).Decode(&hook)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &hook, nil
}

// ListServiceHooks lists all service hooks for a project
// GET /api/0/projects/{organization_slug}/{project_slug}/hooks/
func (c *Client) ListServiceHooks(projectSlug string) ([]ServiceHook, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/0/projects/%s/%s/hooks/", c.organization, projectSlug), nil)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing service hooks: status=%d body=%s", resp.StatusCode, string(body))
	}

	var hooks []ServiceHook
	err = json.NewDecoder(resp.Body).Decode(&hooks)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return hooks, nil
}

// GetServiceHook retrieves a service hook by ID
// GET /api/0/projects/{organization_slug}/{project_slug}/hooks/{hook_id}/
func (c *Client) GetServiceHook(projectSlug, hookID string) (*ServiceHook, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/0/projects/%s/%s/hooks/%s/", c.organization, projectSlug, hookID), nil)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error getting service hook: status=%d body=%s", resp.StatusCode, string(body))
	}

	var hook ServiceHook
	err = json.NewDecoder(resp.Body).Decode(&hook)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &hook, nil
}

// DeleteServiceHook deletes a service hook
// DELETE /api/0/projects/{organization_slug}/{project_slug}/hooks/{hook_id}/
func (c *Client) DeleteServiceHook(projectSlug, hookID string) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/0/projects/%s/%s/hooks/%s/", c.organization, projectSlug, hookID), nil)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error deleting service hook: status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
}
