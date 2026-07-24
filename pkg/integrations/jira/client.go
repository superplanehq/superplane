package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

// Client speaks to Jira Cloud through Atlassian's OAuth API proxy (api.atlassian.com/ex/jira/{cloudId}/...).
type Client struct {
	CloudID     string
	AccessToken string
	http        core.HTTPContext
	integration core.IntegrationContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	cloudID, err := ctx.Properties().GetString(PropertyCloudID)
	if err != nil || cloudID == "" {
		return nil, fmt.Errorf("missing Jira cloud id; connect Jira via OAuth first")
	}

	accessToken, err := ctx.Secrets().Get(SecretOAuthAccessToken)
	if err != nil || accessToken == "" {
		return nil, fmt.Errorf("missing Jira OAuth access token; connect Jira via OAuth first")
	}

	return &Client{
		CloudID:     cloudID,
		AccessToken: accessToken,
		http:        httpCtx,
		integration: ctx,
	}, nil
}

func (c *Client) apiURL(path string) string {
	return atlassianAPIProxyHost + "/" + c.CloudID + path
}

// execRequest sends the request with a Bearer token, refreshing once and retrying on a 401.
func (c *Client) execRequest(method, requestURL string, body io.Reader) ([]byte, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("error reading request body: %v", err)
		}
	}

	responseBody, status, err := c.doRequest(method, requestURL, bodyBytes)
	if err != nil {
		return nil, err
	}

	if status == http.StatusUnauthorized {
		if refreshErr := c.refresh(); refreshErr != nil {
			return nil, fmt.Errorf("request got 401 and token refresh failed: %w", refreshErr)
		}
		responseBody, status, err = c.doRequest(method, requestURL, bodyBytes)
		if err != nil {
			return nil, err
		}
	}

	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("request got %d code: %s", status, string(responseBody))
	}

	return responseBody, nil
}

func (c *Client) doRequest(method, requestURL string, body []byte) ([]byte, int, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, requestURL, reader)
	if err != nil {
		return nil, 0, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("error reading body: %v", err)
	}

	return responseBody, res.StatusCode, nil
}

// refresh exchanges the stored refresh token for a new access/refresh token pair and persists them.
func (c *Client) refresh() error {
	if c.integration == nil {
		return fmt.Errorf("no integration context available to refresh the OAuth token")
	}

	clientID, err := c.integration.Properties().GetString(PropertyClientID)
	if err != nil {
		return fmt.Errorf("error reading OAuth client id: %w", err)
	}
	clientSecret, err := c.integration.Secrets().Get(SecretOAuthClientSecret)
	if err != nil {
		return fmt.Errorf("error reading OAuth client secret: %w", err)
	}
	refreshToken, err := c.integration.Secrets().Get(SecretOAuthRefreshToken)
	if err != nil {
		return fmt.Errorf("error reading OAuth refresh token: %w", err)
	}

	token, err := refreshAccessToken(c.http, clientID, clientSecret, refreshToken)
	if err != nil {
		return err
	}

	if err := c.integration.Secrets().Update(SecretOAuthAccessToken, token.AccessToken); err != nil {
		return fmt.Errorf("error storing refreshed access token: %w", err)
	}
	if token.RefreshToken != "" {
		if err := c.integration.Secrets().Update(SecretOAuthRefreshToken, token.RefreshToken); err != nil {
			return fmt.Errorf("error storing refreshed refresh token: %w", err)
		}
	}

	c.AccessToken = token.AccessToken
	return nil
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

// Status represents a Jira workflow status. Category is the normalized
// statusCategory value used by Jira's workflow APIs: "TODO",
// "IN_PROGRESS", "DONE", or "UNDEFINED". It is populated from either the
// flat string returned by /rest/api/3/statuses/search or the nested
// statusCategory.key returned by /rest/api/3/project/{key}/statuses.
type Status struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"-"`
}

type projectStatusCategory struct {
	Key string `json:"key"`
}

type projectStatus struct {
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	StatusCategory projectStatusCategory `json:"statusCategory"`
}

type projectStatusesIssueType struct {
	Statuses []projectStatus `json:"statuses"`
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
			statuses = append(statuses, Status{
				ID:       s.ID,
				Name:     s.Name,
				Category: normalizeStatusCategoryKey(s.StatusCategory.Key),
			})
		}
	}
	return statuses, nil
}

type globalStatusesPage struct {
	IsLast   bool           `json:"isLast"`
	NextPage string         `json:"nextPage"`
	Values   []globalStatus `json:"values"`
}

type globalStatus struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	StatusCategory string `json:"statusCategory"`
}

// ListGlobalStatuses returns every workflow status visible to the caller via
// /rest/api/3/statuses/search. Used by the issueStatus resource picker when
// no project context is available (e.g. when defining a global workflow) and
// to look up status categories at workflow-create time.
func (c *Client) ListGlobalStatuses() ([]Status, error) {
	endpoint := c.apiURL("/rest/api/3/statuses/search?maxResults=200")
	seen := map[string]bool{}
	statuses := []Status{}

	for endpoint != "" {
		body, err := c.execRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}

		var page globalStatusesPage
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("error parsing statuses search response: %v", err)
		}

		for _, s := range page.Values {
			if seen[s.Name] {
				continue
			}
			seen[s.Name] = true
			statuses = append(statuses, Status{
				ID:       s.ID,
				Name:     s.Name,
				Category: normalizeStatusCategoryName(s.StatusCategory),
			})
		}

		if page.IsLast || page.NextPage == "" {
			break
		}
		endpoint = page.NextPage
	}

	return statuses, nil
}

// normalizeStatusCategoryKey converts the lowercase "key" values returned by
// /rest/api/3/project/{key}/statuses (new/indeterminate/done/undefined) into
// the upper-case category names accepted by /rest/api/3/workflows/create.
func normalizeStatusCategoryKey(key string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "new":
		return "TODO"
	case "indeterminate":
		return "IN_PROGRESS"
	case "done":
		return "DONE"
	default:
		return "UNDEFINED"
	}
}

// normalizeStatusCategoryName accepts the category names returned by
// /rest/api/3/statuses/search (already TODO/IN_PROGRESS/DONE/UNDEFINED) and
// returns the canonical value used by workflow create requests.
func normalizeStatusCategoryName(name string) string {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "TODO":
		return "TODO"
	case "IN_PROGRESS":
		return "IN_PROGRESS"
	case "DONE":
		return "DONE"
	default:
		return "UNDEFINED"
	}
}

type Transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   Status `json:"to"`
	// Fields lists the fields that are present on this transition's screen,
	// keyed by Jira field id (for example "resolution", "customfield_10010").
	// Populated by GetIssueTransitions because we always pass
	// expand=transitions.fields — needed to know whether a transition supports
	// setting fields like resolution. Empty when no screen is configured.
	Fields map[string]any `json:"fields,omitempty"`
}

// HasField reports whether this transition's screen includes the named Jira
// field id. Used to avoid the "Field 'X' cannot be set. It is not on the
// appropriate screen" error from Jira when the user supplies a field that
// the chosen transition doesn't actually accept.
func (t Transition) HasField(fieldID string) bool {
	if t.Fields == nil {
		return false
	}
	_, ok := t.Fields[strings.TrimSpace(fieldID)]
	return ok
}

type transitionsResponse struct {
	Transitions []Transition `json:"transitions"`
}

// GetIssueTransitions returns the transitions available from an issue's
// current workflow state, expanded with each transition's per-screen fields
// so callers can decide whether a given field (for example resolution) can
// be set during the transition.
func (c *Client) GetIssueTransitions(issueKey string) ([]Transition, error) {
	query := url.Values{}
	query.Set("expand", "transitions.fields")
	endpoint := c.apiURL("/rest/api/3/issue/" + url.PathEscape(issueKey) + "/transitions?" + query.Encode())

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

// DoTransitionWithOptions advances an issue and optionally applies
// transition-scoped fields. The caller is responsible for ensuring that any
// fields it sets are actually on the chosen transition's screen — Jira
// returns a 400 with "Field 'X' cannot be set. It is not on the appropriate
// screen, or unknown." otherwise. applyStatusWithOptions handles that
// pre-check.
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

// WorkflowSchemeDetail is returned by GET /rest/api/3/workflowscheme/{id}. It
// describes which workflow is used per issue type. Used to resolve the
// workflow bound to an issue (issue type ID -> workflow name).
type WorkflowSchemeDetail struct {
	ID                FlexibleString    `json:"id"`
	Name              string            `json:"name"`
	DefaultWorkflow   string            `json:"defaultWorkflow"`
	IssueTypeMappings map[string]string `json:"issueTypeMappings"`
}

// GetWorkflowScheme returns details for one workflow scheme.
func (c *Client) GetWorkflowScheme(schemeID string) (*WorkflowSchemeDetail, error) {
	endpoint := c.apiURL("/rest/api/3/workflowscheme/" + url.PathEscape(schemeID))
	body, err := c.execRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	var out WorkflowSchemeDetail
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("error parsing workflow scheme response: %v", err)
	}
	if out.IssueTypeMappings == nil {
		out.IssueTypeMappings = map[string]string{}
	}
	return &out, nil
}

// projectWorkflowSchemeAssignment captures one entry in the response of
// /rest/api/3/workflowscheme/project — the assignment of a workflow scheme
// (which can be inlined as workflowScheme) to a project.
type projectWorkflowSchemeAssignment struct {
	ProjectIDs     []string `json:"projectIds"`
	WorkflowScheme struct {
		ID                FlexibleString    `json:"id"`
		Name              string            `json:"name"`
		DefaultWorkflow   string            `json:"defaultWorkflow,omitempty"`
		IssueTypeMappings map[string]string `json:"issueTypeMappings,omitempty"`
	} `json:"workflowScheme"`
}

type projectWorkflowSchemesResponse struct {
	Values []projectWorkflowSchemeAssignment `json:"values"`
}

// GetWorkflowSchemeForProject returns the workflow scheme assigned to a
// company-managed project. For team-managed projects Jira may return an empty
// list (their workflow lives directly on the project), so callers should
// handle a nil result.
func (c *Client) GetWorkflowSchemeForProject(projectID string) (*WorkflowSchemeDetail, error) {
	query := url.Values{}
	query.Set("projectId", projectID)
	endpoint := c.apiURL("/rest/api/3/workflowscheme/project?" + query.Encode())

	body, err := c.execRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	var resp projectWorkflowSchemesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parsing project workflow scheme response: %v", err)
	}

	for _, assignment := range resp.Values {
		schemeID := strings.TrimSpace(assignment.WorkflowScheme.ID.String())
		if schemeID != "" {
			// Resolve the full scheme details (the inlined version omits issueTypeMappings).
			return c.GetWorkflowScheme(schemeID)
		}
		// Jira omits the scheme id for the built-in Default Workflow Scheme
		// (common for company-managed projects that never customized it). The
		// inlined object still carries the default workflow and any per-issue-type
		// mappings, so fall back to it instead of dropping the workflow entirely.
		if defaultWorkflow := strings.TrimSpace(assignment.WorkflowScheme.DefaultWorkflow); defaultWorkflow != "" {
			mappings := assignment.WorkflowScheme.IssueTypeMappings
			if mappings == nil {
				mappings = map[string]string{}
			}
			return &WorkflowSchemeDetail{
				ID:                assignment.WorkflowScheme.ID,
				Name:              assignment.WorkflowScheme.Name,
				DefaultWorkflow:   defaultWorkflow,
				IssueTypeMappings: mappings,
			}, nil
		}
	}

	return nil, nil
}

type workflowSearchEntry struct {
	ID struct {
		Name string `json:"name"`
	} `json:"id"`
	Statuses []globalStatus `json:"statuses"`
}

type workflowSearchResponse struct {
	Values []workflowSearchEntry `json:"values"`
}

// GetWorkflowStatusesByName returns the statuses of the workflow with the
// given exact name. Jira's /rest/api/3/workflow/search?workflowName=... does
// a prefix-style match server-side and can return multiple workflows, so we
// filter for an exact name match here and refuse to guess if none of the
// returned workflows match — returning a different workflow's statuses
// would silently mis-describe the issue's state machine.
func (c *Client) GetWorkflowStatusesByName(workflowName string) ([]Status, error) {
	query := url.Values{}
	query.Set("workflowName", workflowName)
	query.Set("expand", "statuses")
	endpoint := c.apiURL("/rest/api/3/workflow/search?" + query.Encode())

	body, err := c.execRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	var out workflowSearchResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("error parsing workflow search response: %v", err)
	}
	for _, entry := range out.Values {
		if entry.ID.Name == workflowName {
			return statusesFromGlobal(entry.Statuses), nil
		}
	}
	return nil, fmt.Errorf("workflow %q not found", workflowName)
}

func statusesFromGlobal(raw []globalStatus) []Status {
	statuses := make([]Status, 0, len(raw))
	for _, s := range raw {
		statuses = append(statuses, Status{
			ID:       s.ID,
			Name:     s.Name,
			Category: normalizeStatusCategoryName(s.StatusCategory),
		})
	}
	return statuses
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

// IssueWebhookRegistration is one entry of the "webhooks" array in a dynamic webhook registration request.
type IssueWebhookRegistration struct {
	JQLFilter string   `json:"jqlFilter,omitempty"`
	Events    []string `json:"events"`
}

type createIssueWebhookRequest struct {
	URL      string                     `json:"url"`
	Webhooks []IssueWebhookRegistration `json:"webhooks"`
}

type createIssueWebhookResult struct {
	CreatedWebhookID *int64   `json:"createdWebhookId,omitempty"`
	Errors           []string `json:"errors,omitempty"`
}

// createIssueWebhookResponse wraps results under "webhookRegistrationResult", not a bare array.
type createIssueWebhookResponse struct {
	WebhookRegistrationResult []createIssueWebhookResult `json:"webhookRegistrationResult"`
}

// CreateIssueWebhook registers a dynamic webhook for issue events, scoped by an optional JQL filter.
func (c *Client) CreateIssueWebhook(callbackURL, jqlFilter string, events []string) (int64, error) {
	req := createIssueWebhookRequest{
		URL: callbackURL,
		Webhooks: []IssueWebhookRegistration{
			{JQLFilter: jqlFilter, Events: events},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return 0, fmt.Errorf("marshal create webhook request: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, c.apiURL("/rest/api/3/webhook"), bytes.NewReader(body))
	if err != nil {
		return 0, err
	}

	results, err := parseCreateIssueWebhookResponse(responseBody)
	if err != nil {
		return 0, err
	}
	if len(results[0].Errors) > 0 {
		return 0, fmt.Errorf("failed to create webhook: %s", strings.Join(results[0].Errors, "; "))
	}
	if results[0].CreatedWebhookID == nil {
		return 0, fmt.Errorf("create webhook response missing createdWebhookId: %s", string(responseBody))
	}

	return *results[0].CreatedWebhookID, nil
}

// parseCreateIssueWebhookResponse accepts either the wrapped shape or a bare array.
func parseCreateIssueWebhookResponse(responseBody []byte) ([]createIssueWebhookResult, error) {
	var wrapped createIssueWebhookResponse
	if err := json.Unmarshal(responseBody, &wrapped); err == nil && len(wrapped.WebhookRegistrationResult) > 0 {
		return wrapped.WebhookRegistrationResult, nil
	}

	var results []createIssueWebhookResult
	if err := json.Unmarshal(responseBody, &results); err == nil && len(results) > 0 {
		return results, nil
	}

	return nil, fmt.Errorf("unrecognized create webhook response: %s", string(responseBody))
}

// DeleteIssueWebhooks removes previously-registered dynamic webhooks by id.
func (c *Client) DeleteIssueWebhooks(webhookIDs []int64) error {
	if len(webhookIDs) == 0 {
		return nil
	}

	body, err := json.Marshal(map[string][]int64{"webhookIds": webhookIDs})
	if err != nil {
		return fmt.Errorf("marshal delete webhook request: %w", err)
	}

	_, err = c.execRequest(http.MethodDelete, c.apiURL("/rest/api/3/webhook"), bytes.NewReader(body))
	return err
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

	u := c.apiURL("/rest/api/3/search")
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

	base := c.apiURL("")
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
	base := c.apiURL("")
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
	base := c.apiURL("")
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
	base := c.apiURL("")
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
	base := c.apiURL("")
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
	base := c.apiURL("")
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
	base := c.apiURL("")
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
	base := c.apiURL("")
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

	base := c.apiURL("")
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
	base := c.apiURL("")
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
	base := c.apiURL("")
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
	base := c.apiURL("")
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

	base := c.apiURL("")
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

	base := c.apiURL("")
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
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

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
	base := c.apiURL("")
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

// OpsTeam is one JSM Operations team from GET /v1/teams.
type OpsTeam struct {
	TeamID   string `json:"teamId"`
	TeamName string `json:"teamName"`
}

type listOpsTeamsResponse struct {
	PlatformTeams []OpsTeam `json:"platformTeams"`
}

// Heartbeat is a JSM Operations heartbeat monitor.
type Heartbeat struct {
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	Interval      int      `json:"interval"`
	IntervalUnit  string   `json:"intervalUnit"`
	Enabled       bool     `json:"enabled"`
	Status        string   `json:"status,omitempty"`
	OwnerTeamID   string   `json:"ownerTeamId,omitempty"`
	AlertMessage  string   `json:"alertMessage,omitempty"`
	AlertTags     []string `json:"alertTags,omitempty"`
	AlertPriority string   `json:"alertPriority,omitempty"`
}

type heartbeatLinks struct {
	Next string `json:"next,omitempty"`
}

type heartbeatPaginatedResponse struct {
	Values []Heartbeat     `json:"values"`
	Links  *heartbeatLinks `json:"links,omitempty"`
}

// CreateHeartbeatRequest is the JSON body for POST .../heartbeats.
type CreateHeartbeatRequest struct {
	Name          string   `json:"name"`
	Description   string   `json:"description,omitempty"`
	Interval      int      `json:"interval"`
	IntervalUnit  string   `json:"intervalUnit"`
	Enabled       *bool    `json:"enabled,omitempty"`
	AlertMessage  string   `json:"alertMessage,omitempty"`
	AlertTags     []string `json:"alertTags,omitempty"`
	AlertPriority string   `json:"alertPriority,omitempty"`
}

// UpdateHeartbeatRequest is the JSON body for PATCH .../heartbeats?name=...
type UpdateHeartbeatRequest struct {
	Description   string   `json:"description,omitempty"`
	Interval      *int     `json:"interval,omitempty"`
	IntervalUnit  string   `json:"intervalUnit,omitempty"`
	Enabled       *bool    `json:"enabled,omitempty"`
	AlertMessage  string   `json:"alertMessage,omitempty"`
	AlertTags     []string `json:"alertTags,omitempty"`
	AlertPriority string   `json:"alertPriority,omitempty"`
}

// PingHeartbeatResponse is returned when a heartbeat ping succeeds.
type PingHeartbeatResponse struct {
	Message string `json:"message"`
}

func (c *Client) opsAPIURL(cloudID, pathSuffix string) string {
	return fmt.Sprintf("%s/jsm/ops/api/%s/v1%s", atlassianIncidentAPIHost, cloudID, pathSuffix)
}

// ListOpsTeams returns JSM Operations teams visible to the authenticated user.
func (c *Client) ListOpsTeams(cloudID string) ([]OpsTeam, error) {
	u := c.opsAPIURL(cloudID, "/teams")
	responseBody, err := c.execRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	return parseOpsTeamsResponse(responseBody)
}

func parseOpsTeamsResponse(responseBody []byte) ([]OpsTeam, error) {
	body := bytes.TrimSpace(responseBody)
	if len(body) == 0 || bytes.Equal(body, []byte("null")) {
		return []OpsTeam{}, nil
	}

	// Live Jira Cloud often returns a JSON array; OpenAPI also documents { "platformTeams": [...] }.
	if body[0] == '[' {
		var teams []OpsTeam
		if err := json.Unmarshal(body, &teams); err != nil {
			return nil, fmt.Errorf("parse ops teams array response: %w", err)
		}
		return normalizeOpsTeams(teams), nil
	}

	var wrapped listOpsTeamsResponse
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("parse ops teams response: %w", err)
	}
	if wrapped.PlatformTeams == nil {
		return []OpsTeam{}, nil
	}
	return normalizeOpsTeams(wrapped.PlatformTeams), nil
}

func normalizeOpsTeams(teams []OpsTeam) []OpsTeam {
	out := make([]OpsTeam, 0, len(teams))
	for _, team := range teams {
		id := strings.TrimSpace(team.TeamID)
		name := strings.TrimSpace(team.TeamName)
		if id == "" && name == "" {
			continue
		}
		if name == "" {
			name = id
		}
		out = append(out, OpsTeam{TeamID: id, TeamName: name})
	}
	return out
}

// ListHeartbeats returns all heartbeats for an operations team, following links.next pagination.
func (c *Client) ListHeartbeats(cloudID, teamID string) ([]Heartbeat, error) {
	nextURL := c.opsAPIURL(cloudID, fmt.Sprintf("/teams/%s/heartbeats", url.PathEscape(teamID)))
	var all []Heartbeat
	const maxPages = 100
	for range maxPages {
		responseBody, err := c.execRequest(http.MethodGet, nextURL, nil)
		if err != nil {
			return nil, err
		}

		var page heartbeatPaginatedResponse
		if err := json.Unmarshal(responseBody, &page); err != nil {
			return nil, fmt.Errorf("parse heartbeats response: %w", err)
		}

		all = append(all, page.Values...)

		if page.Links == nil || page.Links.Next == "" || len(page.Values) == 0 {
			break
		}

		if strings.HasPrefix(page.Links.Next, "http") {
			nextURL = page.Links.Next
		} else {
			nextURL = atlassianIncidentAPIHost + page.Links.Next
		}
	}
	return all, nil
}

// CreateHeartbeat creates a heartbeat in JSM Operations.
func (c *Client) CreateHeartbeat(cloudID, teamID string, req *CreateHeartbeatRequest) (*Heartbeat, error) {
	u := c.opsAPIURL(cloudID, fmt.Sprintf("/teams/%s/heartbeats", url.PathEscape(teamID)))
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal create heartbeat body: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var out Heartbeat
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, fmt.Errorf("parse create heartbeat response: %w", err)
	}
	return &out, nil
}

// PingHeartbeat sends a ping for the named heartbeat.
func (c *Client) PingHeartbeat(cloudID, teamID, name string) (*PingHeartbeatResponse, error) {
	q := url.Values{}
	q.Set("name", name)
	u := c.opsAPIURL(cloudID, fmt.Sprintf("/teams/%s/heartbeats/ping?%s", url.PathEscape(teamID), q.Encode()))

	responseBody, err := c.execRequest(http.MethodPost, u, nil)
	if err != nil {
		return nil, err
	}

	var out PingHeartbeatResponse
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, fmt.Errorf("parse ping heartbeat response: %w", err)
	}
	return &out, nil
}

// UpdateHeartbeat updates a heartbeat identified by name (name cannot be changed).
func (c *Client) UpdateHeartbeat(cloudID, teamID, name string, req *UpdateHeartbeatRequest) (*Heartbeat, error) {
	q := url.Values{}
	q.Set("name", name)
	u := c.opsAPIURL(cloudID, fmt.Sprintf("/teams/%s/heartbeats?%s", url.PathEscape(teamID), q.Encode()))
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal update heartbeat body: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPatch, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var out Heartbeat
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return nil, fmt.Errorf("parse update heartbeat response: %w", err)
	}
	return &out, nil
}

// DeleteHeartbeat deletes a heartbeat by name.
func (c *Client) DeleteHeartbeat(cloudID, teamID, name string) error {
	q := url.Values{}
	q.Set("name", name)
	u := c.opsAPIURL(cloudID, fmt.Sprintf("/teams/%s/heartbeats?%s", url.PathEscape(teamID), q.Encode()))
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

const (
	opsAlertPollMaxAttempts = 42
	opsAlertPollInterval    = 450 * time.Millisecond
)

// OpsAlertResponder is a responder or visibleTo entry for the Jira Service Management Ops Alerts API.
type OpsAlertResponder struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// OpsCreateAlertRequest is the JSON body for POST /v1/alerts.
type OpsCreateAlertRequest struct {
	Message         string              `json:"message"`
	Responders      []OpsAlertResponder `json:"responders,omitempty"`
	VisibleTo       []OpsAlertResponder `json:"visibleTo,omitempty"`
	Note            string              `json:"note,omitempty"`
	Alias           string              `json:"alias,omitempty"`
	Entity          string              `json:"entity,omitempty"`
	Source          string              `json:"source,omitempty"`
	Tags            []string            `json:"tags,omitempty"`
	Actions         []string            `json:"actions,omitempty"`
	Description     string              `json:"description,omitempty"`
	Priority        string              `json:"priority,omitempty"`
	ExtraProperties map[string]any      `json:"extraProperties,omitempty"`
}

// OpsAsyncSuccessResponse is returned by create/delete and several mutating Ops alert endpoints (often with HTTP 202).
type OpsAsyncSuccessResponse struct {
	Result    string  `json:"result"`
	RequestID string  `json:"requestId"`
	Took      float64 `json:"took"`
}

func (c *Client) opsAlertsBasePath(cloudID string) string {
	return fmt.Sprintf(
		"https://api.atlassian.com/jsm/ops/api/%s/v1/alerts",
		url.PathEscape(strings.TrimSpace(cloudID)),
	)
}

func (c *Client) execOpsAlertJSON(method, fullURL string, body io.Reader) ([]byte, error) {
	return c.execRequest(method, fullURL, body)
}

func parseOpsAsyncSuccess(body []byte) (*OpsAsyncSuccessResponse, error) {
	var out OpsAsyncSuccessResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse ops alert response: %w", err)
	}
	return &out, nil
}

// CreateOpsAlert creates an alert via the JSM Ops REST API (asynchronous processing; see SuccessResponse).
func (c *Client) CreateOpsAlert(cloudID string, req *OpsCreateAlertRequest) (*OpsAsyncSuccessResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("create alert request is required")
	}
	u := c.opsAlertsBasePath(cloudID)
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal create alert body: %w", err)
	}
	body, err := c.execOpsAlertJSON(http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	return parseOpsAsyncSuccess(body)
}

// GetOpsAlert returns one alert by id (GET /v1/alerts/{id}).
func (c *Client) GetOpsAlert(cloudID, alertID string) (map[string]any, error) {
	alertID = strings.TrimSpace(alertID)
	if alertID == "" {
		return nil, fmt.Errorf("alert id is required")
	}
	u := fmt.Sprintf("%s/%s", c.opsAlertsBasePath(cloudID), url.PathEscape(alertID))
	body, err := c.execOpsAlertJSON(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse get alert response: %w", err)
	}
	return out, nil
}

// DeleteOpsAlert deletes an alert by id (DELETE /v1/alerts/{id}).
func (c *Client) DeleteOpsAlert(cloudID, alertID string) (*OpsAsyncSuccessResponse, error) {
	alertID = strings.TrimSpace(alertID)
	if alertID == "" {
		return nil, fmt.Errorf("alert id is required")
	}
	u := fmt.Sprintf("%s/%s", c.opsAlertsBasePath(cloudID), url.PathEscape(alertID))
	body, err := c.execOpsAlertJSON(http.MethodDelete, u, nil)
	if err != nil {
		return nil, err
	}
	return parseOpsAsyncSuccess(body)
}

// AssignOpsAlert assigns an alert to a user via POST /v1/alerts/{id}/assign.
func (c *Client) AssignOpsAlert(cloudID, alertID, accountID string) (*OpsAsyncSuccessResponse, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, fmt.Errorf("assignee account id is required")
	}
	payload, err := json.Marshal(map[string]string{"accountId": accountID})
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/%s/assign", c.opsAlertsBasePath(cloudID), url.PathEscape(strings.TrimSpace(alertID)))
	body, err := c.execOpsAlertJSON(http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	return parseOpsAsyncSuccess(body)
}

// AcknowledgeOpsAlert acknowledges an alert (POST /v1/alerts/{id}/acknowledge).
func (c *Client) AcknowledgeOpsAlert(cloudID, alertID string) (*OpsAsyncSuccessResponse, error) {
	u := fmt.Sprintf("%s/%s/acknowledge", c.opsAlertsBasePath(cloudID), url.PathEscape(strings.TrimSpace(alertID)))
	body, err := c.execOpsAlertJSON(http.MethodPost, u, nil)
	if err != nil {
		return nil, err
	}
	return parseOpsAsyncSuccess(body)
}

// CloseOpsAlert closes an alert (POST /v1/alerts/{id}/close).
func (c *Client) CloseOpsAlert(cloudID, alertID string) (*OpsAsyncSuccessResponse, error) {
	u := fmt.Sprintf("%s/%s/close", c.opsAlertsBasePath(cloudID), url.PathEscape(strings.TrimSpace(alertID)))
	body, err := c.execOpsAlertJSON(http.MethodPost, u, nil)
	if err != nil {
		return nil, err
	}
	return parseOpsAsyncSuccess(body)
}

// PatchOpsAlertPriority updates alert priority (PATCH /v1/alerts/{id}/priority).
func (c *Client) PatchOpsAlertPriority(cloudID, alertID, priority string) (*OpsAsyncSuccessResponse, error) {
	payload, err := json.Marshal(map[string]string{"priority": strings.TrimSpace(priority)})
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/%s/priority", c.opsAlertsBasePath(cloudID), url.PathEscape(strings.TrimSpace(alertID)))
	body, err := c.execOpsAlertJSON(http.MethodPatch, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	return parseOpsAsyncSuccess(body)
}

// PatchOpsAlertMessage updates the alert message (PATCH /v1/alerts/{id}/message).
func (c *Client) PatchOpsAlertMessage(cloudID, alertID, message string) (*OpsAsyncSuccessResponse, error) {
	payload, err := json.Marshal(map[string]string{"message": message})
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/%s/message", c.opsAlertsBasePath(cloudID), url.PathEscape(strings.TrimSpace(alertID)))
	body, err := c.execOpsAlertJSON(http.MethodPatch, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	return parseOpsAsyncSuccess(body)
}

// PatchOpsAlertDescription updates the alert description (PATCH /v1/alerts/{id}/description).
func (c *Client) PatchOpsAlertDescription(cloudID, alertID, description string) (*OpsAsyncSuccessResponse, error) {
	payload, err := json.Marshal(map[string]string{"description": description})
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/%s/description", c.opsAlertsBasePath(cloudID), url.PathEscape(strings.TrimSpace(alertID)))
	body, err := c.execOpsAlertJSON(http.MethodPatch, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	return parseOpsAsyncSuccess(body)
}

// AddOpsAlertNote adds a note (POST /v1/alerts/{id}/notes).
func (c *Client) AddOpsAlertNote(cloudID, alertID, note string) (map[string]any, error) {
	payload, err := json.Marshal(map[string]string{"note": note})
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("%s/%s/notes", c.opsAlertsBasePath(cloudID), url.PathEscape(strings.TrimSpace(alertID)))
	body, err := c.execOpsAlertJSON(http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse add alert note response: %w", err)
	}
	return out, nil
}

// PatchOpsAlertNote updates an existing note (PATCH /v1/alerts/{alertId}/notes/{id}).
func (c *Client) PatchOpsAlertNote(cloudID, alertID, noteID, note string) (map[string]any, error) {
	payload, err := json.Marshal(map[string]string{"note": note})
	if err != nil {
		return nil, err
	}
	base := c.opsAlertsBasePath(cloudID)
	u := fmt.Sprintf(
		"%s/%s/notes/%s",
		base,
		url.PathEscape(strings.TrimSpace(alertID)),
		url.PathEscape(strings.TrimSpace(noteID)),
	)
	body, err := c.execOpsAlertJSON(http.MethodPatch, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse update alert note response: %w", err)
	}
	return out, nil
}

type opsListAlertsEnvelope struct {
	Values []map[string]any `json:"values"`
}

// ListOpsAlerts returns recent Ops alerts from GET /v1/alerts (newest batches only; capped by size).
func (c *Client) ListOpsAlerts(cloudID string, size int) ([]map[string]any, error) {
	if size <= 0 || size > 100 {
		size = 100
	}
	u := fmt.Sprintf("%s?size=%d", c.opsAlertsBasePath(cloudID), size)
	body, err := c.execOpsAlertJSON(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var env opsListAlertsEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parse list alerts response: %w", err)
	}
	if env.Values == nil {
		env.Values = []map[string]any{}
	}
	return env.Values, nil
}

// GetOpsAlertRequestStatus returns async request status GET /v1/alerts/requests/{id}.
func (c *Client) GetOpsAlertRequestStatus(cloudID, requestID string) (map[string]any, error) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return nil, fmt.Errorf("request id is required")
	}
	u := fmt.Sprintf("%s/requests/%s", c.opsAlertsBasePath(cloudID), url.PathEscape(requestID))
	body, err := c.execOpsAlertJSON(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parse alert request status: %w", err)
	}
	return out, nil
}

func opsAlertStringField(m map[string]any, key string) string {
	raw, ok := m[key]
	if !ok || raw == nil {
		return ""
	}
	switch s := raw.(type) {
	case string:
		return strings.TrimSpace(s)
	default:
		return strings.TrimSpace(fmt.Sprint(s))
	}
}

func opsAlertStatusIsSuccessful(status map[string]any) bool {
	v, ok := status["isSuccess"]
	if !ok {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	default:
		return false
	}
}

func opsAlertStatusIsProcessed(status map[string]any) bool {
	return opsAlertStringField(status, "processedAt") != ""
}

func opsAlertStatusAlertID(status map[string]any) string {
	return opsAlertStringField(status, "alertId")
}

// opsAlertRequestStatusIndicatesFailure reports API-level failure once a request is processed.
func opsAlertRequestStatusIndicatesFailure(status map[string]any) bool {
	if !opsAlertStatusIsSuccessful(status) {
		return true
	}
	msg := strings.ToLower(opsAlertStringField(status, "status"))
	if msg == "" {
		return false
	}
	for _, frag := range []string{
		"fail",
		"error",
		"invalid",
		"reject",
		"does not exist",
		"not found",
		"unable",
		"denied",
	} {
		if strings.Contains(msg, frag) {
			return true
		}
	}
	return false
}

func opsAlertAsyncRequestError(status map[string]any, requestID string) error {
	detail := opsAlertStringField(status, "status")
	if detail == "" {
		detail = "request did not succeed"
	}
	return fmt.Errorf("async Ops alert request %s failed: %s", requestID, detail)
}

// ResolveAlertIDAfterOpsRequest polls GET alerts/requests until the async request is fully processed.
func (c *Client) ResolveAlertIDAfterOpsRequest(cloudID, requestID, knownAlertID string) (string, error) {
	requestID = strings.TrimSpace(requestID)
	knownAlertID = strings.TrimSpace(knownAlertID)

	if requestID == "" {
		if knownAlertID == "" {
			return "", fmt.Errorf("no async request id to resolve Ops alert outcome")
		}
		return knownAlertID, nil
	}

	for attempt := range opsAlertPollMaxAttempts {
		if attempt > 0 {
			time.Sleep(opsAlertPollInterval)
		}

		status, err := c.GetOpsAlertRequestStatus(cloudID, requestID)
		if err != nil {
			return "", err
		}

		if !opsAlertStatusIsProcessed(status) {
			continue
		}

		if opsAlertRequestStatusIndicatesFailure(status) {
			return "", opsAlertAsyncRequestError(status, requestID)
		}

		if knownAlertID != "" {
			return knownAlertID, nil
		}

		aid := opsAlertStatusAlertID(status)
		if aid == "" {
			return "", fmt.Errorf(
				"async Ops alert request %s finished without an alert id (status: %s)",
				requestID,
				opsAlertStringField(status, "status"),
			)
		}
		return aid, nil
	}

	return "", fmt.Errorf(
		"timed out waiting for Ops alert async request %s to finish processing",
		requestID,
	)
}
