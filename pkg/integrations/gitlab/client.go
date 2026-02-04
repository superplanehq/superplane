package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/superplanehq/superplane/pkg/core"
)

const apiVersion = "v4"

type Client struct {
	baseURL    string
	token      string
	authType   string
	groupID    string
	httpClient core.HTTPContext
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	config, err := ctx.GetConfig("authType")
	if err != nil {
		return nil, fmt.Errorf("failed to get authType: %v", err)
	}
	authType := string(config)

	baseURLBytes, _ := ctx.GetConfig("baseUrl")
	baseURL := normalizeBaseURL(string(baseURLBytes))

	var token string

	groupIDBytes, err := ctx.GetConfig("groupId")
	if err != nil || len(groupIDBytes) == 0 {
		return nil, fmt.Errorf("groupId is required")
	}
	groupID := string(groupIDBytes)

	switch authType {
	case AuthTypePersonalAccessToken:
		tokenBytes, err := ctx.GetConfig("personalAccessToken")
		if err != nil {
			return nil, err
		}
		token = string(tokenBytes)
		if token == "" {
			return nil, fmt.Errorf("personal access token not found")
		}

	case AuthTypeAppOAuth:
		token, err = findSecret(ctx, OAuthAccessToken)
		if err != nil {
			return nil, err
		}
		if token == "" {
			return nil, fmt.Errorf("OAuth access token not found")
		}

	default:
		return nil, fmt.Errorf("unknown auth type: %s", authType)
	}

	return &Client{
		baseURL:    baseURL,
		token:      token,
		authType:   authType,
		groupID:    groupID,
		httpClient: httpClient,
	}, nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.authType == AuthTypePersonalAccessToken {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	} else {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return c.httpClient.Do(req)
}

type Project struct {
	ID                int    `json:"id"`
	PathWithNamespace string `json:"path_with_namespace"`
	WebURL            string `json:"web_url"`
}

func (c *Client) listProjects() ([]Project, error) {
	if c.groupID == "" {
		return nil, fmt.Errorf("groupID is missing")
	}

	allProjects := []Project{}
	page := 1

	for {
		projects, nextPage, err := c.fetchProjectsPage(page)
		if err != nil {
			return nil, err
		}

		allProjects = append(allProjects, projects...)

		if nextPage == "" {
			break
		}
		page++
	}

	return allProjects, nil
}

func (c *Client) fetchProjectsPage(page int) ([]Project, string, error) {
	apiURL := fmt.Sprintf("%s/api/%s/groups/%s/projects?include_subgroups=true&per_page=100&page=%d", c.baseURL, apiVersion, url.PathEscape(c.groupID), page)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, "", fmt.Errorf("group not found (404). Please verify that the Group ID '%s' is correct and you have access to it", c.groupID)
		}
		return nil, "", fmt.Errorf("failed to list projects: status %d", resp.StatusCode)
	}

	var projects []Project
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, "", fmt.Errorf("failed to decode projects: %v", err)
	}

	return projects, resp.Header.Get("X-Next-Page"), nil
}

type IssueRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	AssigneeIDs []int    `json:"assignee_ids,omitempty"`
}

type Issue struct {
	ID          int      `json:"id"`
	IID         int      `json:"iid"`
	ProjectID   int      `json:"project_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	State       string   `json:"state"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	Labels      []string `json:"labels"`
	WebURL      string   `json:"web_url"`
	Author      User     `json:"author"`
	Assignees   []User   `json:"assignees"`
}

type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	State     string `json:"state"`
	AvatarURL string `json:"avatar_url"`
	WebURL    string `json:"web_url"`
}

func (c *Client) CreateIssue(ctx context.Context, projectID string, req *IssueRequest) (*Issue, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/issues", c.baseURL, apiVersion, url.PathEscape(projectID))

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create issue: status %d", resp.StatusCode)
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode issue: %v", err)
	}

	return &issue, nil
}

func (c *Client) getCurrentUser() (*User, error) {
	apiURL := fmt.Sprintf("%s/api/%s/user", c.baseURL, apiVersion)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get current user: status %d", resp.StatusCode)
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user: %v", err)
	}

	return &user, nil
}

func (c *Client) ListGroupMembers(groupID string) ([]User, error) {
	allMembers := []User{}
	page := 1

	for {
		members, nextPage, err := c.fetchGroupMembersPage(groupID, page)
		if err != nil {
			return nil, err
		}

		allMembers = append(allMembers, members...)

		if nextPage == "" {
			break
		}
		page++
	}

	return allMembers, nil
}

func (c *Client) fetchGroupMembersPage(groupID string, page int) ([]User, string, error) {
	apiURL := fmt.Sprintf("%s/api/%s/groups/%s/members?per_page=100&page=%d", c.baseURL, apiVersion, url.PathEscape(groupID), page)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to list group members: status %d", resp.StatusCode)
	}

	var members []User
	if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
		return nil, "", fmt.Errorf("failed to decode members: %v", err)
	}

	return members, resp.Header.Get("X-Next-Page"), nil
}

func (c *Client) FetchIntegrationData() (*User, []Project, error) {
	user, err := c.getCurrentUser()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get current user: %v", err)
	}

	projects, err := c.listProjects()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list projects: %v", err)
	}

	return user, projects, nil
}
