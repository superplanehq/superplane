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

// Client speaks to Jira Cloud through Atlassian's OAuth proxy. All endpoints
// resolve to `https://api.atlassian.com/ex/jira/{cloudId}/rest/api/3/...`.
type Client struct {
	Token   string
	CloudID string
	SiteURL string
	http    core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	token, err := findSecret(ctx, OAuthAccessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting access token: %v", err)
	}
	if token == "" {
		return nil, fmt.Errorf("missing access token")
	}

	metadata := Metadata{}
	if err := mapstructure.Decode(ctx.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("error decoding integration metadata: %v", err)
	}

	if metadata.CloudID == "" {
		return nil, fmt.Errorf("missing Jira cloud ID — re-authorize the integration")
	}

	return &Client{
		Token:   token,
		CloudID: metadata.CloudID,
		SiteURL: metadata.SiteURL,
		http:    httpCtx,
	}, nil
}

func (c *Client) apiURL(path string) string {
	return fmt.Sprintf("%s/ex/jira/%s%s", APIBaseURL, url.PathEscape(c.CloudID), path)
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

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

// User represents a Jira user.
type User struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	EmailAddr   string `json:"emailAddress,omitempty"`
}

// GetCurrentUser fetches the authenticated user via /rest/api/3/myself.
func (c *Client) GetCurrentUser() (*User, error) {
	body, err := c.execRequest(http.MethodGet, c.apiURL("/rest/api/3/myself"), nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
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
	body, err := c.execRequest(http.MethodGet, c.apiURL("/rest/api/3/project"), nil)
	if err != nil {
		return nil, err
	}

	var projects []Project
	if err := json.Unmarshal(body, &projects); err != nil {
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

// GetIssueOptions allows callers to request additional fields/expansions.
type GetIssueOptions struct {
	Fields string
	Expand string
}

// GetIssue fetches a single issue by its key.
func (c *Client) GetIssue(issueKey string) (*Issue, error) {
	return c.GetIssueWithOptions(issueKey, GetIssueOptions{})
}

func (c *Client) GetIssueWithOptions(issueKey string, opts GetIssueOptions) (*Issue, error) {
	endpoint := c.apiURL("/rest/api/3/issue/" + url.PathEscape(issueKey))

	query := url.Values{}
	if opts.Fields != "" {
		query.Set("fields", opts.Fields)
	}
	if opts.Expand != "" {
		query.Set("expand", opts.Expand)
	}
	if len(query) > 0 {
		endpoint = endpoint + "?" + query.Encode()
	}

	body, err := c.execRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("error parsing issue response: %v", err)
	}
	return &issue, nil
}

// CreateIssueRequest is the request body for creating an issue.
type CreateIssueRequest struct {
	Fields CreateIssueFields `json:"fields"`
}

type CreateIssueFields struct {
	Project     ProjectRef `json:"project"`
	IssueType   IssueType  `json:"issuetype"`
	Summary     string     `json:"summary"`
	Description *ADFDoc    `json:"description,omitempty"`
}

type ProjectRef struct {
	Key string `json:"key"`
}

type IssueType struct {
	Name string `json:"name"`
}

type ADFDoc struct {
	Type    string    `json:"type"`
	Version int       `json:"version"`
	Content []ADFNode `json:"content"`
}

type ADFNode struct {
	Type    string    `json:"type"`
	Content []ADFText `json:"content,omitempty"`
}

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
				Type:    "paragraph",
				Content: []ADFText{{Type: "text", Text: text}},
			},
		},
	}
}

type CreateIssueResponse struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

// CreateIssue creates a new issue.
func (c *Client) CreateIssue(req *CreateIssueRequest) (*CreateIssueResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, c.apiURL("/rest/api/3/issue"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response CreateIssueResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing create issue response: %v", err)
	}
	return &response, nil
}

// UpdateIssueRequest is the request body for updating an issue.
type UpdateIssueRequest struct {
	Fields map[string]any `json:"fields,omitempty"`
}

type UpdateIssueOptions struct {
	NotifyUsers *bool
}

// UpdateIssue updates an existing issue. Jira returns 204 No Content on success.
func (c *Client) UpdateIssue(issueKey string, req *UpdateIssueRequest, opts UpdateIssueOptions) error {
	endpoint := c.apiURL("/rest/api/3/issue/" + url.PathEscape(issueKey))

	query := url.Values{}
	if opts.NotifyUsers != nil {
		if *opts.NotifyUsers {
			query.Set("notifyUsers", "true")
		} else {
			query.Set("notifyUsers", "false")
		}
	}
	if len(query) > 0 {
		endpoint = endpoint + "?" + query.Encode()
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	if _, err := c.execRequest(http.MethodPut, endpoint, bytes.NewReader(body)); err != nil {
		return err
	}
	return nil
}

type DeleteIssueOptions struct {
	DeleteSubtasks bool
}

// DeleteIssue removes an issue. Jira returns 204 No Content on success.
func (c *Client) DeleteIssue(issueKey string, opts DeleteIssueOptions) error {
	endpoint := c.apiURL("/rest/api/3/issue/" + url.PathEscape(issueKey))

	query := url.Values{}
	if opts.DeleteSubtasks {
		query.Set("deleteSubtasks", "true")
	}
	if len(query) > 0 {
		endpoint = endpoint + "?" + query.Encode()
	}

	if _, err := c.execRequest(http.MethodDelete, endpoint, nil); err != nil {
		return err
	}
	return nil
}

// WebhookRegistration describes a webhook to register with Jira.
type WebhookRegistration struct {
	Events    []string `json:"events"`
	JQLFilter string   `json:"jqlFilter"`
}

type registerWebhooksRequest struct {
	URL      string                `json:"url"`
	Webhooks []WebhookRegistration `json:"webhooks"`
}

type registerWebhooksResponse struct {
	WebhookRegistrationResult []registeredWebhook `json:"webhookRegistrationResult"`
}

type registeredWebhook struct {
	CreatedWebhookID int      `json:"createdWebhookId,omitempty"`
	Errors           []string `json:"errors,omitempty"`
}

// RegisterWebhooks creates dynamic webhooks against the configured Jira site.
// Returns the IDs of the newly created webhooks.
func (c *Client) RegisterWebhooks(webhookURL string, registrations []WebhookRegistration) ([]int, error) {
	payload := registerWebhooksRequest{URL: webhookURL, Webhooks: registrations}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling webhook payload: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, c.apiURL("/rest/api/3/webhook"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var resp registerWebhooksResponse
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return nil, fmt.Errorf("error parsing webhook response: %v", err)
	}

	ids := make([]int, 0, len(resp.WebhookRegistrationResult))
	for _, r := range resp.WebhookRegistrationResult {
		if len(r.Errors) > 0 {
			return ids, fmt.Errorf("webhook registration error: %v", r.Errors)
		}
		if r.CreatedWebhookID != 0 {
			ids = append(ids, r.CreatedWebhookID)
		}
	}
	return ids, nil
}

type deleteWebhooksRequest struct {
	WebhookIDs []int `json:"webhookIds"`
}

// DeleteWebhooks removes the given webhook IDs.
func (c *Client) DeleteWebhooks(ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	payload := deleteWebhooksRequest{WebhookIDs: ids}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling delete payload: %v", err)
	}

	if _, err := c.execRequest(http.MethodDelete, c.apiURL("/rest/api/3/webhook"), bytes.NewReader(body)); err != nil {
		return err
	}
	return nil
}

// RefreshWebhooks extends the expiry of the given webhook IDs (Jira deletes
// webhooks after 30 days unless refreshed).
func (c *Client) RefreshWebhooks(ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	payload := deleteWebhooksRequest{WebhookIDs: ids}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling refresh payload: %v", err)
	}

	if _, err := c.execRequest(http.MethodPut, c.apiURL("/rest/api/3/webhook/refresh"), bytes.NewReader(body)); err != nil {
		return err
	}
	return nil
}
