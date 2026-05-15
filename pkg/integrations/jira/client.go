package jira

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

// Client speaks to Jira Cloud directly using the user's site URL and Basic
// Auth (email + API token). Endpoints resolve to `{siteUrl}/rest/api/3/...`.
type Client struct {
	SiteURL string
	Email   string
	Token   string
	http    core.HTTPContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	siteURL, err := ctx.GetConfig("siteUrl")
	if err != nil {
		return nil, fmt.Errorf("error reading site URL: %v", err)
	}

	email, err := ctx.GetConfig("email")
	if err != nil {
		return nil, fmt.Errorf("error reading email: %v", err)
	}

	token, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("error reading API token: %v", err)
	}

	if len(siteURL) == 0 {
		return nil, fmt.Errorf("missing Jira site URL")
	}
	if len(email) == 0 {
		return nil, fmt.Errorf("missing Jira email")
	}
	if len(token) == 0 {
		return nil, fmt.Errorf("missing API token")
	}

	return &Client{
		SiteURL: strings.TrimRight(string(siteURL), "/"),
		Email:   string(email),
		Token:   string(token),
		http:    httpCtx,
	}, nil
}

func (c *Client) apiURL(path string) string {
	return c.SiteURL + path
}

func (c *Client) basicAuthHeader() string {
	creds := c.Email + ":" + c.Token
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.basicAuthHeader())

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

type User struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	EmailAddr   string `json:"emailAddress,omitempty"`
}

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

type Project struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

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

type IssueTypeMeta struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Subtask bool   `json:"subtask"`
}

type createMetaIssueTypesResponse struct {
	IssueTypes []IssueTypeMeta `json:"issueTypes"`
}

// GetProjectIssueTypes returns the issue types available for creating issues
// in the given project. Uses the create-metadata endpoint which scopes the
// list to types the user is permitted to create.
func (c *Client) GetProjectIssueTypes(projectKey string) ([]IssueTypeMeta, error) {
	endpoint := c.apiURL("/rest/api/3/issue/createmeta/" + url.PathEscape(projectKey) + "/issuetypes")

	body, err := c.execRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp createMetaIssueTypesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parsing issue types response: %v", err)
	}
	return resp.IssueTypes, nil
}

type Status struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type projectStatusesIssueType struct {
	Statuses []Status `json:"statuses"`
}

// GetProjectStatuses returns the unique set of statuses across all issue
// types in a project. /rest/api/3/project/{key}/statuses returns an entry
// per issue type, each with its own status list — we flatten and dedupe by
// status name.
func (c *Client) GetProjectStatuses(projectKey string) ([]Status, error) {
	endpoint := c.apiURL("/rest/api/3/project/" + url.PathEscape(projectKey) + "/statuses")

	body, err := c.execRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var raw []projectStatusesIssueType
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("error parsing statuses response: %v", err)
	}

	seen := map[string]bool{}
	statuses := []Status{}
	for _, it := range raw {
		for _, s := range it.Statuses {
			if seen[s.Name] {
				continue
			}
			seen[s.Name] = true
			statuses = append(statuses, s)
		}
	}
	return statuses, nil
}

type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   Status `json:"to"`
}

type transitionsResponse struct {
	Transitions []Transition `json:"transitions"`
}

// GetIssueTransitions returns the transitions available from an issue's
// current workflow state.
func (c *Client) GetIssueTransitions(issueKey string) ([]Transition, error) {
	endpoint := c.apiURL("/rest/api/3/issue/" + url.PathEscape(issueKey) + "/transitions")

	body, err := c.execRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp transitionsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parsing transitions response: %v", err)
	}
	return resp.Transitions, nil
}

type doTransitionRequest struct {
	Transition transitionID `json:"transition"`
}

type transitionID struct {
	ID string `json:"id"`
}

// ListAssignableUsers returns the users assignable to issues in a given
// project. /rest/api/3/user/assignable/search is paginated; we cap at 50
// entries, which matches the picker's practical UX.
func (c *Client) ListAssignableUsers(projectKey string) ([]User, error) {
	query := url.Values{}
	query.Set("project", projectKey)
	query.Set("maxResults", "50")
	endpoint := c.apiURL("/rest/api/3/user/assignable/search?" + query.Encode())

	body, err := c.execRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var users []User
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("error parsing assignable users response: %v", err)
	}
	return users, nil
}

type Priority struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListPriorities returns all priorities configured on the Jira site.
// Priorities are instance-level, not project-scoped.
func (c *Client) ListPriorities() ([]Priority, error) {
	body, err := c.execRequest(http.MethodGet, c.apiURL("/rest/api/3/priority"), nil)
	if err != nil {
		return nil, err
	}

	var priorities []Priority
	if err := json.Unmarshal(body, &priorities); err != nil {
		return nil, fmt.Errorf("error parsing priorities response: %v", err)
	}
	return priorities, nil
}

// DoTransition advances an issue along the given workflow transition.
func (c *Client) DoTransition(issueKey, id string) error {
	endpoint := c.apiURL("/rest/api/3/issue/" + url.PathEscape(issueKey) + "/transitions")

	body, err := json.Marshal(doTransitionRequest{Transition: transitionID{ID: id}})
	if err != nil {
		return fmt.Errorf("error marshaling transition request: %v", err)
	}

	if _, err := c.execRequest(http.MethodPost, endpoint, bytes.NewReader(body)); err != nil {
		return err
	}
	return nil
}

type Issue struct {
	ID     string         `json:"id"`
	Key    string         `json:"key"`
	Self   string         `json:"self"`
	Fields map[string]any `json:"fields"`
}

type GetIssueOptions struct {
	Fields string
	Expand string
}

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

type CreateIssueRequest struct {
	Fields CreateIssueFields `json:"fields"`
}

type CreateIssueFields struct {
	Project     ProjectRef `json:"project"`
	IssueType   IssueType  `json:"issuetype"`
	Summary     string     `json:"summary"`
	Description *ADFDoc    `json:"description,omitempty"`
	Assignee    *UserRef   `json:"assignee,omitempty"`
}

type ProjectRef struct {
	Key string `json:"key"`
}

type UserRef struct {
	AccountID string `json:"accountId"`
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

type UpdateIssueRequest struct {
	Fields map[string]any `json:"fields,omitempty"`
}

type UpdateIssueOptions struct {
	NotifyUsers *bool
}

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
