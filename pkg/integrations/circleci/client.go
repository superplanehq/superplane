package circleci

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/superplanehq/superplane/pkg/core"
)

const baseURL = "https://circleci.com/api/v2"

type Client struct {
	APIToken string
	http     core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, err
	}

	return &Client{
		APIToken: string(apiToken),
		http:     http,
	}, nil
}

func (c *Client) execRequest(method, requestURL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Circle-Token", c.APIToken)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

type UserResponse struct {
	ID    string `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
}

func (c *Client) GetCurrentUser() (*UserResponse, error) {
	reqURL := fmt.Sprintf("%s/me", baseURL)
	responseBody, err := c.execRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	var user UserResponse
	err = json.Unmarshal(responseBody, &user)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &user, nil
}

type CollaborationResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	VCSType string `json:"vcs-type"`
}

func (c *Client) GetCollaborations() ([]CollaborationResponse, error) {
	reqURL := fmt.Sprintf("%s/me/collaborations", baseURL)
	responseBody, err := c.execRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	var collaborations []CollaborationResponse
	if err := json.Unmarshal(responseBody, &collaborations); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return collaborations, nil
}

type ProjectResponse struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	OrganizationName string `json:"organization_name"`
}

func (c *Client) GetProject(projectSlug string) (*ProjectResponse, error) {
	path := fmt.Sprintf("%s/project/%s", baseURL, projectSlug)
	responseBody, err := c.execRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var project ProjectResponse
	err = json.Unmarshal(responseBody, &project)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &project, nil
}

type PipelineResponse struct {
	ID          string                 `json:"id"`
	Number      int                    `json:"number"`
	State       string                 `json:"state"`
	ProjectSlug string                 `json:"project_slug"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
	VCS         map[string]interface{} `json:"vcs"`
}

func (c *Client) GetPipelinesByOrg(orgSlug string) (*ProjectPipelinesResponse, error) {
	reqURL := fmt.Sprintf("%s/pipeline?org-slug=%s", baseURL, url.QueryEscape(orgSlug))
	responseBody, err := c.execRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	var response ProjectPipelinesResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response, nil
}

func (c *Client) GetPipeline(pipelineID string) (*PipelineResponse, error) {
	url := fmt.Sprintf("%s/pipeline/%s", baseURL, pipelineID)
	responseBody, err := c.execRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var pipeline PipelineResponse
	err = json.Unmarshal(responseBody, &pipeline)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &pipeline, nil
}

type RunPipelineParams struct {
	DefinitionID string            `json:"definition_id"`
	Config       map[string]string `json:"config"`   // e.g. {"branch": "main"} or {"tag": "v1.0"}
	Checkout     map[string]string `json:"checkout"` // same as config
	Parameters   map[string]string `json:"parameters,omitempty"`
}

type RunPipelineResponse struct {
	ID        string `json:"id"`
	Number    int    `json:"number"`
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
}

func (c *Client) RunPipeline(projectSlug string, params RunPipelineParams) (*RunPipelineResponse, error) {
	path := fmt.Sprintf("%s/project/%s/pipeline/run", baseURL, projectSlug)

	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("error marshaling params: %v", err)
	}

	responseBody, err := c.execRequest("POST", path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response RunPipelineResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response, nil
}

type WorkflowResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	StoppedAt string `json:"stopped_at"`
}

func (c *Client) GetPipelineWorkflows(pipelineID string) ([]WorkflowResponse, error) {
	url := fmt.Sprintf("%s/pipeline/%s/workflow", baseURL, pipelineID)
	responseBody, err := c.execRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Items []WorkflowResponse `json:"items"`
	}
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return response.Items, nil
}

func (c *Client) GetWorkflow(workflowID string) (*WorkflowResponse, error) {
	url := fmt.Sprintf("%s/workflow/%s", baseURL, workflowID)
	responseBody, err := c.execRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var workflow WorkflowResponse
	err = json.Unmarshal(responseBody, &workflow)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &workflow, nil
}

type PipelineStatusResult struct {
	AllDone      bool
	AnyFailed    bool
	Workflows    []WorkflowResponse
	IsErrorState bool
}

func (c *Client) CheckPipelineStatus(pipelineID string) (*PipelineStatusResult, error) {
	workflows, err := c.GetPipelineWorkflows(pipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline workflows: %w", err)
	}

	if len(workflows) == 0 {
		pipeline, err := c.GetPipeline(pipelineID)
		if err != nil {
			return nil, fmt.Errorf("failed to get pipeline: %w", err)
		}

		isErrorState := pipeline.State == "errored"

		return &PipelineStatusResult{
			AllDone:      isErrorState,
			AnyFailed:    isErrorState,
			Workflows:    workflows,
			IsErrorState: isErrorState,
		}, nil
	}

	allDone := true
	anyFailed := false

	for _, w := range workflows {
		// Check if workflow is still running
		if w.Status == "running" || w.Status == "on_hold" || w.Status == "not_run" || w.Status == "failing" {
			allDone = false
		}
		// Check if workflow failed
		if w.Status == "failed" || w.Status == "canceled" || w.Status == "error" || w.Status == "failing" || w.Status == "unauthorized" {
			anyFailed = true
		}
	}

	return &PipelineStatusResult{
		AllDone:      allDone,
		AnyFailed:    anyFailed,
		Workflows:    workflows,
		IsErrorState: false,
	}, nil
}

type WebhookResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type CreateWebhookParams struct {
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Scope  struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"scope"`
	SigningSecret string `json:"signing-secret"`
	VerifyTLS     bool   `json:"verify-tls"`
}

func (c *Client) CreateWebhook(name, webhookURL, secret, projectSlug string, events []string) (*WebhookResponse, error) {
	project, err := c.GetProject(projectSlug)
	if err != nil {
		return nil, fmt.Errorf("fetching project for webhook: %w", err)
	}
	if project.ID == "" {
		return nil, fmt.Errorf("project has no id")
	}

	path := fmt.Sprintf("%s/webhook", baseURL)
	params := CreateWebhookParams{
		Name:          name,
		URL:           webhookURL,
		Events:        events,
		SigningSecret: secret,
		VerifyTLS:     true,
	}
	params.Scope.Type = "project"
	params.Scope.ID = project.ID

	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("error marshaling params: %v", err)
	}

	responseBody, err := c.execRequest("POST", path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error creating webhook: %w", err)
	}

	var webhook WebhookResponse
	if err := json.Unmarshal(responseBody, &webhook); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}
	return &webhook, nil
}

func (c *Client) GetWebhook(webhookID string) (*WebhookResponse, error) {
	url := fmt.Sprintf("%s/webhook/%s", baseURL, webhookID)
	responseBody, err := c.execRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting webhook: %w", err)
	}

	var webhook WebhookResponse
	err = json.Unmarshal(responseBody, &webhook)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &webhook, nil
}

func (c *Client) ListWebhooks(projectSlug string) ([]WebhookResponse, error) {
	project, err := c.GetProject(projectSlug)
	if err != nil {
		return nil, fmt.Errorf("fetching project for webhooks: %w", err)
	}
	if project.ID == "" {
		return nil, fmt.Errorf("project has no id")
	}

	url := fmt.Sprintf("%s/webhook?scope-type=project&scope-id=%s", baseURL, project.ID)
	responseBody, err := c.execRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error listing webhooks: %w", err)
	}

	var response struct {
		Items []WebhookResponse `json:"items"`
	}
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return response.Items, nil
}

func (c *Client) DeleteWebhook(webhookID string) error {
	url := fmt.Sprintf("%s/webhook/%s", baseURL, webhookID)
	_, err := c.execRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error deleting webhook: %v", err)
	}

	return nil
}

type JobResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	StartedAt     string `json:"started_at"`
	StoppedAt     string `json:"stopped_at"`
	JobNumber     int    `json:"job_number"`
	ProjectSlug   string `json:"project_slug"`
	ApprovalReqID string `json:"approval_request_id,omitempty"`
}

func (c *Client) GetWorkflowJobs(workflowID string) ([]JobResponse, error) {
	url := fmt.Sprintf("%s/workflow/%s/job", baseURL, workflowID)
	responseBody, err := c.execRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Items []JobResponse `json:"items"`
	}
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return response.Items, nil
}

type ProjectPipelinesResponse struct {
	Items         []PipelineResponse `json:"items"`
	NextPageToken string             `json:"next_page_token"`
}

func (c *Client) GetProjectPipelinesWithPageToken(projectSlug, branch, pageToken string) (*ProjectPipelinesResponse, error) {
	reqURL := fmt.Sprintf("%s/project/%s/pipeline", baseURL, projectSlug)

	query := url.Values{}
	if branch != "" {
		query.Set("branch", branch)
	}
	if pageToken != "" {
		query.Set("page-token", pageToken)
	}

	if len(query) > 0 {
		reqURL = fmt.Sprintf("%s?%s", reqURL, query.Encode())
	}

	responseBody, err := c.execRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	var response ProjectPipelinesResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response, nil
}

type PipelineDefinitionResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

func (c *Client) GetPipelineDefinitions(projectID string) ([]PipelineDefinitionResponse, error) {
	url := fmt.Sprintf("%s/projects/%s/pipeline-definitions", baseURL, projectID)
	responseBody, err := c.execRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Items []PipelineDefinitionResponse `json:"items"`
	}
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return response.Items, nil
}
