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

const BaseURL = "https://circleci.com/api/v2"

type Client struct {
	APIToken string
	http     core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, err
	}

	return &Client{
		APIToken: string(apiToken),
		http:     http,
	}, nil
}

func (c *Client) execRequest(method, URL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Circle-Token", c.APIToken)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// User represents the current user
type User struct {
	ID    string `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
}

func (c *Client) GetCurrentUser() (*User, error) {
	URL := fmt.Sprintf("%s/me", BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var user User
	err = json.Unmarshal(responseBody, &user)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &user, nil
}

// Project represents a CircleCI project
type Project struct {
	ID               string `json:"id"`
	Slug             string `json:"slug"`
	Name             string `json:"name"`
	OrganizationName string `json:"organization_name"`
	OrganizationID   string `json:"organization_id"`
	VCSInfo          struct {
		VCSUrl        string `json:"vcs_url"`
		Provider      string `json:"provider"`
		DefaultBranch string `json:"default_branch"`
	} `json:"vcs_info"`
}

func (c *Client) GetProject(slug string) (*Project, error) {
	URL := fmt.Sprintf("%s/project/%s", BaseURL, url.PathEscape(slug))
	responseBody, err := c.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var project Project
	err = json.Unmarshal(responseBody, &project)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &project, nil
}

// ListProjectsResponse represents the response from listing projects
type ListProjectsResponse struct {
	Items         []Project `json:"items"`
	NextPageToken string    `json:"next_page_token"`
}

func (c *Client) ListProjects() ([]Project, error) {
	URL := fmt.Sprintf("%s/me/collaborations", BaseURL)
	responseBody, err := c.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var collaborations []struct {
		VCSType string `json:"vcs_type"`
		Name    string `json:"name"`
		ID      string `json:"id"`
		Slug    string `json:"slug"`
	}
	err = json.Unmarshal(responseBody, &collaborations)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	// Get projects for each collaboration
	var projects []Project
	for _, collab := range collaborations {
		// Fetch projects for this organization
		orgProjects, err := c.listProjectsForOrg(collab.Slug)
		if err != nil {
			continue // Skip orgs we can't access
		}
		projects = append(projects, orgProjects...)
	}

	return projects, nil
}

func (c *Client) listProjectsForOrg(orgSlug string) ([]Project, error) {
	URL := fmt.Sprintf("%s/me/projects?org-slug=%s", BaseURL, url.QueryEscape(orgSlug))
	responseBody, err := c.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var projects []Project
	err = json.Unmarshal(responseBody, &projects)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return projects, nil
}

// Pipeline represents a CircleCI pipeline
type Pipeline struct {
	ID          string `json:"id"`
	Number      int    `json:"number"`
	State       string `json:"state"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	ProjectSlug string `json:"project_slug"`
	Trigger     struct {
		Type       string `json:"type"`
		ReceivedAt string `json:"received_at"`
		Actor      struct {
			Login     string `json:"login"`
			AvatarURL string `json:"avatar_url"`
		} `json:"actor"`
	} `json:"trigger"`
	VCS struct {
		ProviderName        string `json:"provider_name"`
		OriginRepositoryURL string `json:"origin_repository_url"`
		TargetRepositoryURL string `json:"target_repository_url"`
		Revision            string `json:"revision"`
		Branch              string `json:"branch"`
		Tag                 string `json:"tag"`
	} `json:"vcs,omitempty"`
}

type TriggerPipelineRequest struct {
	Branch     string         `json:"branch,omitempty"`
	Tag        string         `json:"tag,omitempty"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

type TriggerPipelineResponse struct {
	ID        string `json:"id"`
	Number    int    `json:"number"`
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
}

func (c *Client) TriggerPipeline(projectSlug string, request *TriggerPipelineRequest) (*TriggerPipelineResponse, error) {
	URL := fmt.Sprintf("%s/project/%s/pipeline", BaseURL, url.PathEscape(projectSlug))

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response TriggerPipelineResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &response, nil
}

func (c *Client) GetPipeline(pipelineID string) (*Pipeline, error) {
	URL := fmt.Sprintf("%s/pipeline/%s", BaseURL, pipelineID)
	responseBody, err := c.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var pipeline Pipeline
	err = json.Unmarshal(responseBody, &pipeline)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &pipeline, nil
}

// Workflow represents a CircleCI workflow
type Workflow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	StoppedAt   string `json:"stopped_at,omitempty"`
	PipelineID  string `json:"pipeline_id"`
	ProjectSlug string `json:"project_slug"`
}

type ListWorkflowsResponse struct {
	Items         []Workflow `json:"items"`
	NextPageToken string     `json:"next_page_token"`
}

func (c *Client) ListPipelineWorkflows(pipelineID string) ([]Workflow, error) {
	URL := fmt.Sprintf("%s/pipeline/%s/workflow", BaseURL, pipelineID)
	responseBody, err := c.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var response ListWorkflowsResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return response.Items, nil
}

func (c *Client) GetWorkflow(workflowID string) (*Workflow, error) {
	URL := fmt.Sprintf("%s/workflow/%s", BaseURL, workflowID)
	responseBody, err := c.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var workflow Workflow
	err = json.Unmarshal(responseBody, &workflow)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &workflow, nil
}

// Webhook represents a CircleCI webhook
type Webhook struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	URL           string       `json:"url"`
	Events        []string     `json:"events"`
	Scope         WebhookScope `json:"scope"`
	VerifyTLS     bool         `json:"verify-tls"`
	SigningSecret string       `json:"signing-secret,omitempty"`
	CreatedAt     string       `json:"created-at,omitempty"`
	UpdatedAt     string       `json:"updated-at,omitempty"`
}

type WebhookScope struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type CreateWebhookRequest struct {
	Name          string       `json:"name"`
	URL           string       `json:"url"`
	Secret        string       `json:"secret,omitempty"`
	Events        []string     `json:"events"`
	Scope         WebhookScope `json:"scope"`
	VerifyTLS     bool         `json:"verify-tls"`
	SigningSecret string       `json:"signing-secret"`
}

type ListWebhooksResponse struct {
	Items         []Webhook `json:"items"`
	NextPageToken string    `json:"next_page_token"`
}

func (c *Client) CreateWebhook(request *CreateWebhookRequest) (*Webhook, error) {
	URL := fmt.Sprintf("%s/webhook", BaseURL)

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	err = json.Unmarshal(responseBody, &webhook)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &webhook, nil
}

func (c *Client) DeleteWebhook(webhookID string) error {
	URL := fmt.Sprintf("%s/webhook/%s", BaseURL, webhookID)
	_, err := c.execRequest(http.MethodDelete, URL, nil)
	return err
}

func (c *Client) ListWebhooks(projectID string) ([]Webhook, error) {
	URL := fmt.Sprintf("%s/webhook?scope-id=%s&scope-type=project", BaseURL, projectID)
	responseBody, err := c.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var response ListWebhooksResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return response.Items, nil
}
