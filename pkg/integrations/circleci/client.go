package circleci

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

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

// UserResponse represents the current user information from GET /me.
// The projects field is keyed by VCS URL (e.g. https://github.com/org/repo).
type UserResponse struct {
	ID       string         `json:"id"`
	Login    string         `json:"login"`
	Name     string         `json:"name"`
	Projects map[string]any `json:"projects,omitempty"`
}

// GetCurrentUser verifies the API token by fetching current user info
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

// slugFromVCSURL converts a CircleCI VCS URL (from /me projects keys) to a v2 project slug.
// Example: "https://github.com/org/repo" -> "gh/org/repo"
func slugFromVCSURL(vcsURL string) (slug, repoName string, ok bool) {
	parsed, err := url.Parse(vcsURL)
	if err != nil || parsed.Host == "" || parsed.Path == "" {
		return "", "", false
	}
	path := strings.TrimSuffix(strings.Trim(parsed.Path, "/"), ".git")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", false
	}
	org := parts[0]
	repo := strings.Join(parts[1:], "/")
	var vcsType string
	switch {
	case strings.Contains(parsed.Host, "github"):
		vcsType = "gh"
	case strings.Contains(parsed.Host, "bitbucket"):
		vcsType = "bb"
	case strings.Contains(parsed.Host, "gitlab"):
		vcsType = "gl"
	default:
		vcsType = "gh"
	}
	return fmt.Sprintf("%s/%s/%s", vcsType, org, repo), repo, true
}

// ListProjects returns projects from the v2 /me endpoint (projects key is VCS URL -> project info).
func (c *Client) ListProjects() ([]struct{ Slug, Name string }, error) {
	user, err := c.GetCurrentUser()
	if err != nil {
		return nil, err
	}
	if user.Projects == nil {
		return []struct{ Slug, Name string }{}, nil
	}
	var out []struct{ Slug, Name string }
	for vcsURL := range user.Projects {
		slug, name, ok := slugFromVCSURL(vcsURL)
		if !ok {
			continue
		}
		out = append(out, struct{ Slug, Name string }{Slug: slug, Name: name})
	}
	return out, nil
}

// ProjectResponse represents project information
type ProjectResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// GetProject fetches project details by project slug.
// CircleCI v2 API expects the slug as path segments: /project/gh/org/repo (no encoding).
func (c *Client) GetProject(projectSlug string) (*ProjectResponse, error) {
	projectSlug = normalizeProjectSlug(projectSlug)
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
	projectSlug = normalizeProjectSlug(projectSlug)
	path := fmt.Sprintf("%s/project/%s/pipeline", baseURL, projectSlug)

	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("error marshaling params: %v", err)
	}

	responseBody, err := c.execRequest("POST", path, bytes.NewReader(body))
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

// TriggerPipelineRunParams is the body for the pipeline/run API (GitHub App and Bitbucket DC).
type TriggerPipelineRunParams struct {
	DefinitionID string            `json:"definition_id"`
	Config       map[string]string `json:"config"`   // e.g. {"branch": "main"} or {"tag": "v1.0"}
	Checkout     map[string]string `json:"checkout"` // same as config
	Parameters   map[string]string `json:"parameters,omitempty"`
}

// TriggerPipelineRun triggers a pipeline via the pipeline/run endpoint.
// Use this for projects connected via GitHub App or Bitbucket Data Center.
// See https://circleci.com/docs/triggers-overview/#run-a-pipeline-using-the-api
func (c *Client) TriggerPipelineRun(projectSlug string, params TriggerPipelineRunParams) (*TriggerPipelineResponse, error) {
	projectSlug = normalizeProjectSlug(projectSlug)
	path := fmt.Sprintf("%s/project/%s/pipeline/run", baseURL, projectSlug)

	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("error marshaling params: %v", err)
	}

	responseBody, err := c.execRequest("POST", path, bytes.NewReader(body))
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

// GetWorkflow fetches workflow details by ID
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

// CreateWebhook creates a new CircleCI webhook for the given project slug.
// Resolves the slug to the project UUID because CircleCI requires scope.id to be a UUID.
func (c *Client) CreateWebhook(name, webhookURL, secret, projectSlug string, events []string) (*WebhookResponse, error) {
	projectSlug = normalizeProjectSlug(projectSlug)
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

// DeleteWebhook deletes a webhook by ID
func (c *Client) DeleteWebhook(webhookID string) error {
	url := fmt.Sprintf("%s/webhook/%s", baseURL, webhookID)
	_, err := c.execRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error deleting webhook: %v", err)
	}

	return nil
}

// uuidLike matches CircleCI org/project UUIDs (e.g. 18ec629d-27b4-460d-ad93-5f9195cfe719).
var uuidLike = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// normalizeProjectSlug rewrites gh/uuid/uuid to circleci/uuid/uuid so GitHub App–connected
// projects work when the stored slug was built with the wrong vcs type.
func normalizeProjectSlug(slug string) string {
	parts := strings.SplitN(slug, "/", 3)
	if len(parts) != 3 || parts[0] != "gh" {
		return slug
	}
	if uuidLike.MatchString(strings.ToLower(parts[1])) && uuidLike.MatchString(strings.ToLower(parts[2])) {
		return "circleci/" + parts[1] + "/" + parts[2]
	}
	return slug
}
