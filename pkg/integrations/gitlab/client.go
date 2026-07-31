package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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

	groupIDBytes, _ := ctx.GetConfig("groupId")
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

// listProjects lists the group's projects when a group is configured,
// and the user's personal projects otherwise.
func (c *Client) listProjects(user *User) ([]Project, error) {
	if c.groupID == "" {
		return fetchAllResources[Project](c, func(page int) string {
			return fmt.Sprintf("%s/api/%s/users/%d/projects?per_page=100&page=%d", c.baseURL, apiVersion, user.ID, page)
		})
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

func (c *Client) GetIssue(ctx context.Context, projectID, issueIID string) (*Issue, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/issues/%s", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(issueIID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get issue: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode issue: %v", err)
	}

	return &issue, nil
}

// UpdateIssueRequest mirrors GitLab's PUT /projects/:id/issues/:issue_iid body.
// Fields are pointers so a nil field is omitted (not changed) while a non-nil
// field is always sent, even if it points to a zero value - e.g. a non-nil
// pointer to "" clears the description, and a non-nil pointer to an empty
// slice clears the assignees. Callers must leave a field nil to skip it.
type UpdateIssueRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	StateEvent  *string `json:"state_event,omitempty"`
	Labels      *string `json:"labels,omitempty"`
	AddLabels   *string `json:"add_labels,omitempty"`
	AssigneeIDs *[]int  `json:"assignee_ids,omitempty"`
	MilestoneID *int    `json:"milestone_id,omitempty"`
	DueDate     *string `json:"due_date,omitempty"`
}

func (c *Client) UpdateIssue(ctx context.Context, projectID, issueIID string, req *UpdateIssueRequest) (*Issue, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/issues/%s", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(issueIID))

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to update issue: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode issue: %v", err)
	}

	return &issue, nil
}

func (c *Client) CreateIssueNote(ctx context.Context, projectID, issueIID string, req *CreateNoteRequest) (*Note, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/issues/%s/notes", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(issueIID))

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

	if resp.StatusCode == http.StatusAccepted {
		var quickAction quickActionNoteResponse
		if err := json.NewDecoder(resp.Body).Decode(&quickAction); err != nil {
			return nil, fmt.Errorf("failed to decode quick action response: %v", err)
		}
		return &Note{Body: strings.Join(quickAction.Summary, "; ")}, nil
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create issue note: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var note Note
	if err := json.NewDecoder(resp.Body).Decode(&note); err != nil {
		return nil, fmt.Errorf("failed to decode note: %v", err)
	}

	return &note, nil
}

// UpdateIssueNote edits an existing note (comment) on an issue.
// See https://docs.gitlab.com/api/notes/#modify-existing-issue-note
func (c *Client) UpdateIssueNote(ctx context.Context, projectID, issueIID, noteID string, req *UpdateNoteRequest) (*Note, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/issues/%s/notes/%s", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(issueIID), url.PathEscape(noteID))

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to update issue note: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var note Note
	if err := json.NewDecoder(resp.Body).Decode(&note); err != nil {
		return nil, fmt.Errorf("failed to decode note: %v", err)
	}

	return &note, nil
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

type Environment struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug,omitempty"`
	ExternalURL string `json:"external_url,omitempty"`
	State       string `json:"state,omitempty"`
	Tier        string `json:"tier,omitempty"`
}

// ListEnvironments lists the available environments for a project. Only
// available environments are returned so users pick a live deployment target.
func (c *Client) ListEnvironments(projectID string) ([]Environment, error) {
	return fetchAllResources[Environment](c, func(page int) string {
		return fmt.Sprintf("%s/api/%s/projects/%s/environments?per_page=100&page=%d&states=available", c.baseURL, apiVersion, url.PathEscape(projectID), page)
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

// ListProjectMembers lists all members of a project, including inherited ones.
func (c *Client) ListProjectMembers(projectID string) ([]User, error) {
	return fetchAllResources[User](c, func(page int) string {
		return fmt.Sprintf("%s/api/%s/projects/%s/members/all?per_page=100&page=%d", c.baseURL, apiVersion, url.PathEscape(projectID), page)
	})
}

func (c *Client) FetchIntegrationData() (*User, []Project, error) {
	user, err := c.getCurrentUser()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get current user: %v", err)
	}

	projects, err := c.listProjects(user)
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

type Note struct {
	ID           int    `json:"id"`
	Body         string `json:"body"`
	Author       User   `json:"author"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	System       bool   `json:"system"`
	ProjectID    int    `json:"project_id,omitempty"`
	NoteableID   int    `json:"noteable_id,omitempty"`
	NoteableIID  int    `json:"noteable_iid,omitempty"`
	NoteableType string `json:"noteable_type,omitempty"`
	Resolvable   bool   `json:"resolvable"`
	Confidential bool   `json:"confidential"`
	Internal     bool   `json:"internal"`
}

// quickActionNoteResponse is what GitLab returns instead of a Note when a
// note's body is only quick actions (e.g. "/ready") and has no visible
// comment content: status 202 with a summary of the applied commands.
type quickActionNoteResponse struct {
	Summary []string `json:"summary"`
}

type CreateNoteRequest struct {
	Body string `json:"body"`
}

type UpdateNoteRequest struct {
	Body string `json:"body"`
}

func (c *Client) CreateMergeRequestNote(ctx context.Context, projectID, mergeRequestIID string, req *CreateNoteRequest) (*Note, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/merge_requests/%s/notes", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(mergeRequestIID))

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

	if resp.StatusCode == http.StatusAccepted {
		var quickAction quickActionNoteResponse
		if err := json.NewDecoder(resp.Body).Decode(&quickAction); err != nil {
			return nil, fmt.Errorf("failed to decode quick action response: %v", err)
		}
		return &Note{Body: strings.Join(quickAction.Summary, "; ")}, nil
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create merge request note: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var note Note
	if err := json.NewDecoder(resp.Body).Decode(&note); err != nil {
		return nil, fmt.Errorf("failed to decode note: %v", err)
	}

	return &note, nil
}

type MergeRequest struct {
	ID                       int        `json:"id"`
	IID                      int        `json:"iid"`
	ProjectID                int        `json:"project_id"`
	Title                    string     `json:"title"`
	Description              string     `json:"description"`
	State                    string     `json:"state"`
	CreatedAt                string     `json:"created_at"`
	UpdatedAt                string     `json:"updated_at"`
	MergedAt                 *string    `json:"merged_at"`
	MergeUser                *User      `json:"merge_user"`
	ClosedAt                 *string    `json:"closed_at"`
	ClosedBy                 *User      `json:"closed_by"`
	SourceBranch             string     `json:"source_branch"`
	TargetBranch             string     `json:"target_branch"`
	Author                   User       `json:"author"`
	Assignees                []User     `json:"assignees"`
	Reviewers                []User     `json:"reviewers"`
	Labels                   []string   `json:"labels"`
	Milestone                *Milestone `json:"milestone"`
	Draft                    bool       `json:"draft"`
	DetailedMergeStatus      string     `json:"detailed_merge_status"`
	MergeError               *string    `json:"merge_error"`
	SHA                      string     `json:"sha"`
	MergeCommitSHA           *string    `json:"merge_commit_sha"`
	SquashCommitSHA          *string    `json:"squash_commit_sha"`
	Squash                   bool       `json:"squash"`
	ShouldRemoveSourceBranch bool       `json:"should_remove_source_branch"`
	WebURL                   string     `json:"web_url"`
}

type AcceptMergeRequestRequest struct {
	MergeCommitMessage       string `json:"merge_commit_message,omitempty"`
	SquashCommitMessage      string `json:"squash_commit_message,omitempty"`
	Squash                   *bool  `json:"squash,omitempty"`
	ShouldRemoveSourceBranch *bool  `json:"should_remove_source_branch,omitempty"`
	SHA                      string `json:"sha,omitempty"`
}

// AcceptMergeRequest merges an open merge request.
// See https://docs.gitlab.com/api/merge_requests/#merge-a-merge-request
func (c *Client) AcceptMergeRequest(ctx context.Context, projectID, mergeRequestIID string, req *AcceptMergeRequestRequest) (*MergeRequest, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/merge_requests/%s/merge", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(mergeRequestIID))

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("user does not have permission to accept this merge request")
	case http.StatusMethodNotAllowed:
		return nil, fmt.Errorf("merge request cannot be merged (it may be a draft, closed, already merged, or blocked): %s", parseGitlabErrorMessage(readResponseBody(resp)))
	case http.StatusConflict:
		return nil, errors.New(mergeRequestConflictMessage(resp))
	case http.StatusUnprocessableEntity:
		return nil, fmt.Errorf("branch cannot be merged: %s", parseGitlabErrorMessage(readResponseBody(resp)))
	default:
		return nil, fmt.Errorf("failed to accept merge request: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var mergeRequest MergeRequest
	if err := json.NewDecoder(resp.Body).Decode(&mergeRequest); err != nil {
		return nil, fmt.Errorf("failed to decode merge request: %v", err)
	}

	return &mergeRequest, nil
}

type CreateMergeRequestRequest struct {
	SourceBranch       string `json:"source_branch"`
	TargetBranch       string `json:"target_branch"`
	Title              string `json:"title"`
	Description        string `json:"description,omitempty"`
	AssigneeIDs        []int  `json:"assignee_ids,omitempty"`
	ReviewerIDs        []int  `json:"reviewer_ids,omitempty"`
	Labels             string `json:"labels,omitempty"`
	MilestoneID        *int   `json:"milestone_id,omitempty"`
	RemoveSourceBranch *bool  `json:"remove_source_branch,omitempty"`
	Squash             *bool  `json:"squash,omitempty"`
}

// CreateMergeRequest opens a new merge request in a project.
// See https://docs.gitlab.com/api/merge_requests/#create-mr
func (c *Client) CreateMergeRequest(ctx context.Context, projectID string, req *CreateMergeRequestRequest) (*MergeRequest, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/merge_requests", c.baseURL, apiVersion, url.PathEscape(projectID))

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
		return nil, fmt.Errorf("failed to create merge request: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var mergeRequest MergeRequest
	if err := json.NewDecoder(resp.Body).Decode(&mergeRequest); err != nil {
		return nil, fmt.Errorf("failed to decode merge request: %v", err)
	}

	return &mergeRequest, nil
}

// GetMergeRequest fetches a single merge request, including its current reviewers.
func (c *Client) GetMergeRequest(ctx context.Context, projectID, mergeRequestIID string) (*MergeRequest, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/merge_requests/%s", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(mergeRequestIID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get merge request: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var mergeRequest MergeRequest
	if err := json.NewDecoder(resp.Body).Decode(&mergeRequest); err != nil {
		return nil, fmt.Errorf("failed to decode merge request: %v", err)
	}

	return &mergeRequest, nil
}

// UpdateMergeRequestRequest mirrors GitLab's PUT /projects/:id/merge_requests/:merge_request_iid
// body. Fields are pointers so a nil field is omitted (left unchanged) while a
// non-nil field is always sent, even when it points to a zero value - e.g. a
// non-nil pointer to "" clears the description, and a non-nil pointer to an
// empty slice clears the assignees.
type UpdateMergeRequestRequest struct {
	Title        *string `json:"title,omitempty"`
	Description  *string `json:"description,omitempty"`
	TargetBranch *string `json:"target_branch,omitempty"`
	StateEvent   *string `json:"state_event,omitempty"`
	Labels       *string `json:"labels,omitempty"`
	AddLabels    *string `json:"add_labels,omitempty"`
	AssigneeIDs  *[]int  `json:"assignee_ids,omitempty"`
}

// UpdateMergeRequest edits an existing merge request's fields.
// See https://docs.gitlab.com/api/merge_requests/#update-mr
func (c *Client) UpdateMergeRequest(ctx context.Context, projectID, mergeRequestIID string, req *UpdateMergeRequestRequest) (*MergeRequest, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/merge_requests/%s", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(mergeRequestIID))

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to update merge request: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var mergeRequest MergeRequest
	if err := json.NewDecoder(resp.Body).Decode(&mergeRequest); err != nil {
		return nil, fmt.Errorf("failed to decode merge request: %v", err)
	}

	return &mergeRequest, nil
}

// UpdateMergeRequestReviewersRequest sets the full reviewer list of a merge
// request. GitLab replaces the existing reviewers with the given IDs, so an
// empty (but non-nil) slice clears all reviewers.
type UpdateMergeRequestReviewersRequest struct {
	ReviewerIDs []int `json:"reviewer_ids"`
}

// UpdateMergeRequestReviewers replaces the reviewers of a merge request with the
// given set of user IDs.
// See https://docs.gitlab.com/api/merge_requests/#update-mr
func (c *Client) UpdateMergeRequestReviewers(ctx context.Context, projectID, mergeRequestIID string, reviewerIDs []int) (*MergeRequest, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/merge_requests/%s", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(mergeRequestIID))

	if reviewerIDs == nil {
		reviewerIDs = []int{}
	}

	body, err := json.Marshal(&UpdateMergeRequestReviewersRequest{ReviewerIDs: reviewerIDs})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to update merge request reviewers: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var mergeRequest MergeRequest
	if err := json.NewDecoder(resp.Body).Decode(&mergeRequest); err != nil {
		return nil, fmt.Errorf("failed to decode merge request: %v", err)
	}

	return &mergeRequest, nil
}

type MergeRequestApprover struct {
	User       User   `json:"user"`
	ApprovedAt string `json:"approved_at"`
}

type MergeRequestApproval struct {
	ID                int                    `json:"id"`
	IID               int                    `json:"iid"`
	ProjectID         int                    `json:"project_id"`
	Title             string                 `json:"title"`
	Description       string                 `json:"description"`
	State             string                 `json:"state"`
	CreatedAt         string                 `json:"created_at"`
	UpdatedAt         string                 `json:"updated_at"`
	MergeStatus       string                 `json:"merge_status"`
	ApprovalsRequired int                    `json:"approvals_required"`
	ApprovalsLeft     int                    `json:"approvals_left"`
	ApprovedBy        []MergeRequestApprover `json:"approved_by"`
}

type ApproveMergeRequestRequest struct {
	SHA string `json:"sha,omitempty"`
}

// ApproveMergeRequest approves a merge request as the authenticated user.
// See https://docs.gitlab.com/api/merge_request_approvals/#approve-merge-request
func (c *Client) ApproveMergeRequest(ctx context.Context, projectID, mergeRequestIID string, req *ApproveMergeRequestRequest) (*MergeRequestApproval, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/merge_requests/%s/approve", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(mergeRequestIID))

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

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("user is not allowed to approve this merge request")
	case http.StatusConflict:
		return nil, errors.New(mergeRequestConflictMessage(resp))
	default:
		return nil, fmt.Errorf("failed to approve merge request: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var approval MergeRequestApproval
	if err := json.NewDecoder(resp.Body).Decode(&approval); err != nil {
		return nil, fmt.Errorf("failed to decode merge request approval: %v", err)
	}

	return &approval, nil
}

type AwardEmoji struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	User        User   `json:"user"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	AwardableID int    `json:"awardable_id,omitempty"`
}

type CreateAwardEmojiRequest struct {
	Name string `json:"name"`
}

// errAwardEmojiAlreadyExists is returned when the authenticated user has already
// awarded the given emoji to the target - GitLab reports this as a 404 with an
// "Award Emoji Name has already been taken" message rather than a 409 Conflict.
var errAwardEmojiAlreadyExists = errors.New("award emoji already exists")

// CreateMergeRequestAwardEmoji adds an award emoji to the merge request itself.
// If the authenticated user has already awarded this emoji, it returns the
// existing award emoji instead of failing, making the operation idempotent.
func (c *Client) CreateMergeRequestAwardEmoji(ctx context.Context, projectID, mergeRequestIID string, req *CreateAwardEmojiRequest) (*AwardEmoji, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/merge_requests/%s/award_emoji", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(mergeRequestIID))
	awardEmoji, err := c.createAwardEmoji(ctx, apiURL, req)
	if errors.Is(err, errAwardEmojiAlreadyExists) {
		return c.findExistingAwardEmoji(req.Name, func() ([]AwardEmoji, error) {
			return c.ListMergeRequestAwardEmoji(projectID, mergeRequestIID)
		})
	}
	return awardEmoji, err
}

// CreateMergeRequestNoteAwardEmoji adds an award emoji to a note on a merge request.
// If the authenticated user has already awarded this emoji, it returns the
// existing award emoji instead of failing, making the operation idempotent.
func (c *Client) CreateMergeRequestNoteAwardEmoji(ctx context.Context, projectID, mergeRequestIID, noteID string, req *CreateAwardEmojiRequest) (*AwardEmoji, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/merge_requests/%s/notes/%s/award_emoji", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(mergeRequestIID), url.PathEscape(noteID))
	awardEmoji, err := c.createAwardEmoji(ctx, apiURL, req)
	if errors.Is(err, errAwardEmojiAlreadyExists) {
		return c.findExistingAwardEmoji(req.Name, func() ([]AwardEmoji, error) {
			return c.ListMergeRequestNoteAwardEmoji(projectID, mergeRequestIID, noteID)
		})
	}
	return awardEmoji, err
}

// ListMergeRequestAwardEmoji lists the award emoji on a merge request.
func (c *Client) ListMergeRequestAwardEmoji(projectID, mergeRequestIID string) ([]AwardEmoji, error) {
	return fetchAllResources[AwardEmoji](c, func(page int) string {
		return fmt.Sprintf("%s/api/%s/projects/%s/merge_requests/%s/award_emoji?per_page=100&page=%d", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(mergeRequestIID), page)
	})
}

// ListMergeRequestNoteAwardEmoji lists the award emoji on a note of a merge request.
func (c *Client) ListMergeRequestNoteAwardEmoji(projectID, mergeRequestIID, noteID string) ([]AwardEmoji, error) {
	return fetchAllResources[AwardEmoji](c, func(page int) string {
		return fmt.Sprintf("%s/api/%s/projects/%s/merge_requests/%s/notes/%s/award_emoji?per_page=100&page=%d", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(mergeRequestIID), url.PathEscape(noteID), page)
	})
}

// findExistingAwardEmoji locates the authenticated user's award emoji with the given
// name among the results of listFn, used to recover from errAwardEmojiAlreadyExists.
func (c *Client) findExistingAwardEmoji(name string, listFn func() ([]AwardEmoji, error)) (*AwardEmoji, error) {
	user, err := c.getCurrentUser()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %v", err)
	}

	awardEmoji, err := listFn()
	if err != nil {
		return nil, fmt.Errorf("failed to list existing award emoji: %v", err)
	}

	for _, e := range awardEmoji {
		if e.Name == name && e.User.ID == user.ID {
			return &e, nil
		}
	}

	return nil, fmt.Errorf("award emoji %q reported as already existing but could not be found", name)
}

func (c *Client) createAwardEmoji(ctx context.Context, apiURL string, req *CreateAwardEmojiRequest) (*AwardEmoji, error) {
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
		responseBody := readResponseBody(resp)
		if resp.StatusCode == http.StatusNotFound && strings.Contains(parseGitlabErrorMessage(responseBody), "already been taken") {
			return nil, errAwardEmojiAlreadyExists
		}
		return nil, fmt.Errorf("failed to create award emoji: status %d, response: %s", resp.StatusCode, responseBody)
	}

	var awardEmoji AwardEmoji
	if err := json.NewDecoder(resp.Body).Decode(&awardEmoji); err != nil {
		return nil, fmt.Errorf("failed to decode award emoji: %v", err)
	}

	return &awardEmoji, nil
}

type DeploymentEnvironment struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ExternalURL string `json:"external_url,omitempty"`
}

type Deployment struct {
	ID          int                    `json:"id"`
	IID         int                    `json:"iid"`
	Ref         string                 `json:"ref"`
	SHA         string                 `json:"sha"`
	Status      string                 `json:"status"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at,omitempty"`
	User        *User                  `json:"user,omitempty"`
	Environment *DeploymentEnvironment `json:"environment,omitempty"`
	Deployable  map[string]any         `json:"deployable,omitempty"`
}

type CreateDeploymentRequest struct {
	Environment string `json:"environment"`
	Ref         string `json:"ref"`
	SHA         string `json:"sha"`
	Tag         bool   `json:"tag"`
	Status      string `json:"status"`
}

type UpdateDeploymentRequest struct {
	Status string `json:"status"`
}

// CreateDeployment creates a deployment for a project environment.
// GitLab creates the environment automatically if it does not yet exist.
func (c *Client) CreateDeployment(ctx context.Context, projectID string, req *CreateDeploymentRequest) (*Deployment, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/deployments", c.baseURL, apiVersion, url.PathEscape(projectID))

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
		return nil, fmt.Errorf("failed to create deployment: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var deployment Deployment
	if err := json.NewDecoder(resp.Body).Decode(&deployment); err != nil {
		return nil, fmt.Errorf("failed to decode deployment: %v", err)
	}

	return &deployment, nil
}

// UpdateDeployment updates the status of an existing deployment.
func (c *Client) UpdateDeployment(ctx context.Context, projectID string, deploymentID int, req *UpdateDeploymentRequest) (*Deployment, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/deployments/%d", c.baseURL, apiVersion, url.PathEscape(projectID), deploymentID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to update deployment: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var deployment Deployment
	if err := json.NewDecoder(resp.Body).Decode(&deployment); err != nil {
		return nil, fmt.Errorf("failed to decode deployment: %v", err)
	}

	return &deployment, nil
}

type ReleaseCommit struct {
	ID             string   `json:"id"`
	ShortID        string   `json:"short_id"`
	Title          string   `json:"title"`
	CreatedAt      string   `json:"created_at"`
	ParentIDs      []string `json:"parent_ids,omitempty"`
	Message        string   `json:"message"`
	AuthorName     string   `json:"author_name"`
	AuthorEmail    string   `json:"author_email"`
	AuthoredDate   string   `json:"authored_date"`
	CommitterName  string   `json:"committer_name"`
	CommitterEmail string   `json:"committer_email"`
	CommittedDate  string   `json:"committed_date"`
}

type ReleaseMilestoneIssueStats struct {
	Total  int `json:"total"`
	Closed int `json:"closed"`
}

// ReleaseMilestone is kept separate from Milestone (used by issues) since the releases API returns a fuller shape.
type ReleaseMilestone struct {
	ID          int                         `json:"id"`
	IID         int                         `json:"iid"`
	ProjectID   int                         `json:"project_id"`
	Title       string                      `json:"title"`
	Description string                      `json:"description"`
	State       string                      `json:"state"`
	CreatedAt   string                      `json:"created_at"`
	UpdatedAt   string                      `json:"updated_at"`
	DueDate     string                      `json:"due_date,omitempty"`
	StartDate   string                      `json:"start_date,omitempty"`
	WebURL      string                      `json:"web_url"`
	IssueStats  *ReleaseMilestoneIssueStats `json:"issue_stats,omitempty"`
}

type ReleaseAssetSource struct {
	Format string `json:"format"`
	URL    string `json:"url"`
}

type ReleaseAssetLink struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	LinkType string `json:"link_type"`
}

type ReleaseAssets struct {
	Count   int                  `json:"count"`
	Sources []ReleaseAssetSource `json:"sources,omitempty"`
	Links   []ReleaseAssetLink   `json:"links,omitempty"`
}

type ReleaseEvidence struct {
	SHA         string `json:"sha"`
	Filepath    string `json:"filepath"`
	CollectedAt string `json:"collected_at"`
}

type ReleaseLinks struct {
	ClosedIssuesURL        string `json:"closed_issues_url,omitempty"`
	ClosedMergeRequestsURL string `json:"closed_merge_requests_url,omitempty"`
	EditURL                string `json:"edit_url,omitempty"`
	MergedMergeRequestsURL string `json:"merged_merge_requests_url,omitempty"`
	OpenedIssuesURL        string `json:"opened_issues_url,omitempty"`
	OpenedMergeRequestsURL string `json:"opened_merge_requests_url,omitempty"`
	Self                   string `json:"self,omitempty"`
}

type Release struct {
	TagName         string             `json:"tag_name"`
	Name            string             `json:"name"`
	Description     string             `json:"description"`
	CreatedAt       string             `json:"created_at"`
	ReleasedAt      string             `json:"released_at"`
	UpcomingRelease bool               `json:"upcoming_release"`
	Author          *User              `json:"author,omitempty"`
	Commit          *ReleaseCommit     `json:"commit,omitempty"`
	Milestones      []ReleaseMilestone `json:"milestones,omitempty"`
	CommitPath      string             `json:"commit_path,omitempty"`
	TagPath         string             `json:"tag_path,omitempty"`
	Assets          *ReleaseAssets     `json:"assets,omitempty"`
	Evidences       []ReleaseEvidence  `json:"evidences,omitempty"`
	EvidenceSHA     string             `json:"evidence_sha,omitempty"`
	Links           *ReleaseLinks      `json:"_links,omitempty"`
}

type CreateReleaseRequest struct {
	TagName     string   `json:"tag_name"`
	Ref         string   `json:"ref,omitempty"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Milestones  []string `json:"milestones,omitempty"`
	ReleasedAt  string   `json:"released_at,omitempty"`
}

// CreateRelease creates a release, tagging Ref first if TagName doesn't already exist.
func (c *Client) CreateRelease(ctx context.Context, projectID string, req *CreateReleaseRequest) (*Release, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/releases", c.baseURL, apiVersion, url.PathEscape(projectID))

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
		return nil, fmt.Errorf("failed to create release: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release: %v", err)
	}

	return &release, nil
}

// UpdateReleaseRequest fields are pointers so a nil field is left unchanged while a non-nil field (even a zero value) is always sent.
type UpdateReleaseRequest struct {
	Name        *string   `json:"name,omitempty"`
	Description *string   `json:"description,omitempty"`
	Milestones  *[]string `json:"milestones,omitempty"`
	ReleasedAt  *string   `json:"released_at,omitempty"`
}

// UpdateRelease edits an existing release's name, description, milestones, or released-at date; the tag and assets can't be changed here.
func (c *Client) UpdateRelease(ctx context.Context, projectID, tagName string, req *UpdateReleaseRequest) (*Release, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/releases/%s", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(tagName))

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to update release: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release: %v", err)
	}

	return &release, nil
}

// GetRelease fetches a single release by tag name.
func (c *Client) GetRelease(ctx context.Context, projectID, tagName string) (*Release, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/releases/%s", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(tagName))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get release: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release: %v", err)
	}

	return &release, nil
}

// DeleteRelease deletes a release and returns the deleted release; it does not delete the underlying tag.
func (c *Client) DeleteRelease(ctx context.Context, projectID, tagName string) (*Release, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/releases/%s", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(tagName))

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to delete release: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release: %v", err)
	}

	return &release, nil
}

// DeleteTag deletes a Git tag from the project's repository.
func (c *Client) DeleteTag(ctx context.Context, projectID, tagName string) error {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/repository/tags/%s", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(tagName))

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, apiURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete tag: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	return nil
}

// GetLatestRelease returns the most recently published release, skipping upcoming (scheduled) ones.
func (c *Client) GetLatestRelease(ctx context.Context, projectID string) (*Release, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/releases?order_by=released_at&sort=desc&per_page=100", c.baseURL, apiVersion, url.PathEscape(projectID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list releases: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode releases: %v", err)
	}

	for _, release := range releases {
		if !release.UpcomingRelease {
			return &release, nil
		}
	}

	return nil, errors.New("no published releases found")
}

// CommitStatus is a GitLab commit build/CI status.
// See https://docs.gitlab.com/api/commits/#set-the-pipeline-status-of-a-commit
type CommitStatus struct {
	ID           int      `json:"id"`
	SHA          string   `json:"sha"`
	Ref          string   `json:"ref"`
	Status       string   `json:"status"`
	Name         string   `json:"name"`
	TargetURL    string   `json:"target_url"`
	Description  string   `json:"description"`
	CreatedAt    string   `json:"created_at"`
	StartedAt    string   `json:"started_at"`
	FinishedAt   string   `json:"finished_at"`
	AllowFailure bool     `json:"allow_failure"`
	Coverage     *float64 `json:"coverage"`
	PipelineID   int      `json:"pipeline_id,omitempty"`
	Author       *User    `json:"author,omitempty"`
}

// CreateCommitStatusRequest mirrors GitLab's POST /projects/:id/statuses/:sha body.
// Only State is required; the rest are omitted when empty.
type CreateCommitStatusRequest struct {
	State       string   `json:"state"`
	Ref         string   `json:"ref,omitempty"`
	Name        string   `json:"name,omitempty"`
	TargetURL   string   `json:"target_url,omitempty"`
	Description string   `json:"description,omitempty"`
	Coverage    *float64 `json:"coverage,omitempty"`
	PipelineID  *int     `json:"pipeline_id,omitempty"`
}

// CreateCommitStatus sets (publishes) a build/CI status on a commit.
func (c *Client) CreateCommitStatus(ctx context.Context, projectID, sha string, req *CreateCommitStatusRequest) (*CommitStatus, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/statuses/%s", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(sha))

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
		return nil, fmt.Errorf("failed to create commit status: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var status CommitStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode commit status: %v", err)
	}

	return &status, nil
}

// CommitPipeline is the last pipeline associated with a commit - GitLab's native
// rolled-up CI status for that commit.
type CommitPipeline struct {
	ID        int    `json:"id"`
	IID       int    `json:"iid,omitempty"`
	ProjectID int    `json:"project_id,omitempty"`
	Ref       string `json:"ref"`
	SHA       string `json:"sha"`
	Status    string `json:"status"`
	Source    string `json:"source,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	WebURL    string `json:"web_url,omitempty"`
}

// CommitStats is the line-change summary of a commit.
type CommitStats struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Total     int `json:"total"`
}

// Commit is a GitLab repository commit. Its top-level Status and LastPipeline are
// GitLab's native overall CI status for the commit (rolled up from its pipeline).
// See https://docs.gitlab.com/api/commits/#get-a-single-commit
type Commit struct {
	ID             string          `json:"id"`
	ShortID        string          `json:"short_id"`
	Title          string          `json:"title"`
	Message        string          `json:"message"`
	AuthorName     string          `json:"author_name"`
	AuthorEmail    string          `json:"author_email"`
	AuthoredDate   string          `json:"authored_date"`
	CommitterName  string          `json:"committer_name"`
	CommitterEmail string          `json:"committer_email"`
	CommittedDate  string          `json:"committed_date"`
	CreatedAt      string          `json:"created_at"`
	ParentIDs      []string        `json:"parent_ids"`
	WebURL         string          `json:"web_url"`
	Status         string          `json:"status"`
	LastPipeline   *CommitPipeline `json:"last_pipeline,omitempty"`
	Stats          *CommitStats    `json:"stats,omitempty"`
}

// GetCommit returns a single commit, including its top-level Status and LastPipeline
// (GitLab's native overall CI status). The ref may be a SHA, branch, or tag name.
func (c *Client) GetCommit(ctx context.Context, projectID, ref string) (*Commit, error) {
	apiURL := fmt.Sprintf("%s/api/%s/projects/%s/repository/commits/%s", c.baseURL, apiVersion, url.PathEscape(projectID), url.PathEscape(ref))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get commit: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var commit Commit
	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		return nil, fmt.Errorf("failed to decode commit: %v", err)
	}

	return &commit, nil
}

// mergeRequestConflictMessage extracts GitLab's error message from a 409
// response to a merge request accept/approve call. GitLab returns 409 not only
// for a sha guard mismatch but also e.g. when the merge request is locked or
// another merge is in progress, so surface the server's reason and only fall
// back to the documented SHA-mismatch message when the body is empty.
func mergeRequestConflictMessage(resp *http.Response) string {
	if message := parseGitlabErrorMessage(readResponseBody(resp)); message != "" {
		return message
	}
	return "SHA does not match HEAD of source branch"
}

// parseGitlabErrorMessage extracts the "message" field from a GitLab JSON error
// body (e.g. {"message":"404 Award Emoji Name has already been taken"}),
// falling back to the raw body if it isn't in that shape.
func parseGitlabErrorMessage(body string) string {
	var errResp struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(body), &errResp); err == nil && errResp.Message != "" {
		return errResp.Message
	}
	return body
}

func readResponseBody(resp *http.Response) string {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return ""
	}
	return string(body)
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors,omitempty"`
}

// graphQL executes a query against GitLab's GraphQL API and decodes the "data" field into out.
func (c *Client) graphQL(ctx context.Context, query string, variables map[string]any, out any) error {
	body, err := json.Marshal(graphQLRequest{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	apiURL := fmt.Sprintf("%s/api/graphql", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("graphql request failed: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var gqlResp graphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return fmt.Errorf("failed to decode graphql response: %v", err)
	}

	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("graphql error: %s", gqlResp.Errors[0].Message)
	}

	if out == nil {
		return nil
	}

	return json.Unmarshal(gqlResp.Data, out)
}

type Group struct {
	ID       int    `json:"id"`
	FullPath string `json:"full_path"`
}

// GetGroup fetches a single group by numeric ID or full path.
func (c *Client) GetGroup(groupID string) (*Group, error) {
	apiURL := fmt.Sprintf("%s/api/%s/groups/%s", c.baseURL, apiVersion, url.PathEscape(groupID))

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
		return nil, fmt.Errorf("failed to get group: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var group Group
	if err := json.NewDecoder(resp.Body).Decode(&group); err != nil {
		return nil, fmt.Errorf("failed to decode group: %v", err)
	}

	return &group, nil
}

type CiMinutesNamespaceUsage struct {
	Month                 string `json:"month"`
	MonthIso8601          string `json:"monthIso8601"`
	Minutes               int    `json:"minutes"`
	SharedRunnersDuration int    `json:"sharedRunnersDuration"`
}

type CiMinutesProjectRef struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	FullPath string `json:"fullPath"`
}

type CiMinutesProjectUsage struct {
	Minutes               int                  `json:"minutes"`
	SharedRunnersDuration int                  `json:"sharedRunnersDuration"`
	Project               *CiMinutesProjectRef `json:"project"`
}

const ciMinutesNamespaceUsageQuery = `
query($namespaceId: NamespaceID, $date: Date) {
  ciMinutesUsage(namespaceId: $namespaceId, date: $date) {
    nodes {
      month
      monthIso8601
      minutes
      sharedRunnersDuration
    }
  }
}`

const ciMinutesProjectUsageQuery = `
query($namespaceId: NamespaceID, $date: Date, $after: String) {
  ciMinutesProjectMonthlyUsage(namespaceId: $namespaceId, date: $date, first: 100, after: $after) {
    nodes {
      minutes
      sharedRunnersDuration
      project {
        id
        name
        fullPath
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}`

type ciMinutesNamespaceUsageResponse struct {
	CiMinutesUsage struct {
		Nodes []CiMinutesNamespaceUsage `json:"nodes"`
	} `json:"ciMinutesUsage"`
}

type graphQLPageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type ciMinutesProjectUsageResponse struct {
	CiMinutesProjectMonthlyUsage struct {
		Nodes    []CiMinutesProjectUsage `json:"nodes"`
		PageInfo graphQLPageInfo         `json:"pageInfo"`
	} `json:"ciMinutesProjectMonthlyUsage"`
}

// GetCiMinutesUsage returns a namespace's CI/CD minutes usage for the given month with a per-project breakdown, a nil namespaceGID for the current user's personal namespace, and a nil usage if GitLab has no namespace-level record for that month.
func (c *Client) GetCiMinutesUsage(ctx context.Context, namespaceGID *string, date string) (*CiMinutesNamespaceUsage, []CiMinutesProjectUsage, error) {
	variables := map[string]any{"date": date}
	if namespaceGID != nil {
		variables["namespaceId"] = *namespaceGID
	}

	var namespaceResp ciMinutesNamespaceUsageResponse
	if err := c.graphQL(ctx, ciMinutesNamespaceUsageQuery, variables, &namespaceResp); err != nil {
		return nil, nil, err
	}

	var usage *CiMinutesNamespaceUsage
	if len(namespaceResp.CiMinutesUsage.Nodes) > 0 {
		usage = &namespaceResp.CiMinutesUsage.Nodes[0]
	}

	projects, err := c.getAllCiMinutesProjectUsage(ctx, variables)
	if err != nil {
		return nil, nil, err
	}

	return usage, projects, nil
}

// getAllCiMinutesProjectUsage pages through the full per-project usage breakdown, since GitLab caps each response at 100 nodes.
func (c *Client) getAllCiMinutesProjectUsage(ctx context.Context, baseVariables map[string]any) ([]CiMinutesProjectUsage, error) {
	var allProjects []CiMinutesProjectUsage
	after := ""

	for {
		variables := make(map[string]any, len(baseVariables)+1)
		for k, v := range baseVariables {
			variables[k] = v
		}
		if after != "" {
			variables["after"] = after
		}

		var resp ciMinutesProjectUsageResponse
		if err := c.graphQL(ctx, ciMinutesProjectUsageQuery, variables, &resp); err != nil {
			return nil, err
		}

		allProjects = append(allProjects, resp.CiMinutesProjectMonthlyUsage.Nodes...)

		if !resp.CiMinutesProjectMonthlyUsage.PageInfo.HasNextPage {
			break
		}
		after = resp.CiMinutesProjectMonthlyUsage.PageInfo.EndCursor
	}

	return allProjects, nil
}
