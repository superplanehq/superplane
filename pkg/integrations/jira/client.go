package jira

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	AuthType string
	// For API Token auth
	Email string
	Token string
	// For OAuth auth
	AccessToken string
	CloudID     string
	// Common
	BaseURL string
	http    core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	authType, _ := ctx.GetConfig("authType")

	if string(authType) == AuthTypeOAuth {
		return newOAuthClient(httpCtx, ctx)
	}

	return newAPITokenClient(httpCtx, ctx)
}

func newAPITokenClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	baseURL, err := ctx.GetConfig("baseUrl")
	if err != nil {
		return nil, fmt.Errorf("error getting baseUrl: %v", err)
	}

	email, err := ctx.GetConfig("email")
	if err != nil {
		return nil, fmt.Errorf("error getting email: %v", err)
	}

	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("error getting apiToken: %v", err)
	}

	return &Client{
		AuthType: AuthTypeAPIToken,
		Email:    string(email),
		Token:    string(apiToken),
		BaseURL:  string(baseURL),
		http:     httpCtx,
	}, nil
}

func newOAuthClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	accessToken, err := findOAuthSecret(ctx, OAuthAccessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting access token: %v", err)
	}

	if accessToken == "" {
		return nil, fmt.Errorf("OAuth access token not found")
	}

	metadata := ctx.GetMetadata()
	metadataMap, ok := metadata.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid metadata format")
	}

	cloudID, _ := metadataMap["cloudId"].(string)
	if cloudID == "" {
		return nil, fmt.Errorf("cloud ID not found in metadata")
	}

	return &Client{
		AuthType:    AuthTypeOAuth,
		AccessToken: accessToken,
		CloudID:     cloudID,
		http:        httpCtx,
	}, nil
}

func (c *Client) authHeader() string {
	if c.AuthType == AuthTypeOAuth {
		return fmt.Sprintf("Bearer %s", c.AccessToken)
	}
	credentials := fmt.Sprintf("%s:%s", c.Email, c.Token)
	return fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(credentials)))
}

func (c *Client) apiURL(path string) string {
	var url string
	if c.AuthType == AuthTypeOAuth {
		url = fmt.Sprintf("https://api.atlassian.com/ex/jira/%s%s", c.CloudID, path)
	} else {
		url = fmt.Sprintf("%s%s", c.BaseURL, path)
	}
	return url
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.authHeader())

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// User represents a Jira user from the /myself endpoint.
type User struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	EmailAddr   string `json:"emailAddress"`
}

// GetCurrentUser verifies credentials by fetching the authenticated user.
func (c *Client) GetCurrentUser() (*User, error) {
	url := c.apiURL("/rest/api/3/myself")
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(responseBody, &user); err != nil {
		return nil, fmt.Errorf("error parsing user response: %v", err)
	}

	return &user, nil
}

// Project represents a Jira project.
type Project struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// ListProjects returns all projects accessible to the authenticated user.
func (c *Client) ListProjects() ([]Project, error) {
	url := c.apiURL("/rest/api/3/project")
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var projects []Project
	if err := json.Unmarshal(responseBody, &projects); err != nil {
		return nil, fmt.Errorf("error parsing projects response: %v", err)
	}

	return projects, nil
}

// Issue represents a Jira issue.
type Issue struct {
	ID     string         `json:"id"`
	Key    string         `json:"key"`
	Self   string         `json:"self"`
	Fields map[string]any `json:"fields"`
}

// GetIssue fetches a single issue by its key.
func (c *Client) GetIssue(issueKey string) (*Issue, error) {
	url := c.apiURL(fmt.Sprintf("/rest/api/3/issue/%s", issueKey))
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(responseBody, &issue); err != nil {
		return nil, fmt.Errorf("error parsing issue response: %v", err)
	}

	return &issue, nil
}

// CreateIssueRequest is the request body for creating an issue.
type CreateIssueRequest struct {
	Fields CreateIssueFields `json:"fields"`
}

// CreateIssueFields contains the fields for creating an issue.
type CreateIssueFields struct {
	Project     ProjectRef `json:"project"`
	IssueType   IssueType  `json:"issuetype"`
	Summary     string     `json:"summary"`
	Description *ADFDoc    `json:"description,omitempty"`
}

// ProjectRef references a project by key.
type ProjectRef struct {
	Key string `json:"key"`
}

// IssueType specifies the issue type by name.
type IssueType struct {
	Name string `json:"name"`
}

// ADFDoc represents an Atlassian Document Format document.
type ADFDoc struct {
	Type    string    `json:"type"`
	Version int       `json:"version"`
	Content []ADFNode `json:"content"`
}

// ADFNode represents a node in an ADF document.
type ADFNode struct {
	Type    string    `json:"type"`
	Content []ADFText `json:"content,omitempty"`
}

// ADFText represents a text node in ADF.
type ADFText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// WrapInADF wraps plain text in a minimal ADF document.
func WrapInADF(text string) *ADFDoc {
	if text == "" {
		return nil
	}
	return &ADFDoc{
		Type:    "doc",
		Version: 1,
		Content: []ADFNode{
			{
				Type: "paragraph",
				Content: []ADFText{
					{Type: "text", Text: text},
				},
			},
		},
	}
}

// CreateIssueResponse is the response from creating an issue.
type CreateIssueResponse struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

// CreateIssue creates a new issue in Jira.
func (c *Client) CreateIssue(req *CreateIssueRequest) (*CreateIssueResponse, error) {
	url := c.apiURL("/rest/api/3/issue")

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response CreateIssueResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing create issue response: %v", err)
	}

	return &response, nil
}

// WebhookRegistrationRequest is the request body for registering webhooks.
type WebhookRegistrationRequest struct {
	URL      string        `json:"url"`
	Webhooks []WebhookSpec `json:"webhooks"`
}

// WebhookSpec defines a single webhook configuration.
type WebhookSpec struct {
	JQLFilter string   `json:"jqlFilter"`
	Events    []string `json:"events"`
}

// WebhookRegistrationResponse is the response from registering webhooks.
type WebhookRegistrationResponse struct {
	WebhookRegistrationResult []WebhookRegistrationResult `json:"webhookRegistrationResult"`
}

// WebhookRegistrationResult contains the result of a single webhook registration.
type WebhookRegistrationResult struct {
	CreatedWebhookID int64    `json:"createdWebhookId"`
	Errors           []string `json:"errors,omitempty"`
}

// RegisterWebhook registers a new webhook in Jira.
func (c *Client) RegisterWebhook(webhookURL, jqlFilter string, events []string) (*WebhookRegistrationResponse, error) {
	url := c.apiURL("/rest/api/3/webhook")

	req := WebhookRegistrationRequest{
		URL: webhookURL,
		Webhooks: []WebhookSpec{
			{
				JQLFilter: jqlFilter,
				Events:    events,
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response WebhookRegistrationResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing webhook registration response: %v", err)
	}

	return &response, nil
}

// FailedWebhookResponse is the response from the failed webhooks endpoint.
type FailedWebhookResponse struct {
	Values []FailedWebhook `json:"values"`
}

// FailedWebhook contains information about a failed webhook delivery.
type FailedWebhook struct {
	ID                string `json:"id"`
	Body              string `json:"body"`
	URL               string `json:"url"`
	FailureReason     string `json:"failureReason"`
	LatestFailureTime string `json:"latestFailureTime"`
}

// GetFailedWebhooks returns webhooks that failed to be delivered in the last 72 hours.
func (c *Client) GetFailedWebhooks() (*FailedWebhookResponse, error) {
	url := c.apiURL("/rest/api/3/webhook/failed")
	responseBody, err := c.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response FailedWebhookResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing failed webhooks response: %v", err)
	}

	return &response, nil
}

// WebhookDeletionRequest is the request body for deleting webhooks.
type WebhookDeletionRequest struct {
	WebhookIDs []int64 `json:"webhookIds"`
}

// DeleteWebhook removes webhooks from Jira.
func (c *Client) DeleteWebhook(webhookIDs []int64) error {
	url := c.apiURL("/rest/api/3/webhook")

	req := WebhookDeletionRequest{
		WebhookIDs: webhookIDs,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	_, err = c.execRequest(http.MethodDelete, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	return nil
}
