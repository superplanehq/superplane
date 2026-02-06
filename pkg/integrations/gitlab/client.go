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

	groupIDBytes, err := ctx.GetConfig("groupId")
	if err != nil || len(groupIDBytes) == 0 {
		return nil, fmt.Errorf("groupId is required")
	}
	groupID := string(groupIDBytes)

	token, err := getAuthToken(ctx, authType)
	if err != nil {
		return nil, err
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
	setAuthHeaders(req, c.authType, c.token)
	return c.httpClient.Do(req)
}



// T is the type of the resource item (e.g. Project, Milestone, User).
func fetchResourcesPage[T any](c *Client, apiURL string) ([]T, string, error) {
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
			return nil, "", fmt.Errorf("resource not found: status 404")
		}
		return nil, "", fmt.Errorf("failed to list resources: status %d", resp.StatusCode)
	}

	var resources []T
	if err := json.NewDecoder(resp.Body).Decode(&resources); err != nil {
		return nil, "", fmt.Errorf("failed to decode resources: %v", err)
	}

	return resources, resp.Header.Get("X-Next-Page"), nil
}

// urlBuilder is a function that returns the URL for a given page.
func fetchAllResources[T any](c *Client, urlBuilder func(page int) string) ([]T, error) {
	var allResources []T
	page := 1

	for {
		resources, nextPage, err := fetchResourcesPage[T](c, urlBuilder(page))
		if err != nil {
			return nil, err
		}

		allResources = append(allResources, resources...)

		if nextPage == "" {
			break
		}
		page++
	}

	return allResources, nil
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

	return fetchAllResources[Project](c, func(page int) string {
		return fmt.Sprintf("%s/api/%s/groups/%s/projects?include_subgroups=true&per_page=100&page=%d", c.baseURL, apiVersion, url.PathEscape(c.groupID), page)
	})
}

type IssueRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Labels      string   `json:"labels,omitempty"`
	AssigneeIDs []int    `json:"assignee_ids,omitempty"`
	MilestoneID *int     `json:"milestone_id,omitempty"`
	DueDate     string   `json:"due_date,omitempty"`
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

type Milestone struct {
	ID    int    `json:"id"`
	IID   int    `json:"iid"`
	Title string `json:"title"`
	State string `json:"state"`
}

func (c *Client) ListMilestones(projectID string) ([]Milestone, error) {
	return fetchAllResources[Milestone](c, func(page int) string {
		return fmt.Sprintf("%s/api/%s/projects/%s/milestones?per_page=100&page=%d&state=active", c.baseURL, apiVersion, url.PathEscape(projectID), page)
	})
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
	return fetchAllResources[User](c, func(page int) string {
		return fmt.Sprintf("%s/api/%s/groups/%s/members?per_page=100&page=%d", c.baseURL, apiVersion, url.PathEscape(groupID), page)
	})
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
