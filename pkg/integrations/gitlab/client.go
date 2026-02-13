package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Labels      string `json:"labels,omitempty"`
	AssigneeIDs []int  `json:"assignee_ids,omitempty"`
	MilestoneID *int   `json:"milestone_id,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
}

type Issue struct {
	ID          int        `json:"id"`
	IID         int        `json:"iid"`
	ProjectID   int        `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	State       string     `json:"state"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
	ClosedAt    *string    `json:"closed_at"`
	ClosedBy    *User      `json:"closed_by"`
	Labels      []string   `json:"labels"`
	Milestone   *Milestone `json:"milestone"`
	DueDate     *string    `json:"due_date"`
	WebURL      string     `json:"web_url"`
	Author      User       `json:"author"`
	Assignees   []User     `json:"assignees"`
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

type PipelineVariable struct {
	Key          string `json:"key"`
	Value        string `json:"value"`
	VariableType string `json:"variable_type,omitempty"`
}

type CreatePipelineRequest struct {
	Ref    string            `json:"ref"`
	Inputs map[string]string `json:"inputs,omitempty"`
}

type PipelineInput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Pipeline struct {
	ID             int            `json:"id"`
	IID            int            `json:"iid"`
	ProjectID      int            `json:"project_id"`
	Status         string         `json:"status"`
	Source         string         `json:"source,omitempty"`
	Ref            string         `json:"ref"`
	SHA            string         `json:"sha"`
	BeforeSHA      string         `json:"before_sha,omitempty"`
	Tag            bool           `json:"tag,omitempty"`
	YamlErrors     *string        `json:"yaml_errors,omitempty"`
	WebURL         string         `json:"web_url"`
	URL            string         `json:"url,omitempty"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
	StartedAt      string         `json:"started_at,omitempty"`
	FinishedAt     string         `json:"finished_at,omitempty"`
	CommittedAt    string         `json:"committed_at,omitempty"`
	Duration       float64        `json:"duration,omitempty"`
	QueuedDuration float64        `json:"queued_duration,omitempty"`
	Coverage       string         `json:"coverage,omitempty"`
	User           map[string]any `json:"user,omitempty"`
	DetailedStatus map[string]any `json:"detailed_status,omitempty"`
}

type PipelineTestReportSummary struct {
	Total      map[string]any   `json:"total"`
	TestSuites []map[string]any `json:"test_suites"`
}

func (c *Client) CreatePipeline(ctx context.Context, projectID string, req *CreatePipelineRequest) (*Pipeline, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/pipeline", c.baseURL, apiVersion, url.PathEscape(projectID))

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
		return nil, fmt.Errorf("failed to create pipeline: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var pipeline Pipeline
	if err := json.NewDecoder(resp.Body).Decode(&pipeline); err != nil {
		return nil, fmt.Errorf("failed to decode pipeline: %v", err)
	}

	if pipeline.WebURL == "" && pipeline.URL != "" {
		pipeline.WebURL = pipeline.URL
	}

	return &pipeline, nil
}

func (c *Client) GetPipeline(projectID string, pipelineID int) (*Pipeline, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/pipelines/%d", c.baseURL, apiVersion, url.PathEscape(projectID), pipelineID)
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
		return nil, fmt.Errorf("failed to get pipeline: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var pipeline Pipeline
	if err := json.NewDecoder(resp.Body).Decode(&pipeline); err != nil {
		return nil, fmt.Errorf("failed to decode pipeline: %v", err)
	}

	if pipeline.WebURL == "" && pipeline.URL != "" {
		pipeline.WebURL = pipeline.URL
	}

	return &pipeline, nil
}

func (c *Client) CancelPipeline(ctx context.Context, projectID string, pipelineID int) error {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/pipelines/%d/cancel", c.baseURL, apiVersion, url.PathEscape(projectID), pipelineID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
		return nil
	default:
		return fmt.Errorf("failed to cancel pipeline: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}
}

func (c *Client) GetLatestPipeline(projectID, ref string) (*Pipeline, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/pipelines/latest", c.baseURL, apiVersion, url.PathEscape(projectID))
	if ref != "" {
		apiURL += fmt.Sprintf("?ref=%s", url.QueryEscape(ref))
	}

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
		return nil, fmt.Errorf("failed to get latest pipeline: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var pipeline Pipeline
	if err := json.NewDecoder(resp.Body).Decode(&pipeline); err != nil {
		return nil, fmt.Errorf("failed to decode pipeline: %v", err)
	}

	if pipeline.WebURL == "" && pipeline.URL != "" {
		pipeline.WebURL = pipeline.URL
	}

	return &pipeline, nil
}

func (c *Client) GetPipelineTestReportSummary(projectID string, pipelineID int) (*PipelineTestReportSummary, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/pipelines/%d/test_report_summary", c.baseURL, apiVersion, url.PathEscape(projectID), pipelineID)
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
		return nil, fmt.Errorf("failed to get pipeline test report summary: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var summary PipelineTestReportSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, fmt.Errorf("failed to decode pipeline test report summary: %v", err)
	}

	return &summary, nil
}

func (c *Client) ListPipelines(projectID string) ([]Pipeline, error) {
	return fetchAllResources[Pipeline](c, func(page int) string {
		return fmt.Sprintf("%s/api/%s/projects/%s/pipelines?per_page=100&page=%d", c.baseURL, apiVersion, url.PathEscape(projectID), page)
	})
}

func readResponseBody(resp *http.Response) string {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return ""
	}
	return string(body)
}
