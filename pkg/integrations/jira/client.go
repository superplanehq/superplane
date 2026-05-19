package jira

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
	ID         string `json:"id"`
	Key        string `json:"key"`
	Name       string `json:"name"`
	Style      string `json:"style,omitempty"`
	Simplified bool   `json:"simplified,omitempty"`
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

func (c *Client) GetProject(projectKey string) (*Project, error) {
	endpoint := c.apiURL("/rest/api/3/project/" + url.PathEscape(projectKey))

	body, err := c.execRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var project Project
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("error parsing project response: %v", err)
	}
	return &project, nil
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

type transitionID struct {
	ID string `json:"id"`
}

type DoTransitionOptions struct {
	Comment    string
	Resolution string
}

type doTransitionRequest struct {
	Transition transitionID   `json:"transition"`
	Fields     map[string]any `json:"fields,omitempty"`
	Update     map[string]any `json:"update,omitempty"`
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

type Resolution struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListResolutions returns all resolutions configured on the Jira site.
// Resolutions are instance-level, not project-scoped.
func (c *Client) ListResolutions() ([]Resolution, error) {
	body, err := c.execRequest(http.MethodGet, c.apiURL("/rest/api/3/resolution"), nil)
	if err != nil {
		return nil, err
	}

	var resolutions []Resolution
	if err := json.Unmarshal(body, &resolutions); err != nil {
		return nil, fmt.Errorf("error parsing resolutions response: %v", err)
	}
	return resolutions, nil
}

// DoTransition advances an issue along the given workflow transition.
func (c *Client) DoTransition(issueKey, id string) error {
	return c.DoTransitionWithOptions(issueKey, id, DoTransitionOptions{})
}

// DoTransitionWithOptions advances an issue and optionally applies transition-scoped fields.
func (c *Client) DoTransitionWithOptions(issueKey, id string, opts DoTransitionOptions) error {
	endpoint := c.apiURL("/rest/api/3/issue/" + url.PathEscape(issueKey) + "/transitions")

	req := doTransitionRequest{Transition: transitionID{ID: id}}
	if resolution := strings.TrimSpace(opts.Resolution); resolution != "" {
		req.Fields = map[string]any{
			"resolution": map[string]any{"name": resolution},
		}
	}
	if comment := strings.TrimSpace(opts.Comment); comment != "" {
		req.Update = map[string]any{
			"comment": []map[string]any{
				{
					"add": map[string]any{
						"body": WrapInADF(comment),
					},
				},
			},
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshaling transition request: %v", err)
	}

	if _, err := c.execRequest(http.MethodPost, endpoint, bytes.NewReader(body)); err != nil {
		return err
	}
	return nil
}

type FlexibleString string

func (s *FlexibleString) UnmarshalJSON(b []byte) error {
	raw := strings.TrimSpace(string(b))
	if raw == "" || raw == "null" {
		*s = ""
		return nil
	}

	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		*s = FlexibleString(str)
		return nil
	}

	*s = FlexibleString(raw)
	return nil
}

func (s FlexibleString) String() string {
	return string(s)
}

type WorkflowScope struct {
	Type    string                   `json:"type"`
	Project *WorkflowScopeProjectRef `json:"project,omitempty"`
}

type WorkflowScopeProjectRef struct {
	ID string `json:"id"`
}

type WorkflowLayout struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type WorkflowStatusUpdate struct {
	Description     string `json:"description"`
	Name            string `json:"name"`
	StatusCategory  string `json:"statusCategory"`
	StatusReference string `json:"statusReference"`
}

type WorkflowCreateStatus struct {
	Layout          WorkflowLayout `json:"layout"`
	Properties      map[string]any `json:"properties"`
	StatusReference string         `json:"statusReference"`
}

type WorkflowTransitionLink struct {
	FromPort            int    `json:"fromPort"`
	FromStatusReference string `json:"fromStatusReference"`
	ToPort              int    `json:"toPort"`
}

type WorkflowCreateTransition struct {
	Actions           []any                    `json:"actions"`
	Description       string                   `json:"description"`
	ID                string                   `json:"id"`
	Links             []WorkflowTransitionLink `json:"links"`
	Name              string                   `json:"name"`
	Properties        map[string]any           `json:"properties"`
	ToStatusReference string                   `json:"toStatusReference"`
	Triggers          []any                    `json:"triggers"`
	Type              string                   `json:"type"`
	Validators        []any                    `json:"validators"`
}

type WorkflowCreate struct {
	Description      string                     `json:"description"`
	Name             string                     `json:"name"`
	StartPointLayout WorkflowLayout             `json:"startPointLayout"`
	Statuses         []WorkflowCreateStatus     `json:"statuses"`
	Transitions      []WorkflowCreateTransition `json:"transitions"`
}

type CreateWorkflowRequest struct {
	Scope     WorkflowScope          `json:"scope"`
	Statuses  []WorkflowStatusUpdate `json:"statuses"`
	Workflows []WorkflowCreate       `json:"workflows"`
}

type WorkflowVersion struct {
	ID            string `json:"id"`
	VersionNumber int    `json:"versionNumber"`
}

type CreatedWorkflow struct {
	Description string          `json:"description,omitempty"`
	ID          string          `json:"id"`
	IsEditable  bool            `json:"isEditable,omitempty"`
	Name        string          `json:"name"`
	Scope       WorkflowScope   `json:"scope"`
	Version     WorkflowVersion `json:"version"`
}

type CreateWorkflowResponse struct {
	Statuses  []WorkflowStatusUpdate `json:"statuses"`
	Workflows []CreatedWorkflow      `json:"workflows"`
}

func (c *Client) CreateWorkflow(req *CreateWorkflowRequest) (*CreateWorkflowResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling workflow create request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, c.apiURL("/rest/api/3/workflows/create"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response CreateWorkflowResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("error parsing workflow create response: %v", err)
	}
	return &response, nil
}

type WorkflowScheme struct {
	ID              FlexibleString `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description,omitempty"`
	DefaultWorkflow string         `json:"defaultWorkflow,omitempty"`
	Draft           bool           `json:"draft,omitempty"`
	Self            string         `json:"self,omitempty"`
}

type workflowSchemesPage struct {
	StartAt    int              `json:"startAt"`
	MaxResults int              `json:"maxResults"`
	Total      int              `json:"total"`
	IsLast     bool             `json:"isLast"`
	Values     []WorkflowScheme `json:"values"`
}

func (c *Client) ListWorkflowSchemes() ([]WorkflowScheme, error) {
	var out []WorkflowScheme
	startAt := 0
	const pageSize = 50

	for range 20 {
		query := url.Values{}
		query.Set("startAt", strconv.Itoa(startAt))
		query.Set("maxResults", strconv.Itoa(pageSize))
		endpoint := c.apiURL("/rest/api/3/workflowscheme?" + query.Encode())

		body, err := c.execRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}

		var page workflowSchemesPage
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("error parsing workflow schemes response: %v", err)
		}

		out = append(out, page.Values...)
		if page.IsLast || len(page.Values) == 0 {
			break
		}
		startAt += len(page.Values)
		if page.Total > 0 && startAt >= page.Total {
			break
		}
	}

	return out, nil
}

type WorkflowSchemeAssignmentResponse struct {
	ProjectID        string
	WorkflowSchemeID string
	Task             *TaskProgress
}

type TaskProgress struct {
	ID       FlexibleString `json:"id"`
	Self     string         `json:"self,omitempty"`
	Status   string         `json:"status,omitempty"`
	Message  string         `json:"message,omitempty"`
	Progress int            `json:"progress,omitempty"`
}

type assignWorkflowSchemeRequest struct {
	ProjectID                    string `json:"projectId"`
	TargetSchemeID               string `json:"targetSchemeId"`
	MappingsByIssueTypeOverrides []any  `json:"mappingsByIssueTypeOverride,omitempty"`
}

// AssignWorkflowSchemeToProject switches the workflow scheme for a classic Jira project.
func (c *Client) AssignWorkflowSchemeToProject(projectID, schemeID string) (*WorkflowSchemeAssignmentResponse, error) {
	req := assignWorkflowSchemeRequest{
		ProjectID:                    projectID,
		TargetSchemeID:               schemeID,
		MappingsByIssueTypeOverrides: []any{},
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling workflow scheme assignment request: %v", err)
	}

	endpoint := c.apiURL("/rest/api/3/workflowscheme/project/switch")
	responseBody, status, err := c.execRequestWithStatus(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if status < 200 || (status >= 300 && status != http.StatusSeeOther) {
		return nil, fmt.Errorf("request got %d code: %s", status, string(responseBody))
	}

	out := &WorkflowSchemeAssignmentResponse{
		ProjectID:        projectID,
		WorkflowSchemeID: schemeID,
	}
	if len(strings.TrimSpace(string(responseBody))) == 0 {
		return out, nil
	}

	var task TaskProgress
	if err := json.Unmarshal(responseBody, &task); err != nil {
		return nil, fmt.Errorf("error parsing workflow scheme assignment response: %v", err)
	}
	out.Task = &task
	return out, nil
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

// IssueSearchHit is one element from GET /rest/api/3/search.
type IssueSearchHit struct {
	ID     string         `json:"id"`
	Key    string         `json:"key"`
	Fields map[string]any `json:"fields"`
}

type issueSearchAPIResponse struct {
	StartAt    int              `json:"startAt"`
	MaxResults int              `json:"maxResults"`
	Total      int              `json:"total"`
	Issues     []IssueSearchHit `json:"issues"`
}

type jiraSearchPOSTBody struct {
	JQL        string   `json:"jql"`
	StartAt    int      `json:"startAt"`
	MaxResults int      `json:"maxResults"`
	Fields     []string `json:"fields"`
}

func (c *Client) searchIssuesPage(jql string, startAt, maxResults int) (issueSearchAPIResponse, error) {
	var empty issueSearchAPIResponse
	if maxResults <= 0 {
		maxResults = 50
	}
	if maxResults > 100 {
		maxResults = 100
	}

	body := jiraSearchPOSTBody{
		JQL:        jql,
		StartAt:    startAt,
		MaxResults: maxResults,
		Fields:     []string{"summary"},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return empty, fmt.Errorf("marshal search body: %w", err)
	}

	base := strings.TrimSuffix(c.SiteURL, "/")
	u := fmt.Sprintf("%s/rest/api/3/search", base)
	responseBody, err := c.execRequest(http.MethodPost, u, bytes.NewReader(bodyBytes))
	if err != nil {
		return empty, err
	}

	var resp issueSearchAPIResponse
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return empty, fmt.Errorf("parse search response: %w", err)
	}
	if resp.Issues == nil {
		resp.Issues = []IssueSearchHit{}
	}

	return resp, nil
}

// SearchIssues runs a JQL search and returns the first page of issues (maxResults is capped at 100).
func (c *Client) SearchIssues(jql string, maxResults int) ([]IssueSearchHit, error) {
	resp, err := c.searchIssuesPage(jql, 0, maxResults)
	if err != nil {
		return nil, err
	}
	return resp.Issues, nil
}

// SearchIssuesUpTo pages through POST /rest/api/3/search until maxIssues are collected, a page is
// short, or Jira reports no further results. Jira caps each request at 100 issues; busy service
// projects often need more than one page so incident pickers are not dominated by recently
// updated non-incident work.
func (c *Client) SearchIssuesUpTo(jql string, maxIssues int) ([]IssueSearchHit, error) {
	if maxIssues <= 0 {
		maxIssues = 500
	}
	const pageCap = 100

	var out []IssueSearchHit
	startAt := 0
	for len(out) < maxIssues {
		pageMax := pageCap
		if remain := maxIssues - len(out); remain < pageMax {
			pageMax = remain
		}
		if pageMax <= 0 {
			break
		}

		resp, err := c.searchIssuesPage(jql, startAt, pageMax)
		if err != nil {
			return nil, err
		}

		out = append(out, resp.Issues...)
		if len(resp.Issues) == 0 {
			break
		}
		startAt += len(resp.Issues)
		if len(resp.Issues) < pageMax {
			break
		}
		if resp.Total > 0 && startAt >= resp.Total {
			break
		}
	}

	return out, nil
}

// CustomerRequestListed is one row from GET /rest/servicedeskapi/request (Jira Service Management).
type CustomerRequestListed struct {
	IssueKey string `json:"issueKey"`
	Summary  string `json:"summary"`
}

type pagedCustomerRequests struct {
	Values     []CustomerRequestListed `json:"values"`
	IsLastPage bool                    `json:"isLastPage"`
}

// ListCustomerRequestsByServiceDesk pages through customer requests for a service desk.
// Agents often see JSM work here even when Jira platform issue search returns no rows for the same project.
func (c *Client) ListCustomerRequestsByServiceDesk(serviceDeskID string, maxTotal int) ([]CustomerRequestListed, error) {
	serviceDeskID = strings.TrimSpace(serviceDeskID)
	if serviceDeskID == "" {
		return nil, fmt.Errorf("service desk id is required")
	}
	if maxTotal <= 0 {
		maxTotal = 500
	}

	base := strings.TrimSuffix(c.SiteURL, "/")
	const limit = 100
	var out []CustomerRequestListed
	start := 0
	for len(out) < maxTotal {
		q := url.Values{}
		q.Set("serviceDeskId", serviceDeskID)
		q.Set("start", strconv.Itoa(start))
		q.Set("limit", strconv.Itoa(limit))

		u := fmt.Sprintf("%s/rest/servicedeskapi/request?%s", base, q.Encode())
		responseBody, err := c.execRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}

		var page pagedCustomerRequests
		if err := json.Unmarshal(responseBody, &page); err != nil {
			return nil, fmt.Errorf("parse customer requests: %w", err)
		}
		if page.Values == nil {
			page.Values = []CustomerRequestListed{}
		}

		for _, row := range page.Values {
			out = append(out, row)
			if len(out) >= maxTotal {
				break
			}
		}
		if len(out) >= maxTotal {
			break
		}
		if page.IsLastPage || len(page.Values) == 0 {
			break
		}
		start += len(page.Values)
	}

	return out, nil
}

// CustomerRequest is returned by GET /rest/servicedeskapi/request/{issueIdOrKey}.
type CustomerRequest struct {
	IssueID       string `json:"issueId,omitempty"`
	IssueKey      string `json:"issueKey,omitempty"`
	ServiceDeskID string `json:"serviceDeskId,omitempty"`
	RequestTypeID string `json:"requestTypeId,omitempty"`
}

func (c *Client) GetCustomerRequest(issueKey string) (*CustomerRequest, error) {
	base := strings.TrimSuffix(c.SiteURL, "/")
	u := fmt.Sprintf("%s/rest/servicedeskapi/request/%s", base, url.PathEscape(issueKey))
	responseBody, err := c.execRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	var out CustomerRequest
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, fmt.Errorf("parse customer request: %w", err)
	}
	return &out, nil
}

type Approval struct {
	ID            FlexibleString `json:"id"`
	Name          string         `json:"name,omitempty"`
	FinalDecision string         `json:"finalDecision,omitempty"`
	Approvers     []Approver     `json:"approvers,omitempty"`
	CreatedDate   map[string]any `json:"createdDate,omitempty"`
	CompletedDate map[string]any `json:"completedDate,omitempty"`
	Links         map[string]any `json:"_links,omitempty"`
}

type Approver struct {
	Approver         User   `json:"approver,omitempty"`
	ApproverDecision string `json:"approverDecision,omitempty"`
}

type approvalsPage struct {
	Values     []Approval `json:"values"`
	IsLastPage bool       `json:"isLastPage"`
}

func (c *Client) ListApprovals(issueKey string) ([]Approval, error) {
	base := strings.TrimSuffix(c.SiteURL, "/")
	var out []Approval
	start := 0
	const pageSize = 50

	for range 20 {
		query := url.Values{}
		query.Set("start", strconv.Itoa(start))
		query.Set("limit", strconv.Itoa(pageSize))
		u := fmt.Sprintf("%s/rest/servicedeskapi/request/%s/approval?%s", base, url.PathEscape(issueKey), query.Encode())

		responseBody, err := c.execRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}

		var page approvalsPage
		if err := json.Unmarshal(responseBody, &page); err != nil {
			return nil, fmt.Errorf("parse approvals: %w", err)
		}

		out = append(out, page.Values...)
		if page.IsLastPage || len(page.Values) == 0 {
			break
		}
		start += len(page.Values)
	}

	return out, nil
}

func (c *Client) SubmitApprovalDecision(issueKey, approvalID, decision string) (*Approval, error) {
	base := strings.TrimSuffix(c.SiteURL, "/")
	u := fmt.Sprintf(
		"%s/rest/servicedeskapi/request/%s/approval/%s",
		base,
		url.PathEscape(issueKey),
		url.PathEscape(approvalID),
	)

	body, err := json.Marshal(map[string]string{"decision": strings.ToLower(strings.TrimSpace(decision))})
	if err != nil {
		return nil, fmt.Errorf("marshal approval decision: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var out Approval
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, fmt.Errorf("parse approval decision response: %w", err)
	}
	return &out, nil
}

func (c *Client) AddCustomerRequestComment(issueKey, body string, public bool) error {
	base := strings.TrimSuffix(c.SiteURL, "/")
	u := fmt.Sprintf("%s/rest/servicedeskapi/request/%s/comment", base, url.PathEscape(issueKey))

	requestBody, err := json.Marshal(map[string]any{
		"body":   body,
		"public": public,
	})
	if err != nil {
		return fmt.Errorf("marshal customer request comment: %w", err)
	}

	_, err = c.execRequest(http.MethodPost, u, bytes.NewReader(requestBody))
	return err
}

func jqlQuotedProjectKey(projectKey string) string {
	escaped := strings.ReplaceAll(projectKey, `\`, `\\`)
	return strings.ReplaceAll(escaped, `"`, `\"`)
}

const atlassianIncidentAPIHost = "https://api.atlassian.com"

type tenantInfoResponse struct {
	CloudID string `json:"cloudId"`
}

// FetchCloudID returns the Atlassian cloud id for the configured Jira site (required for the JSM Incidents API).
func (c *Client) FetchCloudID() (string, error) {
	tenantURL := strings.TrimSuffix(c.SiteURL, "/") + "/_edge/tenant_info"
	responseBody, err := c.execRequest(http.MethodGet, tenantURL, nil)
	if err != nil {
		return "", fmt.Errorf("fetch tenant_info: %w", err)
	}

	var info tenantInfoResponse
	if err := json.Unmarshal(responseBody, &info); err != nil {
		return "", fmt.Errorf("parse tenant_info: %w", err)
	}
	if info.CloudID == "" {
		return "", fmt.Errorf("tenant_info response missing cloudId")
	}
	return info.CloudID, nil
}

// ServiceDesk is returned by GET /rest/servicedeskapi/servicedesk (Jira Service Management).
type ServiceDesk struct {
	ID          string `json:"id"`
	ProjectName string `json:"projectName"`
	ProjectKey  string `json:"projectKey"`
}

// RequestType is returned by GET /rest/servicedeskapi/servicedesk/{id}/requesttype (use expand=practice for the practice field).
type RequestType struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Practice string `json:"practice,omitempty"`
}

type pagedServiceDesks struct {
	Values     []ServiceDesk `json:"values"`
	IsLastPage bool          `json:"isLastPage"`
}

// ListServiceDesks returns service desks the authenticated user can access.
func (c *Client) ListServiceDesks() ([]ServiceDesk, error) {
	base := strings.TrimSuffix(c.SiteURL, "/")
	var out []ServiceDesk
	start := 0
	const pageSize = 50
	for range 20 {
		u := fmt.Sprintf("%s/rest/servicedeskapi/servicedesk?start=%d&limit=%d", base, start, pageSize)
		responseBody, err := c.execRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}

		var page pagedServiceDesks
		if err := json.Unmarshal(responseBody, &page); err != nil {
			return nil, fmt.Errorf("parse service desks: %w", err)
		}

		out = append(out, page.Values...)
		if page.IsLastPage || len(page.Values) == 0 {
			break
		}
		start += len(page.Values)
	}

	return out, nil
}

type pagedRequestTypes struct {
	Values     []RequestType `json:"values"`
	IsLastPage bool          `json:"isLastPage"`
}

// ListRequestTypes returns customer request types for a service desk.
func (c *Client) ListRequestTypes(serviceDeskID string) ([]RequestType, error) {
	base := strings.TrimSuffix(c.SiteURL, "/")
	var out []RequestType
	start := 0
	const pageSize = 50
	for range 20 {
		q := url.Values{}
		q.Set("start", strconv.Itoa(start))
		q.Set("limit", strconv.Itoa(pageSize))
		q.Add("expand", "practice")
		u := fmt.Sprintf(
			"%s/rest/servicedeskapi/servicedesk/%s/requesttype?%s",
			base,
			url.PathEscape(serviceDeskID),
			q.Encode(),
		)
		responseBody, err := c.execRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}

		var page pagedRequestTypes
		if err := json.Unmarshal(responseBody, &page); err != nil {
			return nil, fmt.Errorf("parse request types: %w", err)
		}

		out = append(out, page.Values...)
		if page.IsLastPage || len(page.Values) == 0 {
			break
		}
		start += len(page.Values)
	}

	return filterRequestTypesForIncidentsAPI(out), nil
}

// IsIncidentManagementRequestPractice reports whether a JSM request type's `practice` field
// corresponds to the Incident management work category. The JSM Incidents REST API rejects
// create calls when the request type is not in that category.
func IsIncidentManagementRequestPractice(practice string) bool {
	p := strings.TrimSpace(practice)
	if p == "" {
		return false
	}

	u := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(p, "-", "_"), " ", "_"), ".", "_"))
	if strings.Contains(u, "POST_INCIDENT") {
		return false
	}

	low := strings.ToLower(p)
	if strings.Contains(low, "incident management") {
		return true
	}

	switch u {
	case "ITSM_INCIDENT", "INCIDENT_MANAGEMENT", "INCIDENT", "MANAGE_INCIDENTS", "IM":
		return true
	}
	if strings.Contains(u, "INCIDENT_MANAGEMENT") {
		return true
	}
	if u == "INCIDENT" || strings.HasSuffix(u, "_INCIDENT") {
		return true
	}
	if strings.Contains(u, "INCIDENT") && strings.Contains(u, "MANAGEMENT") {
		return true
	}
	return false
}

func filterRequestTypesForIncidentsAPI(all []RequestType) []RequestType {
	hasPractice := false
	for _, rt := range all {
		if strings.TrimSpace(rt.Practice) != "" {
			hasPractice = true
			break
		}
	}
	if !hasPractice {
		return all
	}

	out := make([]RequestType, 0, len(all))
	for _, rt := range all {
		p := strings.TrimSpace(rt.Practice)
		if p == "" {
			continue
		}
		if IsIncidentManagementRequestPractice(p) {
			out = append(out, rt)
		}
	}
	if len(out) == 0 {
		return all
	}
	return out
}

// RequestTypeField is returned by GET .../requesttype/{id}/field.
type RequestTypeField struct {
	FieldID     string                  `json:"fieldId"`
	Name        string                  `json:"name"`
	Required    bool                    `json:"required"`
	ValidValues []RequestTypeFieldValue `json:"validValues"`
}

// RequestTypeFieldValue is an allowed option on a request type field.
type RequestTypeFieldValue struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type requestTypeFieldsResponse struct {
	RequestTypeFields []RequestTypeField `json:"requestTypeFields"`
}

// ListRequestTypeFields returns fields for a service desk request type (including hidden fields).
func (c *Client) ListRequestTypeFields(serviceDeskID, requestTypeID string) ([]RequestTypeField, error) {
	base := strings.TrimSuffix(c.SiteURL, "/")
	u := fmt.Sprintf(
		"%s/rest/servicedeskapi/servicedesk/%s/requesttype/%s/field",
		base,
		url.PathEscape(serviceDeskID),
		url.PathEscape(requestTypeID),
	)
	responseBody, err := c.execRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var out requestTypeFieldsResponse
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, fmt.Errorf("parse request type fields: %w", err)
	}
	return out.RequestTypeFields, nil
}

type customFieldOptionsPage struct {
	Values     []customFieldOption `json:"values"`
	IsLast     bool                `json:"isLast"`
	StartAt    int                 `json:"startAt"`
	MaxResults int                 `json:"maxResults"`
}

type customFieldOption struct {
	ID       string `json:"id"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled,omitempty"`
}

type JiraFieldInfo struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Custom bool           `json:"custom"`
	Schema map[string]any `json:"schema,omitempty"`
}

type customFieldContext struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	IsGlobalContext bool     `json:"isGlobalContext"`
	ProjectIDs      []string `json:"projectIds,omitempty"`
}

type customFieldContextsPage struct {
	Values     []customFieldContext `json:"values"`
	IsLast     bool                 `json:"isLast"`
	StartAt    int                  `json:"startAt"`
	MaxResults int                  `json:"maxResults"`
}

type createMetaResponse struct {
	Projects []createMetaProject `json:"projects"`
}

type createMetaProject struct {
	IssueTypes []createMetaIssueType `json:"issuetypes"`
}

type createMetaIssueType struct {
	ID     string                     `json:"id"`
	Name   string                     `json:"name"`
	Fields map[string]createMetaField `json:"fields"`
}

type createMetaField struct {
	Name          string                   `json:"name"`
	AllowedValues []createMetaAllowedValue `json:"allowedValues"`
}

type createMetaAllowedValue struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type createMetaIssueTypesOnlyResponse struct {
	IssueTypes []createMetaIssueTypeRef `json:"issueTypes"`
}

type createMetaIssueTypeRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type createMetaIssueTypeFieldsResponse struct {
	Fields map[string]createMetaField `json:"fields"`
}

// ListCustomFieldOptions returns select options for a Jira field on Cloud using best-effort fallbacks.
func (c *Client) ListCustomFieldOptions(fieldID, projectKey, fieldLabel string) []RequestTypeFieldValue {
	fieldID = strings.TrimSpace(fieldID)
	fieldLabel = strings.TrimSpace(fieldLabel)
	projectKey = strings.TrimSpace(projectKey)

	if projectKey != "" && fieldLabel != "" {
		if opts, _ := c.listFieldAllowedValuesFromCreateMeta(projectKey, fieldLabel); len(opts) > 0 {
			return opts
		}
	}

	if fieldID != "" {
		if opts := c.listCustomFieldOptionsDirect(fieldID); len(opts) > 0 {
			return opts
		}
		if strings.HasPrefix(fieldID, "customfield_") {
			if opts := c.listCustomFieldOptionsFromContexts(fieldID); len(opts) > 0 {
				return opts
			}
		}
	}

	return nil
}

func (c *Client) listCustomFieldOptionsDirect(fieldID string) []RequestTypeFieldValue {
	if !strings.HasPrefix(fieldID, "customfield_") {
		return nil
	}

	base := strings.TrimSuffix(c.SiteURL, "/")
	var out []RequestTypeFieldValue
	startAt := 0
	const pageSize = 100

	for range 20 {
		u := fmt.Sprintf(
			"%s/rest/api/3/field/%s/option?startAt=%d&maxResults=%d",
			base,
			url.PathEscape(fieldID),
			startAt,
			pageSize,
		)
		body, status, err := c.execRequestWithStatus(http.MethodGet, u, nil)
		if err != nil || status == http.StatusNotFound {
			return out
		}
		if status < 200 || status >= 300 {
			return out
		}

		var page customFieldOptionsPage
		if err := json.Unmarshal(body, &page); err != nil {
			return out
		}

		out = append(out, pageValuesToFieldOptions(page.Values)...)

		if page.IsLast || len(page.Values) == 0 {
			break
		}
		startAt += len(page.Values)
	}

	return out
}

func (c *Client) listCustomFieldOptionsFromContexts(fieldID string) []RequestTypeFieldValue {
	base := strings.TrimSuffix(c.SiteURL, "/")
	var contexts []customFieldContext
	startAt := 0
	const pageSize = 50

	for range 20 {
		u := fmt.Sprintf(
			"%s/rest/api/3/field/%s/context?startAt=%d&maxResults=%d",
			base,
			url.PathEscape(fieldID),
			startAt,
			pageSize,
		)
		body, status, err := c.execRequestWithStatus(http.MethodGet, u, nil)
		if err != nil || status == http.StatusNotFound {
			return nil
		}
		if status < 200 || status >= 300 {
			return nil
		}

		var page customFieldContextsPage
		if err := json.Unmarshal(body, &page); err != nil {
			return nil
		}
		contexts = append(contexts, page.Values...)
		if page.IsLast || len(page.Values) == 0 {
			break
		}
		startAt += len(page.Values)
	}

	ordered := orderFieldContexts(contexts, "")
	for _, ctx := range ordered {
		if opts := c.listCustomFieldOptionsForContext(fieldID, ctx.ID); len(opts) > 0 {
			return opts
		}
	}
	return nil
}

func orderFieldContexts(contexts []customFieldContext, projectID string) []customFieldContext {
	if projectID == "" {
		return contexts
	}
	var matched, global, rest []customFieldContext
	for _, ctx := range contexts {
		switch {
		case contextIncludesProject(ctx, projectID):
			matched = append(matched, ctx)
		case ctx.IsGlobalContext:
			global = append(global, ctx)
		default:
			rest = append(rest, ctx)
		}
	}
	out := make([]customFieldContext, 0, len(contexts))
	out = append(out, matched...)
	out = append(out, global...)
	out = append(out, rest...)
	return out
}

func contextIncludesProject(ctx customFieldContext, projectID string) bool {
	for _, id := range ctx.ProjectIDs {
		if id == projectID {
			return true
		}
	}
	return false
}

func (c *Client) listCustomFieldOptionsForContext(fieldID, contextID string) []RequestTypeFieldValue {
	base := strings.TrimSuffix(c.SiteURL, "/")
	var out []RequestTypeFieldValue
	startAt := 0
	const pageSize = 100

	for range 20 {
		u := fmt.Sprintf(
			"%s/rest/api/3/field/%s/context/%s/option?startAt=%d&maxResults=%d",
			base,
			url.PathEscape(fieldID),
			url.PathEscape(contextID),
			startAt,
			pageSize,
		)
		body, status, err := c.execRequestWithStatus(http.MethodGet, u, nil)
		if err != nil || status == http.StatusNotFound {
			return out
		}
		if status < 200 || status >= 300 {
			return out
		}

		var page customFieldOptionsPage
		if err := json.Unmarshal(body, &page); err != nil {
			return out
		}

		out = append(out, pageValuesToFieldOptions(page.Values)...)

		if page.IsLast || len(page.Values) == 0 {
			break
		}
		startAt += len(page.Values)
	}

	return out
}

func pageValuesToFieldOptions(values []customFieldOption) []RequestTypeFieldValue {
	out := make([]RequestTypeFieldValue, 0, len(values))
	for _, opt := range values {
		if opt.Disabled {
			continue
		}
		id := strings.TrimSpace(opt.ID)
		label := strings.TrimSpace(opt.Value)
		if id == "" && label == "" {
			continue
		}
		if id == "" {
			id = label
		}
		if label == "" {
			label = id
		}
		out = append(out, RequestTypeFieldValue{Label: label, Value: id})
	}
	return out
}

// ListFields returns all fields visible to the integration (Jira REST API v3).
func (c *Client) ListFields() ([]JiraFieldInfo, error) {
	base := strings.TrimSuffix(c.SiteURL, "/")
	body, err := c.execRequest(http.MethodGet, base+"/rest/api/3/field", nil)
	if err != nil {
		return nil, err
	}
	var fields []JiraFieldInfo
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, fmt.Errorf("parse fields: %w", err)
	}
	return fields, nil
}

// FindGlobalFieldByLabel finds a field id by display name (e.g. "Urgency").
func FindGlobalFieldByLabel(fields []JiraFieldInfo, fieldLabel string) *JiraFieldInfo {
	want := strings.ToLower(strings.TrimSpace(fieldLabel))
	var contains []JiraFieldInfo

	for i := range fields {
		f := &fields[i]
		nameLower := strings.ToLower(strings.TrimSpace(f.Name))
		if nameLower == want {
			return f
		}
		if strings.Contains(nameLower, want) {
			contains = append(contains, *f)
		}
	}

	if len(contains) == 0 {
		return nil
	}

	best := &contains[0]
	bestScore := globalFieldMatchScore(best.Name, want)
	for i := 1; i < len(contains); i++ {
		score := globalFieldMatchScore(contains[i].Name, want)
		if score > bestScore {
			bestScore = score
			best = &contains[i]
		}
	}
	return best
}

func globalFieldMatchScore(name, want string) int {
	nameLower := strings.ToLower(strings.TrimSpace(name))
	switch {
	case nameLower == want:
		return 50
	case strings.HasPrefix(nameLower, want+" "), strings.HasPrefix(nameLower, want+"-"):
		return 40
	case strings.HasPrefix(nameLower, want):
		return 30
	default:
		return 10
	}
}

func (c *Client) listFieldAllowedValuesFromCreateMeta(projectKey, fieldLabel string) ([]RequestTypeFieldValue, string) {
	if opts, id := c.listFieldAllowedValuesFromCreateMetaModern(projectKey, fieldLabel); len(opts) > 0 {
		return opts, id
	}
	return c.listFieldAllowedValuesFromCreateMetaLegacy(projectKey, fieldLabel)
}

func (c *Client) listFieldAllowedValuesFromCreateMetaModern(projectKey, fieldLabel string) ([]RequestTypeFieldValue, string) {
	projectKey = strings.TrimSpace(projectKey)
	if projectKey == "" {
		return nil, ""
	}

	base := strings.TrimSuffix(c.SiteURL, "/")
	issueTypesURL := fmt.Sprintf("%s/rest/api/3/issue/createmeta/%s/issuetypes", base, url.PathEscape(projectKey))
	body, status, err := c.execRequestWithStatus(http.MethodGet, issueTypesURL, nil)
	if err != nil || status < 200 || status >= 300 {
		return nil, ""
	}

	var issueTypesResp createMetaIssueTypesOnlyResponse
	if err := json.Unmarshal(body, &issueTypesResp); err != nil {
		return nil, ""
	}

	for _, issueType := range issueTypesResp.IssueTypes {
		issueTypeID := strings.TrimSpace(issueType.ID)
		if issueTypeID == "" {
			continue
		}
		fieldsURL := fmt.Sprintf(
			"%s/rest/api/3/issue/createmeta/%s/issuetypes/%s",
			base,
			url.PathEscape(projectKey),
			url.PathEscape(issueTypeID),
		)
		fieldsBody, fieldsStatus, fieldsErr := c.execRequestWithStatus(http.MethodGet, fieldsURL, nil)
		if fieldsErr != nil || fieldsStatus < 200 || fieldsStatus >= 300 {
			continue
		}
		var fieldsResp createMetaIssueTypeFieldsResponse
		if err := json.Unmarshal(fieldsBody, &fieldsResp); err != nil {
			continue
		}
		if opts, fieldID := allowedValuesFromCreateMetaFields(fieldsResp.Fields, fieldLabel); len(opts) > 0 {
			return opts, fieldID
		}
	}
	return nil, ""
}

func (c *Client) listFieldAllowedValuesFromCreateMetaLegacy(projectKey, fieldLabel string) ([]RequestTypeFieldValue, string) {
	projectKey = strings.TrimSpace(projectKey)
	if projectKey == "" {
		return nil, ""
	}

	base := strings.TrimSuffix(c.SiteURL, "/")
	q := url.Values{}
	q.Set("projectKeys", projectKey)
	q.Set("expand", "projects.issuetypes.fields")
	u := fmt.Sprintf("%s/rest/api/3/issue/createmeta?%s", base, q.Encode())

	body, status, err := c.execRequestWithStatus(http.MethodGet, u, nil)
	if err != nil || status < 200 || status >= 300 {
		return nil, ""
	}

	var meta createMetaResponse
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, ""
	}

	for _, project := range meta.Projects {
		for _, issueType := range project.IssueTypes {
			if opts, fieldID := allowedValuesFromCreateMetaFields(issueType.Fields, fieldLabel); len(opts) > 0 {
				return opts, fieldID
			}
		}
	}
	return nil, ""
}

func allowedValuesFromCreateMetaFields(fields map[string]createMetaField, fieldLabel string) ([]RequestTypeFieldValue, string) {
	want := strings.ToLower(strings.TrimSpace(fieldLabel))
	for fieldID, field := range fields {
		nameLower := strings.ToLower(strings.TrimSpace(field.Name))
		if fieldLabel != "" && nameLower != want && !strings.Contains(nameLower, want) {
			continue
		}
		if len(field.AllowedValues) == 0 {
			continue
		}
		out := make([]RequestTypeFieldValue, 0, len(field.AllowedValues))
		for _, av := range field.AllowedValues {
			id := strings.TrimSpace(av.ID)
			label := strings.TrimSpace(av.Name)
			if label == "" {
				label = strings.TrimSpace(av.Value)
			}
			if id == "" && label == "" {
				continue
			}
			if id == "" {
				id = label
			}
			if label == "" {
				label = id
			}
			out = append(out, RequestTypeFieldValue{Label: label, Value: id})
		}
		if len(out) > 0 {
			return out, fieldID
		}
	}
	return nil, ""
}

func (c *Client) execRequestWithStatus(method, requestURL string, body io.Reader) ([]byte, int, error) {
	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return nil, 0, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.basicAuthHeader())

	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, res.StatusCode, fmt.Errorf("error reading body: %v", err)
	}

	return responseBody, res.StatusCode, nil
}

// GetRequestType returns one request type, optionally expanded (always requests expand=practice).
func (c *Client) GetRequestType(serviceDeskID, requestTypeID string) (*RequestType, error) {
	base := strings.TrimSuffix(c.SiteURL, "/")
	q := url.Values{}
	q.Add("expand", "practice")
	u := fmt.Sprintf(
		"%s/rest/servicedeskapi/servicedesk/%s/requesttype/%s?%s",
		base,
		url.PathEscape(serviceDeskID),
		url.PathEscape(requestTypeID),
		q.Encode(),
	)
	responseBody, err := c.execRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var rt RequestType
	if err := json.Unmarshal(responseBody, &rt); err != nil {
		return nil, fmt.Errorf("parse request type: %w", err)
	}
	return &rt, nil
}

// CreateIncidentAPIRequest is the JSON body for JSM Incidents POST /v1/incident.
type CreateIncidentAPIRequest struct {
	ServiceDeskID string         `json:"serviceDeskId"`
	RequestTypeID string         `json:"requestTypeId"`
	Fields        map[string]any `json:"fields"`
	Update        map[string]any `json:"update,omitempty"`
	AlertIDs      []string       `json:"alertIds,omitempty"`
}

// CreateIncidentAPIResponse is returned when an incident is created successfully.
type CreateIncidentAPIResponse struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self"`
}

func (c *Client) incidentAPIURL(cloudID, pathSuffix string) string {
	return fmt.Sprintf("%s/jsm/incidents/cloudId/%s/v1%s", atlassianIncidentAPIHost, cloudID, pathSuffix)
}

// CreateIncident creates an incident via the JSM Incidents API.
func (c *Client) CreateIncident(cloudID string, req *CreateIncidentAPIRequest) (*CreateIncidentAPIResponse, error) {
	u := c.incidentAPIURL(cloudID, "/incident")
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal create incident body: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var out CreateIncidentAPIResponse
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, fmt.Errorf("parse create incident response: %w", err)
	}

	return &out, nil
}

// GetIncident returns the incident from the JSM Incidents API (issueID must be the numeric Jira issue id).
func (c *Client) GetIncident(cloudID, issueID string) (map[string]any, error) {
	u := c.incidentAPIURL(cloudID, fmt.Sprintf("/incident/%s", url.PathEscape(issueID)))
	responseBody, err := c.execRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	var out map[string]any
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, fmt.Errorf("parse get incident response: %w", err)
	}

	return out, nil
}

// DeleteIncident deletes an incident via the JSM Incidents API.
func (c *Client) DeleteIncident(cloudID, issueID string) error {
	u := c.incidentAPIURL(cloudID, fmt.Sprintf("/incident/%s", url.PathEscape(issueID)))
	_, err := c.execRequest(http.MethodDelete, u, nil)
	return err
}

// ResolveNumericIssueID returns the numeric issue id for the Incidents API path. If ref is all digits it is
// returned unchanged; otherwise ref is treated as an issue key and resolved with the Jira platform REST API.
func (c *Client) ResolveNumericIssueID(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("issue reference is required")
	}
	if isNumericIssueRef(ref) {
		return ref, nil
	}

	issue, err := c.GetIssue(ref)
	if err != nil {
		return "", fmt.Errorf("resolve issue key %q: %w", ref, err)
	}
	return issue.ID, nil
}

func isNumericIssueRef(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}
