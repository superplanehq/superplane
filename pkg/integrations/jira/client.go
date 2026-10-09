package jira

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	// AuthorizeURL is where the user approves the OAuth application.
	AuthorizeURL = "https://auth.atlassian.com/authorize"

	// TokenURL exchanges authorization codes and refresh tokens for access tokens.
	TokenURL = "https://auth.atlassian.com/oauth/token"

	// AccessibleResourcesURL lists the Jira Cloud sites the OAuth grant has access to.
	AccessibleResourcesURL = "https://api.atlassian.com/oauth/token/accessible-resources"

	// APIProxyHost is how OAuth apps reach Jira's REST APIs - the site's own domain rejects OAuth bearer tokens.
	APIProxyHost = "https://api.atlassian.com/ex/jira"

	// offline_access is required for token refresh but is not selectable in the Developer Console.
	coreScopeList = "read:jira-work write:jira-work manage:jira-webhook read:jira-user " +
		"read:issue-details:jira offline_access"
)

// Client speaks to Jira Cloud through Atlassian's OAuth API proxy (api.atlassian.com/ex/jira/{cloudId}/...).
type Client struct {
	CloudID     string
	AccessToken string
	http        core.HTTPContext
	integration core.IntegrationContext
}

func NewClient(httpCtx core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	cloudID, err := cloudIDFromIntegration(ctx)
	if err != nil {
		return nil, err
	}

	accessToken, err := findSecret(ctx, SecretOAuthAccessToken)
	if err != nil {
		return nil, fmt.Errorf("error reading access token: %v", err)
	}
	if accessToken == "" {
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
	return APIProxyHost + "/" + c.CloudID + path
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
		if err := c.recoverFromUnauthorized(); err != nil {
			return nil, err
		}
		responseBody, status, err = c.doRequest(method, requestURL, bodyBytes)
		if err != nil {
			return nil, err
		}
	}

	if status < 200 || status >= 300 {
		return nil, &APIError{StatusCode: status, Body: string(responseBody)}
	}

	return responseBody, nil
}

// APIError is a non-2xx response from Jira. Consumers use StatusCode to
// decide whether to retry.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request got %d code: %s", e.StatusCode, e.Body)
}

// IsRetryableAPIError reports whether the consumer should nack the message
// so Tackle redelivers it. Rate limits, request timeouts, server errors,
// and transport failures retry. Client errors such as 401, 403, and 404
// do not.
func IsRetryableAPIError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusTooManyRequests ||
			apiErr.StatusCode == http.StatusRequestTimeout ||
			apiErr.StatusCode >= http.StatusInternalServerError
	}

	message := err.Error()
	return strings.Contains(message, "error executing request") ||
		strings.Contains(message, "error reading body") ||
		strings.Contains(message, "error reading request body")
}

// recoverFromUnauthorized refreshes the access token after a 401. Atlassian's refresh tokens are
// single-use, so a concurrent Sync or request can rotate the pair first and make this refresh
// fail even though the connection is fine - in that case, adopt the token it just stored instead
// of failing the caller.
func (c *Client) recoverFromUnauthorized() error {
	if refreshErr := c.Refresh(); refreshErr != nil {
		if c.integration != nil {
			if current, findErr := findSecret(c.integration, SecretOAuthAccessToken); findErr == nil && current != "" && current != c.AccessToken {
				c.AccessToken = current
				return nil
			}
		}
		return unauthorizedRefreshError(refreshErr)
	}
	return nil
}

func unauthorizedRefreshError(refreshErr error) error {
	wrapped := fmt.Errorf("request got 401 and token refresh failed: %w", refreshErr)
	if IsRetryableAPIError(refreshErr) {
		return wrapped
	}
	return errors.Join(&APIError{StatusCode: http.StatusUnauthorized}, wrapped)
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

// Refresh exchanges the stored refresh token for a new access/refresh token pair and persists them,
// along with when the new access token expires so Sync can proactively refresh ahead of that point.
func (c *Client) Refresh() error {
	if c.integration == nil {
		return fmt.Errorf("no integration context available to refresh the OAuth token")
	}

	app := resolveOAuthApp(c.integration)
	if app.ClientID == "" || app.ClientSecret == "" {
		return fmt.Errorf("missing Jira OAuth app credentials")
	}
	refreshToken, err := findSecret(c.integration, SecretOAuthRefreshToken)
	if err != nil {
		return fmt.Errorf("error reading OAuth refresh token: %w", err)
	}
	if refreshToken == "" {
		return fmt.Errorf("missing Jira OAuth refresh token; connect Jira via OAuth first")
	}

	token, err := NewAuth(c.http).RefreshToken(app.ClientID, app.ClientSecret, refreshToken)
	if err != nil {
		return err
	}

	// Atlassian's refresh tokens are single-use: this call already invalidated the one just sent,
	// whether or not the writes below succeed. A missing replacement leaves no way to ever
	// refresh again, so treat it as a hard failure rather than silently keeping the (now dead)
	// old one. Persist it before the access token: if a crash or write failure happens between
	// the two, losing the access token just means the next near-expiry attempt retries normally,
	// but losing the new refresh token means every future refresh fails until the user reconnects.
	if token.RefreshToken == "" {
		return fmt.Errorf("token refresh response did not include a new refresh token")
	}
	if err := c.integration.SetSecret(SecretOAuthRefreshToken, []byte(token.RefreshToken)); err != nil {
		return fmt.Errorf("error storing refreshed refresh token: %w", err)
	}
	if err := c.integration.SetSecret(SecretOAuthAccessToken, []byte(token.AccessToken)); err != nil {
		return fmt.Errorf("error storing refreshed access token: %w", err)
	}

	metadata := readMetadata(c.integration)
	metadata.AccessTokenExpiresAt = ""
	if expiresAt := token.ExpiresAt(); !expiresAt.IsZero() {
		metadata.AccessTokenExpiresAt = expiresAt.Format(time.RFC3339)
	}
	c.integration.SetMetadata(metadata)

	c.AccessToken = token.AccessToken
	return nil
}

type Auth struct {
	client core.HTTPContext
}

func NewAuth(client core.HTTPContext) *Auth {
	return &Auth{client: client}
}

// TokenResponse is Atlassian's OAuth token endpoint response. Refresh tokens
// rotate on every use, so both fields are replaced together on refresh.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// GetExpiration returns how long to wait before resyncing to refresh the
// token: half its lifetime, so one failed resync still leaves time to retry.
func (t *TokenResponse) GetExpiration() time.Duration {
	if t.ExpiresIn > 0 {
		seconds := max(t.ExpiresIn/2, 1)
		return time.Duration(seconds) * time.Second
	}
	return 30 * time.Minute
}

// ExpiresAt returns when the access token stops working, or the zero time when
// Atlassian did not report a lifetime.
func (t *TokenResponse) ExpiresAt() time.Time {
	if t.ExpiresIn <= 0 {
		return time.Time{}
	}
	return time.Now().Add(time.Duration(t.ExpiresIn) * time.Second)
}

func (a *Auth) ExchangeCode(clientID, clientSecret, code, redirectURI string) (*TokenResponse, error) {
	return a.requestToken(map[string]string{
		"grant_type":    "authorization_code",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"code":          code,
		"redirect_uri":  redirectURI,
	})
}

func (a *Auth) RefreshToken(clientID, clientSecret, refreshToken string) (*TokenResponse, error) {
	return a.requestToken(map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     clientID,
		"client_secret": clientSecret,
		"refresh_token": refreshToken,
	})
}

func (a *Auth) requestToken(body map[string]string) (*TokenResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error marshaling token request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, TokenURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("error building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error exchanging token: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading token response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, &APIError{StatusCode: res.StatusCode, Body: string(responseBody)}
	}

	var token TokenResponse
	if err := json.Unmarshal(responseBody, &token); err != nil {
		return nil, fmt.Errorf("error parsing token response: %w", err)
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}

	return &token, nil
}

// HandleCallback validates the OAuth callback request and exchanges its code for tokens.
func (a *Auth) HandleCallback(req *http.Request, clientID, clientSecret, expectedState, redirectURI string) (*TokenResponse, error) {
	query := req.URL.Query()
	code := query.Get("code")
	state := query.Get("state")

	if errParam := query.Get("error"); errParam != "" {
		return nil, fmt.Errorf("OAuth error: %s - %s", errParam, query.Get("error_description"))
	}
	if code == "" || state == "" {
		return nil, fmt.Errorf("missing code or state")
	}
	if expectedState == "" || state != expectedState {
		return nil, fmt.Errorf("invalid state")
	}

	return a.ExchangeCode(clientID, clientSecret, code, redirectURI)
}

// AccessibleResource is one Jira Cloud site the OAuth grant has access to.
type AccessibleResource struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Scopes []string `json:"scopes"`
}

func (a *Auth) AccessibleResources(accessToken string) ([]AccessibleResource, error) {
	req, err := http.NewRequest(http.MethodGet, AccessibleResourcesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error building accessible-resources request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	res, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching accessible resources: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading accessible resources response: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("accessible resources request got %d: %s", res.StatusCode, string(body))
	}

	var resources []AccessibleResource
	if err := json.Unmarshal(body, &resources); err != nil {
		return nil, fmt.Errorf("error parsing accessible resources response: %w", err)
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("no accessible Jira sites were granted to this OAuth app")
	}

	return resources, nil
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

// UnmarshalJSON reads both the flat statusCategory string from
// /rest/api/3/statuses/search and the nested statusCategory.key object
// returned by issue and transition payloads.
func (s *Status) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		StatusCategory any    `json:"statusCategory"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.ID = raw.ID
	s.Name = raw.Name
	s.Category = statusCategoryFromValue(raw.StatusCategory)
	return nil
}

func statusCategoryFromValue(value any) string {
	switch category := value.(type) {
	case string:
		if normalized := normalizeStatusCategoryName(category); normalized != "UNDEFINED" {
			return normalized
		}
		return normalizeStatusCategoryKey(category)
	case map[string]any:
		if key, _ := category["key"].(string); strings.TrimSpace(key) != "" {
			return normalizeStatusCategoryKey(key)
		}
		if name, _ := category["name"].(string); strings.TrimSpace(name) != "" {
			return normalizeStatusCategoryName(name)
		}
	}
	return "UNDEFINED"
}

func isDoneCategory(category string) bool {
	return normalizeStatusCategoryName(category) == "DONE" ||
		normalizeStatusCategoryKey(category) == "DONE"
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

// IssueWebhookRegistration is one entry of the "webhooks" array in a dynamic webhook registration
// request. JQLFilter has no omitempty: Atlassian's schema requires the key to be present even when
// its value is an empty string (which matches every issue in every project), so dropping the key
// entirely for an unfiltered registration makes the whole request fail.
type IssueWebhookRegistration struct {
	JQLFilter string   `json:"jqlFilter"`
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

// IssueWebhook is one dynamic webhook registered by this OAuth app.
type IssueWebhook struct {
	ID             int64    `json:"id"`
	URL            string   `json:"url"`
	JQLFilter      string   `json:"jqlFilter"`
	Events         []string `json:"events"`
	ExpirationDate string   `json:"expirationDate"`
}

type issueWebhooksPage struct {
	Values     []IssueWebhook `json:"values"`
	IsLast     bool           `json:"isLast"`
	StartAt    int            `json:"startAt"`
	MaxResults int            `json:"maxResults"`
	Total      int            `json:"total"`
}

const issueWebhookListPageSize = 100
const issueWebhookListPageLimit = 20

// ListIssueWebhooks returns every dynamic webhook registered by this OAuth app.
func (c *Client) ListIssueWebhooks() ([]IssueWebhook, error) {
	var webhooks []IssueWebhook
	startAt := 0

	for range issueWebhookListPageLimit {
		pageURL, err := url.Parse(c.apiURL("/rest/api/3/webhook"))
		if err != nil {
			return nil, fmt.Errorf("parse webhook list URL: %w", err)
		}
		query := pageURL.Query()
		query.Set("startAt", strconv.Itoa(startAt))
		query.Set("maxResults", strconv.Itoa(issueWebhookListPageSize))
		pageURL.RawQuery = query.Encode()

		responseBody, err := c.execRequest(http.MethodGet, pageURL.String(), nil)
		if err != nil {
			return nil, err
		}

		var page issueWebhooksPage
		if err := json.Unmarshal(responseBody, &page); err != nil {
			return nil, fmt.Errorf("parse webhook list response: %w", err)
		}

		webhooks = append(webhooks, page.Values...)
		if page.IsLast || len(page.Values) == 0 {
			return webhooks, nil
		}
		startAt += len(page.Values)
	}

	return nil, fmt.Errorf("webhook list exceeded page limit")
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

// RefreshIssueWebhooks extends the life of previously-registered dynamic webhooks by another 30
// days from now - Atlassian expires them 30 days after creation or after their last refresh.
func (c *Client) RefreshIssueWebhooks(webhookIDs []int64) error {
	if len(webhookIDs) == 0 {
		return nil
	}

	body, err := json.Marshal(map[string][]int64{"webhookIds": webhookIDs})
	if err != nil {
		return fmt.Errorf("marshal refresh webhook request: %w", err)
	}

	_, err = c.execRequest(http.MethodPut, c.apiURL("/rest/api/3/webhook/refresh"), bytes.NewReader(body))
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

// IssueSearchHit is one element from POST /rest/api/3/search/jql.
type IssueSearchHit struct {
	ID     string         `json:"id"`
	Key    string         `json:"key"`
	Fields map[string]any `json:"fields"`
}

type issueSearchAPIResponse struct {
	MaxResults    int              `json:"maxResults"`
	IsLast        bool             `json:"isLast"`
	NextPageToken string           `json:"nextPageToken"`
	Issues        []IssueSearchHit `json:"issues"`
}

type jiraSearchPOSTBody struct {
	JQL           string   `json:"jql"`
	MaxResults    int      `json:"maxResults"`
	Fields        []string `json:"fields"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
}

func (c *Client) searchIssuesPage(jql, nextPageToken string, maxResults int) (issueSearchAPIResponse, error) {
	var empty issueSearchAPIResponse
	if maxResults <= 0 {
		maxResults = 50
	}
	if maxResults > 100 {
		maxResults = 100
	}

	body := jiraSearchPOSTBody{
		JQL:           jql,
		MaxResults:    maxResults,
		Fields:        []string{"summary", "created"},
		NextPageToken: nextPageToken,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return empty, fmt.Errorf("marshal search body: %w", err)
	}

	u := c.apiURL("/rest/api/3/search/jql")
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
	resp, err := c.searchIssuesPage(jql, "", maxResults)
	if err != nil {
		return nil, err
	}
	return resp.Issues, nil
}

// SearchIssuesUpTo pages through POST /rest/api/3/search/jql until maxIssues are
// collected or Jira reports no further results. Jira caps each request at 100 issues.
func (c *Client) SearchIssuesUpTo(jql string, maxIssues int) ([]IssueSearchHit, error) {
	if maxIssues <= 0 {
		maxIssues = 500
	}
	const pageCap = 100

	var out []IssueSearchHit
	nextPageToken := ""
	for len(out) < maxIssues {
		pageMax := pageCap
		if remain := maxIssues - len(out); remain < pageMax {
			pageMax = remain
		}
		if pageMax <= 0 {
			break
		}

		resp, err := c.searchIssuesPage(jql, nextPageToken, pageMax)
		if err != nil {
			return nil, err
		}

		out = append(out, resp.Issues...)
		if len(resp.Issues) == 0 || resp.IsLast || resp.NextPageToken == "" || resp.NextPageToken == nextPageToken {
			break
		}
		nextPageToken = resp.NextPageToken
	}

	return out, nil
}

func jqlQuotedProjectKey(projectKey string) string {
	escaped := strings.ReplaceAll(projectKey, `\`, `\\`)
	return strings.ReplaceAll(escaped, `"`, `\"`)
}
