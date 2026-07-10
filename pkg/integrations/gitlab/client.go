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
	NoteableID   int    `json:"noteable_id,omitempty"`
	NoteableIID  int    `json:"noteable_iid,omitempty"`
	NoteableType string `json:"noteable_type,omitempty"`
}

type CreateNoteRequest struct {
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

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create merge request note: status %d, response: %s", resp.StatusCode, readResponseBody(resp))
	}

	var note Note
	if err := json.NewDecoder(resp.Body).Decode(&note); err != nil {
		return nil, fmt.Errorf("failed to decode note: %v", err)
	}

	return &note, nil
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
