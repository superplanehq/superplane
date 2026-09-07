package vercel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const defaultVercelBaseURL = "https://api.vercel.com"

type Client struct {
	AccessToken string
	TeamID      string
	BaseURL     string
	http        core.HTTPContext
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request failed with %d: %s", e.StatusCode, e.Body)
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
}

type ProjectLink struct {
	Type             string `json:"type,omitempty"`
	Org              string `json:"org,omitempty"`
	Repo             string `json:"repo,omitempty"`
	ProductionBranch string `json:"productionBranch,omitempty"`
}

type Project struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Framework string       `json:"framework,omitempty"`
	CreatedAt int64        `json:"createdAt,omitempty"`
	Link      *ProjectLink `json:"link,omitempty"`
}

type ProjectDomain struct {
	Name     string `json:"name"`
	Verified bool   `json:"verified,omitempty"`
}

type GitSource struct {
	Type string `json:"type"`
	Org  string `json:"org"`
	Repo string `json:"repo"`
	Ref  string `json:"ref"`
}

type CreateDeploymentRequest struct {
	Name string `json:"name"`
	// Target is only sent for production deployments: Vercel rejects
	// target=preview, since omitting it already produces a preview deployment.
	Target    *string    `json:"target,omitempty"`
	GitSource *GitSource `json:"gitSource,omitempty"`
}

type Deployment struct {
	ID         string `json:"id"`
	UID        string `json:"uid,omitempty"`
	URL        string `json:"url,omitempty"`
	Name       string `json:"name,omitempty"`
	ReadyState string `json:"readyState,omitempty"`
	State      string `json:"state,omitempty"`
	Target     string `json:"target,omitempty"`
	ProjectID  string `json:"projectId,omitempty"`
	CreatedAt  int64  `json:"createdAt,omitempty"`
}

// normalize fills ID from UID: list responses use "uid", single-deployment
// responses use "id".
func (d *Deployment) normalize() {
	if d.ID == "" {
		d.ID = d.UID
	}
	if d.ReadyState == "" {
		d.ReadyState = d.State
	}
}

type UpsertEnvVarRequest struct {
	Key     string   `json:"key"`
	Value   string   `json:"value"`
	Type    string   `json:"type"`
	Targets []string `json:"target"`
}

type EnvVar struct {
	Key    string   `json:"key"`
	Type   string   `json:"type,omitempty"`
	Target []string `json:"target,omitempty"`
	EnvID  string   `json:"id,omitempty"`
}

type upsertEnvVarResponse struct {
	Created *EnvVar            `json:"created"`
	Failed  []upsertEnvFailure `json:"failed"`
}

type upsertEnvFailure struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type createProjectRequest struct {
	Name      string `json:"name"`
	Framework string `json:"framework,omitempty"`
}

type addProjectDomainRequest struct {
	Name string `json:"name"`
}

type Webhook struct {
	ID            string   `json:"id"`
	URL           string   `json:"url,omitempty"`
	Secret        string   `json:"secret,omitempty"`
	SigningSecret string   `json:"signingSecret,omitempty"`
	Events        []string `json:"events,omitempty"`
}

// signingSecret returns the webhook signing secret. Vercel returns it once on
// creation; older responses use "secret", newer ones "signingSecret".
func (w *Webhook) signingSecret() string {
	if w.SigningSecret != "" {
		return w.SigningSecret
	}
	return w.Secret
}

type createWebhookRequest struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	token, err := ctx.GetConfig("accessToken")
	if err != nil {
		return nil, err
	}

	tokenValue := strings.TrimSpace(string(token))
	if tokenValue == "" {
		return nil, fmt.Errorf("accessToken is required")
	}

	teamID := ""
	if teamValue, teamErr := ctx.GetConfig("teamId"); teamErr == nil {
		teamID = strings.TrimSpace(string(teamValue))
	}

	return &Client{
		AccessToken: tokenValue,
		TeamID:      teamID,
		BaseURL:     defaultVercelBaseURL,
		http:        httpClient,
	}, nil
}

func (c *Client) GetUser() (*User, error) {
	body, err := c.execRequestWithResponse(http.MethodGet, "/v2/user", nil, nil)
	if err != nil {
		return nil, err
	}

	response := struct {
		User *User `json:"user"`
	}{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user response: %w", err)
	}

	if response.User == nil || response.User.ID == "" {
		return nil, fmt.Errorf("user id is missing in response")
	}

	return response.User, nil
}

func (c *Client) ListProjects(limit int) ([]Project, error) {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(limit))

	body, err := c.execRequestWithResponse(http.MethodGet, "/v9/projects", query, nil)
	if err != nil {
		return nil, err
	}

	response := struct {
		Projects []Project `json:"projects"`
	}{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal projects response: %w", err)
	}

	return response.Projects, nil
}

func (c *Client) GetProject(idOrName string) (*Project, error) {
	if idOrName == "" {
		return nil, fmt.Errorf("project is required")
	}

	body, err := c.execRequestWithResponse(http.MethodGet, "/v9/projects/"+url.PathEscape(idOrName), nil, nil)
	if err != nil {
		return nil, err
	}

	project := Project{}
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project response: %w", err)
	}

	return &project, nil
}

func (c *Client) CreateDeployment(request CreateDeploymentRequest) (*Deployment, error) {
	if request.Name == "" {
		return nil, fmt.Errorf("deployment name is required")
	}
	if request.GitSource == nil {
		return nil, fmt.Errorf("gitSource is required")
	}

	body, err := c.execRequestWithResponse(http.MethodPost, "/v13/deployments", nil, request)
	if err != nil {
		return nil, err
	}

	deployment := Deployment{}
	if err := json.Unmarshal(body, &deployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment response: %w", err)
	}

	return &deployment, nil
}

func (c *Client) GetDeployment(deploymentID string) (*Deployment, error) {
	if deploymentID == "" {
		return nil, fmt.Errorf("deploymentID is required")
	}

	body, err := c.execRequestWithResponse(
		http.MethodGet,
		"/v13/deployments/"+url.PathEscape(deploymentID),
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}

	deployment := Deployment{}
	if err := json.Unmarshal(body, &deployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment response: %w", err)
	}
	deployment.normalize()

	return &deployment, nil
}

// ponytail: first page only; add cursor pagination (pagination.next) when someone hits it
func (c *Client) ListDeployments(projectID string, target string, state string, limit int) ([]Deployment, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	if projectID != "" {
		query.Set("projectId", projectID)
	}
	if target != "" {
		query.Set("target", target)
	}
	if state != "" {
		query.Set("state", state)
	}

	body, err := c.execRequestWithResponse(http.MethodGet, "/v7/deployments", query, nil)
	if err != nil {
		return nil, err
	}

	response := struct {
		Deployments []Deployment `json:"deployments"`
	}{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployments response: %w", err)
	}

	for i := range response.Deployments {
		response.Deployments[i].normalize()
	}

	return response.Deployments, nil
}

func (c *Client) CancelDeployment(deploymentID string) (*Deployment, error) {
	if deploymentID == "" {
		return nil, fmt.Errorf("deploymentID is required")
	}

	body, err := c.execRequestWithResponse(
		http.MethodPatch,
		"/v12/deployments/"+url.PathEscape(deploymentID)+"/cancel",
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}

	deployment := Deployment{}
	if err := json.Unmarshal(body, &deployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment response: %w", err)
	}
	deployment.normalize()

	return &deployment, nil
}

// RollbackProduction points production traffic back to a previous production
// deployment. Vercel returns an empty 201 on success.
func (c *Client) RollbackProduction(projectID string, deploymentID string, description string) error {
	if projectID == "" {
		return fmt.Errorf("projectID is required")
	}
	if deploymentID == "" {
		return fmt.Errorf("deploymentID is required")
	}

	query := url.Values{}
	if description != "" {
		query.Set("description", description)
	}

	_, err := c.execRequestWithResponse(
		http.MethodPost,
		"/v1/projects/"+url.PathEscape(projectID)+"/rollback/"+url.PathEscape(deploymentID),
		query,
		nil,
	)
	return err
}

func (c *Client) CreateProject(name string, framework string) (*Project, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	body, err := c.execRequestWithResponse(http.MethodPost, "/v11/projects", nil, createProjectRequest{
		Name:      name,
		Framework: framework,
	})
	if err != nil {
		return nil, err
	}

	project := Project{}
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project response: %w", err)
	}

	return &project, nil
}

func (c *Client) UpsertEnvVar(project string, request UpsertEnvVarRequest) (*EnvVar, error) {
	if project == "" {
		return nil, fmt.Errorf("project is required")
	}

	query := url.Values{}
	query.Set("upsert", "true")

	body, err := c.execRequestWithResponse(
		http.MethodPost,
		"/v10/projects/"+url.PathEscape(project)+"/env",
		query,
		request,
	)
	if err != nil {
		return nil, err
	}

	response := upsertEnvVarResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal env var response: %w", err)
	}

	if len(response.Failed) > 0 {
		failure := response.Failed[0]
		return nil, fmt.Errorf("failed to set environment variable %s: %s", request.Key, failure.Error.Message)
	}

	return response.Created, nil
}

func (c *Client) AddProjectDomain(project string, name string) (*ProjectDomain, error) {
	if project == "" {
		return nil, fmt.Errorf("project is required")
	}
	if name == "" {
		return nil, fmt.Errorf("domain is required")
	}

	body, err := c.execRequestWithResponse(
		http.MethodPost,
		"/v10/projects/"+url.PathEscape(project)+"/domains",
		nil,
		addProjectDomainRequest{Name: name},
	)
	if err != nil {
		return nil, err
	}

	domain := ProjectDomain{}
	if err := json.Unmarshal(body, &domain); err != nil {
		return nil, fmt.Errorf("failed to unmarshal project domain response: %w", err)
	}

	return &domain, nil
}

func (c *Client) RemoveProjectDomain(project string, name string) error {
	if project == "" {
		return fmt.Errorf("project is required")
	}
	if name == "" {
		return fmt.Errorf("domain is required")
	}

	_, err := c.execRequestWithResponse(
		http.MethodDelete,
		"/v9/projects/"+url.PathEscape(project)+"/domains/"+url.PathEscape(name),
		nil,
		nil,
	)
	return err
}

func (c *Client) CreateWebhook(request createWebhookRequest) (*Webhook, error) {
	if request.URL == "" {
		return nil, fmt.Errorf("url is required")
	}
	if len(request.Events) == 0 {
		return nil, fmt.Errorf("events are required")
	}

	body, err := c.execRequestWithResponse(http.MethodPost, "/v1/webhooks", nil, request)
	if err != nil {
		return nil, err
	}

	return parseWebhook(body)
}

func (c *Client) DeleteWebhook(webhookID string) error {
	if webhookID == "" {
		return fmt.Errorf("webhookID is required")
	}

	_, err := c.execRequestWithResponse(
		http.MethodDelete,
		"/v1/webhooks/"+url.PathEscape(webhookID),
		nil,
		nil,
	)
	return err
}

func parseWebhook(body []byte) (*Webhook, error) {
	webhook := Webhook{}
	if err := json.Unmarshal(body, &webhook); err == nil && webhook.ID != "" {
		return &webhook, nil
	}

	wrapper := struct {
		Webhook Webhook `json:"webhook"`
	}{}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook response: %w", err)
	}

	if wrapper.Webhook.ID == "" {
		return nil, fmt.Errorf("webhook id is missing in response")
	}

	return &wrapper.Webhook, nil
}

func (c *Client) execRequestWithResponse(
	method string,
	path string,
	query url.Values,
	payload any,
) ([]byte, error) {
	if c.TeamID != "" {
		if query == nil {
			query = url.Values{}
		}
		query.Set("teamId", c.TeamID)
	}

	endpoint := c.BaseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var body io.Reader
	if payload != nil {
		encodedBody, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewReader(encodedBody)
	}

	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, &APIError{StatusCode: res.StatusCode, Body: string(responseBody)}
	}

	return responseBody, nil
}
