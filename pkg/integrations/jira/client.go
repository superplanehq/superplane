package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	Token    string
	BaseURL  string
	AuthType string
	http     core.HTTPContext
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request got %d code: %s", e.StatusCode, e.Body)
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	return newOAuthClientFromIntegration(httpCtx, ctx)
}

func NewOAuthClient(httpCtx core.HTTPContext, accessToken, cloudID string) *Client {
	return &Client{
		Token:    accessToken,
		BaseURL:  fmt.Sprintf("https://api.atlassian.com/ex/jira/%s", url.PathEscape(cloudID)),
		AuthType: AuthTypeOAuth,
		http:     httpCtx,
	}
}

func newOAuthClientFromIntegration(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	accessToken, err := requireOAuthSecret(ctx, OAuthAccessToken)
	if err != nil {
		return nil, err
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode Jira metadata: %w", err)
	}

	if metadata.CloudID == "" {
		return nil, fmt.Errorf("Jira cloud ID is missing: integration needs to sync")
	}

	return NewOAuthClient(httpCtx, accessToken, metadata.CloudID), nil
}

func (c *Client) authHeader() string {
	return "Bearer " + c.Token
}

func (c *Client) execRequest(method, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, c.url(path), body)
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
		return nil, &APIError{StatusCode: res.StatusCode, Body: string(responseBody)}
	}

	return responseBody, nil
}

func (c *Client) url(path string) string {
	return fmt.Sprintf("%s%s", c.BaseURL, path)
}

// User represents a Jira user from the /myself endpoint.
type User struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	EmailAddr   string `json:"emailAddress"`
}

// GetCurrentUser verifies credentials by fetching the authenticated user.
func (c *Client) GetCurrentUser() (*User, error) {
	responseBody, err := c.execRequest(http.MethodGet, "/rest/api/3/myself", nil)
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
	responseBody, err := c.execRequest(http.MethodGet, "/rest/api/3/project", nil)
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
	responseBody, err := c.execRequest(http.MethodGet, fmt.Sprintf("/rest/api/3/issue/%s", url.PathEscape(issueKey)), nil)
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
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, "/rest/api/3/issue", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response CreateIssueResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing create issue response: %v", err)
	}

	return &response, nil
}

type CreateWebhookRequest struct {
	URL      string                `json:"url"`
	Webhooks []WebhookRegistration `json:"webhooks"`
}

type WebhookRegistration struct {
	Events    []string `json:"events"`
	JQLFilter string   `json:"jqlFilter"`
}

type CreateWebhookResponse struct {
	WebhookRegistrationResult []WebhookRegistrationResult `json:"webhookRegistrationResult"`
}

type WebhookRegistrationResult struct {
	CreatedWebhookID int64    `json:"createdWebhookId"`
	Errors           []string `json:"errors,omitempty"`
}

type DeleteWebhookRequest struct {
	WebhookIDs []int64 `json:"webhookIds"`
}

func (c *Client) CreateWebhook(req CreateWebhookRequest) (*CreateWebhookResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling webhook request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, "/rest/api/3/webhook", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	response := CreateWebhookResponse{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing create webhook response: %v", err)
	}

	return &response, nil
}

func (c *Client) DeleteWebhook(webhookID int64) error {
	body, err := json.Marshal(DeleteWebhookRequest{WebhookIDs: []int64{webhookID}})
	if err != nil {
		return fmt.Errorf("error marshaling delete webhook request: %v", err)
	}

	_, err = c.execRequest(http.MethodDelete, "/rest/api/3/webhook", bytes.NewReader(body))
	return err
}

type listWebhooksPage struct {
	IsLast bool      `json:"isLast"`
	Values []Webhook `json:"values"`
}

type Webhook struct {
	ID int64 `json:"id"`
}

type RefreshWebhookRequest struct {
	WebhookIDs []int64 `json:"webhookIds"`
}

type RefreshWebhookResponse struct {
	ExpirationDate string `json:"expirationDate"`
}

// ListWebhooks returns every webhook visible to this OAuth app context.
// Pagination follows Jira's standard maxResults/startAt scheme.
func (c *Client) ListWebhooks() ([]Webhook, error) {
	var all []Webhook
	startAt := 0
	for {
		path := fmt.Sprintf("/rest/api/3/webhook?startAt=%d&maxResults=100", startAt)
		body, err := c.execRequest(http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		page := listWebhooksPage{}
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("error parsing webhooks response: %v", err)
		}

		all = append(all, page.Values...)
		if page.IsLast || len(page.Values) == 0 {
			return all, nil
		}

		startAt += len(page.Values)
	}
}

// RefreshWebhooks extends the expiration of the given webhooks by 30 days.
// Jira accepts at most 100 webhook IDs per call.
func (c *Client) RefreshWebhooks(ids []int64) (*RefreshWebhookResponse, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(RefreshWebhookRequest{WebhookIDs: ids})
	if err != nil {
		return nil, fmt.Errorf("error marshaling refresh webhook request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPut, "/rest/api/3/webhook/refresh", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	response := RefreshWebhookResponse{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing refresh webhook response: %v", err)
	}

	return &response, nil
}
