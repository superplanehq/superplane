package circleci

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const baseURL = "https://circleci.com/api/v2"

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

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
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

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// UserResponse represents the current user information
type UserResponse struct {
	ID    string `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
}

// GetCurrentUser verifies the API token by fetching current user info
func (c *Client) GetCurrentUser() (*UserResponse, error) {
	url := fmt.Sprintf("%s/me", baseURL)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
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

// ProjectResponse represents project information
type ProjectResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// GetProject fetches project details by project slug
func (c *Client) GetProject(projectSlug string) (*ProjectResponse, error) {
	url := fmt.Sprintf("%s/project/%s", baseURL, projectSlug)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
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

// PipelineResponse represents pipeline information
type PipelineResponse struct {
	ID        string                 `json:"id"`
	Number    int                    `json:"number"`
	State     string                 `json:"state"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
	VCS       map[string]interface{} `json:"vcs"`
}

// GetPipeline fetches pipeline details by ID
func (c *Client) GetPipeline(pipelineID string) (*PipelineResponse, error) {
	url := fmt.Sprintf("%s/pipeline/%s", baseURL, pipelineID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
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

// TriggerPipelineParams represents parameters for triggering a pipeline
type TriggerPipelineParams struct {
	Branch     string            `json:"branch,omitempty"`
	Tag        string            `json:"tag,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// TriggerPipelineResponse represents the response from triggering a pipeline
type TriggerPipelineResponse struct {
	ID        string `json:"id"`
	Number    int    `json:"number"`
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
}

// TriggerPipeline triggers a new pipeline for a project
func (c *Client) TriggerPipeline(projectSlug string, params TriggerPipelineParams) (*TriggerPipelineResponse, error) {
	url := fmt.Sprintf("%s/project/%s/pipeline", baseURL, projectSlug)

	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("error marshaling params: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
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

// WorkflowResponse represents workflow information
type WorkflowResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	StoppedAt string `json:"stopped_at"`
}

// GetPipelineWorkflows fetches all workflows for a pipeline
func (c *Client) GetPipelineWorkflows(pipelineID string) ([]WorkflowResponse, error) {
	url := fmt.Sprintf("%s/pipeline/%s/workflow", baseURL, pipelineID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
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

// GetWorkflow fetches workflow details by ID
func (c *Client) GetWorkflow(workflowID string) (*WorkflowResponse, error) {
	url := fmt.Sprintf("%s/workflow/%s", baseURL, workflowID)
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
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

// WebhookResponse represents webhook information
type WebhookResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// CreateWebhookParams represents parameters for creating a webhook
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

// CreateWebhook creates a new CircleCI webhook
func (c *Client) CreateWebhook(name, url, secret, projectSlug string, events []string) (*WebhookResponse, error) {
	webhookURL := fmt.Sprintf("%s/webhook", baseURL)

	// Get project ID (UUID) from slug
	project, err := c.GetProject(projectSlug)
	if err != nil {
		return nil, fmt.Errorf("error getting project ID: %v", err)
	}

	params := CreateWebhookParams{
		Name:          name,
		URL:           url,
		Events:        events,
		SigningSecret: secret,
		VerifyTLS:     true,
	}

	// Set scope to project with UUID (not slug!)
	params.Scope.Type = "project"
	params.Scope.ID = project.ID

	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("error marshaling params: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var webhook WebhookResponse
	err = json.Unmarshal(responseBody, &webhook)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	return &webhook, nil
}

// DeleteWebhook deletes a webhook by ID
func (c *Client) DeleteWebhook(webhookID string) error {
	url := fmt.Sprintf("%s/webhook/%s", baseURL, webhookID)
	_, err := c.execRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("error deleting webhook: %v", err)
	}

	return nil
}
