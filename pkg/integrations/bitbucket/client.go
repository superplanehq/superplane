package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/superplanehq/superplane/pkg/core"
)

const baseURL = "https://api.bitbucket.org/2.0"

type Client struct {
	AuthType string
	Email    string
	Token    string
	HTTP     core.HTTPContext
}

type RepositoryResponse struct {
	Values []Repository `json:"values"`
	Next   string       `json:"next"`
}

type Repository struct {
	UUID     string         `json:"uuid" mapstructure:"uuid"`
	Name     string         `json:"name" mapstructure:"name"`
	FullName string         `json:"full_name" mapstructure:"full_name"`
	Slug     string         `json:"slug" mapstructure:"slug"`
	Links    RepositoryLink `json:"links" mapstructure:"links"`
}

type RepositoryLink struct {
	HTML struct {
		Href string `json:"href" mapstructure:"href"`
	} `json:"html" mapstructure:"html"`
}

func NewClient(authType string, httpContext core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	switch authType {
	case AuthTypeAPIToken:
		token, err := integration.GetConfig("token")
		if err != nil {
			return nil, fmt.Errorf("error getting token config: %w", err)
		}

		email, err := integration.GetConfig("email")
		if err != nil {
			return nil, fmt.Errorf("error getting email config: %w", err)
		}

		return &Client{
			AuthType: AuthTypeAPIToken,
			Email:    string(email),
			Token:    string(token),
			HTTP:     httpContext,
		}, nil

	case AuthTypeWorkspaceAccessToken:
		token, err := integration.GetConfig("token")
		if err != nil {
			return nil, fmt.Errorf("error getting token config: %w", err)
		}
		return &Client{
			AuthType: AuthTypeWorkspaceAccessToken,
			Token:    string(token),
			HTTP:     httpContext,
		}, nil
	}

	return nil, fmt.Errorf("unknown auth type %s", authType)
}

func (c *Client) setAuthHeaders(req *http.Request) {
	if c.AuthType == AuthTypeAPIToken {
		req.SetBasicAuth(c.Email, c.Token)
	} else {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

type Workspace struct {
	UUID string `json:"uuid" mapstructure:"uuid"`
	Name string `json:"name" mapstructure:"name"`
	Slug string `json:"slug" mapstructure:"slug"`
}

func (c *Client) GetWorkspace(workspaceSlug string) (*Workspace, error) {
	url := fmt.Sprintf("%s/workspaces/%s", baseURL, workspaceSlug)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	c.setAuthHeaders(req)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var workspace Workspace
	err = json.Unmarshal(body, &workspace)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &workspace, nil
}

func (c *Client) ListRepositories(workspace string) ([]Repository, error) {
	url := fmt.Sprintf("%s/repositories/%s?pagelen=100", baseURL, workspace)
	repositories := []Repository{}

	for url != "" {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}

		c.setAuthHeaders(req)
		req.Header.Set("Accept", "application/json")

		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error executing request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("error reading response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
		}

		var repoResponse RepositoryResponse
		err = json.Unmarshal(body, &repoResponse)
		if err != nil {
			return nil, fmt.Errorf("error decoding response: %w", err)
		}

		repositories = append(repositories, repoResponse.Values...)
		url = repoResponse.Next
	}

	return repositories, nil
}

type BitbucketHookRequest struct {
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Active      bool     `json:"active"`
	Secret      string   `json:"secret,omitempty"`
	Events      []string `json:"events"`
}

type BitbucketHookResponse struct {
	UUID   string `json:"uuid"`
	URL    string `json:"url"`
	Active bool   `json:"active"`
}

func (c *Client) CreateWebhook(workspace, repoSlug, webhookURL, secret string, events []string) (*BitbucketHookResponse, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s/hooks", baseURL, workspace, repoSlug)

	hookReq := BitbucketHookRequest{
		Description: "SuperPlane",
		URL:         webhookURL,
		Active:      true,
		Secret:      secret,
		Events:      events,
	}

	body, err := json.Marshal(hookReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling webhook request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	c.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}

	var hookResp BitbucketHookResponse
	err = json.Unmarshal(respBody, &hookResp)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &hookResp, nil
}

func (c *Client) DeleteWebhook(workspace, repoSlug, webhookUID string) error {
	url := fmt.Sprintf("%s/repositories/%s/%s/hooks/%s", baseURL, workspace, repoSlug, webhookUID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	c.setAuthHeaders(req)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("error executing request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

//
// Pull requests, comments and build statuses.
//

type Link struct {
	Href string `json:"href,omitempty" mapstructure:"href"`
}

type ResourceLinks struct {
	Self Link `json:"self" mapstructure:"self"`
	HTML Link `json:"html" mapstructure:"html"`
}

type Account struct {
	UUID        string `json:"uuid,omitempty" mapstructure:"uuid"`
	AccountID   string `json:"account_id,omitempty" mapstructure:"account_id"`
	DisplayName string `json:"display_name,omitempty" mapstructure:"display_name"`
	Nickname    string `json:"nickname,omitempty" mapstructure:"nickname"`
}

// AccountRef identifies a user in write requests. Bitbucket accepts the UUID
// (wrapped in braces) or the account_id, but not the display name.
type AccountRef struct {
	UUID string `json:"uuid"`
}

type Branch struct {
	Name string `json:"name" mapstructure:"name"`
}

type Commit struct {
	Hash  string        `json:"hash,omitempty" mapstructure:"hash"`
	Links ResourceLinks `json:"links,omitempty" mapstructure:"links"`
}

type PullRequestEndpoint struct {
	Branch     Branch      `json:"branch" mapstructure:"branch"`
	Commit     *Commit     `json:"commit,omitempty" mapstructure:"commit"`
	Repository *Repository `json:"repository,omitempty" mapstructure:"repository"`
}

type PullRequest struct {
	ID                int                 `json:"id" mapstructure:"id"`
	Title             string              `json:"title" mapstructure:"title"`
	Description       string              `json:"description" mapstructure:"description"`
	State             string              `json:"state" mapstructure:"state"`
	Draft             bool                `json:"draft" mapstructure:"draft"`
	CreatedOn         string              `json:"created_on" mapstructure:"created_on"`
	UpdatedOn         string              `json:"updated_on" mapstructure:"updated_on"`
	CloseSourceBranch bool                `json:"close_source_branch" mapstructure:"close_source_branch"`
	CommentCount      int                 `json:"comment_count" mapstructure:"comment_count"`
	TaskCount         int                 `json:"task_count" mapstructure:"task_count"`
	Reason            string              `json:"reason,omitempty" mapstructure:"reason"`
	Author            *Account            `json:"author,omitempty" mapstructure:"author"`
	ClosedBy          *Account            `json:"closed_by,omitempty" mapstructure:"closed_by"`
	Reviewers         []Account           `json:"reviewers,omitempty" mapstructure:"reviewers"`
	Participants      []Participant       `json:"participants,omitempty" mapstructure:"participants"`
	Source            PullRequestEndpoint `json:"source" mapstructure:"source"`
	Destination       PullRequestEndpoint `json:"destination" mapstructure:"destination"`
	MergeCommit       *Commit             `json:"merge_commit,omitempty" mapstructure:"merge_commit"`
	Links             ResourceLinks       `json:"links" mapstructure:"links"`
}

type Participant struct {
	User           *Account `json:"user,omitempty" mapstructure:"user"`
	Role           string   `json:"role,omitempty" mapstructure:"role"`
	Approved       bool     `json:"approved" mapstructure:"approved"`
	State          string   `json:"state,omitempty" mapstructure:"state"`
	ParticipatedOn string   `json:"participated_on,omitempty" mapstructure:"participated_on"`
}

type CommentContent struct {
	Raw    string `json:"raw" mapstructure:"raw"`
	Markup string `json:"markup,omitempty" mapstructure:"markup"`
	HTML   string `json:"html,omitempty" mapstructure:"html"`
}

type PullRequestComment struct {
	ID        int            `json:"id" mapstructure:"id"`
	CreatedOn string         `json:"created_on" mapstructure:"created_on"`
	UpdatedOn string         `json:"updated_on" mapstructure:"updated_on"`
	Deleted   bool           `json:"deleted" mapstructure:"deleted"`
	Content   CommentContent `json:"content" mapstructure:"content"`
	User      *Account       `json:"user,omitempty" mapstructure:"user"`
	Links     ResourceLinks  `json:"links" mapstructure:"links"`
}

type CommitStatus struct {
	Key         string        `json:"key" mapstructure:"key"`
	Name        string        `json:"name,omitempty" mapstructure:"name"`
	State       string        `json:"state" mapstructure:"state"`
	URL         string        `json:"url,omitempty" mapstructure:"url"`
	Description string        `json:"description,omitempty" mapstructure:"description"`
	Type        string        `json:"type,omitempty" mapstructure:"type"`
	Refname     string        `json:"refname,omitempty" mapstructure:"refname"`
	CreatedOn   string        `json:"created_on,omitempty" mapstructure:"created_on"`
	UpdatedOn   string        `json:"updated_on,omitempty" mapstructure:"updated_on"`
	Commit      *Commit       `json:"commit,omitempty" mapstructure:"commit"`
	Links       ResourceLinks `json:"links,omitempty" mapstructure:"links"`
}

type CreatePullRequestRequest struct {
	Title             string               `json:"title"`
	Description       string               `json:"description,omitempty"`
	Source            PullRequestEndpoint  `json:"source"`
	Destination       *PullRequestEndpoint `json:"destination,omitempty"`
	CloseSourceBranch bool                 `json:"close_source_branch"`
	Reviewers         []AccountRef         `json:"reviewers,omitempty"`
}

// UpdatePullRequestRequest only serializes the fields the user actually filled in.
// Bitbucket treats a present key as an overwrite, so sending a zero value would
// silently wipe the title, description or reviewer list.
type UpdatePullRequestRequest struct {
	Title             *string              `json:"title,omitempty"`
	Description       *string              `json:"description,omitempty"`
	Destination       *PullRequestEndpoint `json:"destination,omitempty"`
	CloseSourceBranch *bool                `json:"close_source_branch,omitempty"`
	Reviewers         *[]AccountRef        `json:"reviewers,omitempty"`
}

type MergePullRequestRequest struct {
	Type              string `json:"type"`
	Message           string `json:"message,omitempty"`
	MergeStrategy     string `json:"merge_strategy,omitempty"`
	CloseSourceBranch *bool  `json:"close_source_branch,omitempty"`
}

type commitStatusesResponse struct {
	Values []CommitStatus `json:"values"`
	Next   string         `json:"next"`
}

// doJSON runs an authenticated request against the Bitbucket API. A nil body skips
// the request payload; a nil out skips response decoding, for endpoints that answer
// with no content.
func (c *Client) doJSON(method, requestURL string, body any, out any, expectedStatuses ...int) error {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("error marshaling request body: %w", err)
		}

		payload = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, requestURL, payload)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	c.setAuthHeaders(req)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("error executing request: %w", err)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if !slices.Contains(expectedStatuses, resp.StatusCode) {
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}

	if out == nil {
		return nil
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	return nil
}

func repositoryURL(workspace, repoSlug string) string {
	return fmt.Sprintf("%s/repositories/%s/%s", baseURL, url.PathEscape(workspace), url.PathEscape(repoSlug))
}

func pullRequestURL(workspace, repoSlug string, pullRequestID int) string {
	return fmt.Sprintf("%s/pullrequests/%d", repositoryURL(workspace, repoSlug), pullRequestID)
}

func (c *Client) CreatePullRequest(workspace, repoSlug string, request *CreatePullRequestRequest) (*PullRequest, error) {
	var pullRequest PullRequest
	err := c.doJSON(
		http.MethodPost,
		fmt.Sprintf("%s/pullrequests", repositoryURL(workspace, repoSlug)),
		request,
		&pullRequest,
		http.StatusOK, http.StatusCreated,
	)

	if err != nil {
		return nil, err
	}

	return &pullRequest, nil
}

func (c *Client) GetPullRequest(workspace, repoSlug string, pullRequestID int) (*PullRequest, error) {
	var pullRequest PullRequest
	err := c.doJSON(
		http.MethodGet,
		pullRequestURL(workspace, repoSlug, pullRequestID),
		nil,
		&pullRequest,
		http.StatusOK,
	)

	if err != nil {
		return nil, err
	}

	return &pullRequest, nil
}

func (c *Client) UpdatePullRequest(workspace, repoSlug string, pullRequestID int, request *UpdatePullRequestRequest) (*PullRequest, error) {
	var pullRequest PullRequest
	err := c.doJSON(
		http.MethodPut,
		pullRequestURL(workspace, repoSlug, pullRequestID),
		request,
		&pullRequest,
		http.StatusOK,
	)

	if err != nil {
		return nil, err
	}

	return &pullRequest, nil
}

func (c *Client) MergePullRequest(workspace, repoSlug string, pullRequestID int, request *MergePullRequestRequest) (*PullRequest, error) {
	var pullRequest PullRequest
	err := c.doJSON(
		http.MethodPost,
		fmt.Sprintf("%s/merge", pullRequestURL(workspace, repoSlug, pullRequestID)),
		request,
		&pullRequest,
		http.StatusOK, http.StatusCreated, http.StatusAccepted,
	)

	if err != nil {
		return nil, err
	}

	// On a slow merge Bitbucket answers 202 with a polling task rather than the pull
	// request, which decodes into an empty struct. Report that instead of emitting a
	// pull request payload that says nothing was merged.
	if pullRequest.ID == 0 {
		return nil, fmt.Errorf("bitbucket queued the merge asynchronously and did not return the merged pull request")
	}

	return &pullRequest, nil
}

func (c *Client) CreatePullRequestComment(workspace, repoSlug string, pullRequestID int, raw string) (*PullRequestComment, error) {
	var comment PullRequestComment
	err := c.doJSON(
		http.MethodPost,
		fmt.Sprintf("%s/comments", pullRequestURL(workspace, repoSlug, pullRequestID)),
		map[string]any{"content": map[string]any{"raw": raw}},
		&comment,
		http.StatusOK, http.StatusCreated,
	)

	if err != nil {
		return nil, err
	}

	return &comment, nil
}

func (c *Client) PublishCommitStatus(workspace, repoSlug, commit string, status *CommitStatus) (*CommitStatus, error) {
	var published CommitStatus
	err := c.doJSON(
		http.MethodPost,
		fmt.Sprintf("%s/commit/%s/statuses/build", repositoryURL(workspace, repoSlug), url.PathEscape(commit)),
		status,
		&published,
		http.StatusOK, http.StatusCreated,
	)

	if err != nil {
		return nil, err
	}

	return &published, nil
}

func (c *Client) ListCommitStatuses(workspace, repoSlug, commit string) ([]CommitStatus, error) {
	next := fmt.Sprintf("%s/commit/%s/statuses?pagelen=100", repositoryURL(workspace, repoSlug), url.PathEscape(commit))
	statuses := []CommitStatus{}

	for next != "" {
		var page commitStatusesResponse
		if err := c.doJSON(http.MethodGet, next, nil, &page, http.StatusOK); err != nil {
			return nil, err
		}

		statuses = append(statuses, page.Values...)
		next = page.Next
	}

	return statuses, nil
}

// ListWorkspaceMembers backs the reviewer picker so users select real accounts
// instead of typing Bitbucket UUIDs by hand.
func (c *Client) ListWorkspaceMembers(workspace string) ([]Account, error) {
	type membership struct {
		User Account `json:"user"`
	}

	type membershipsResponse struct {
		Values []membership `json:"values"`
		Next   string       `json:"next"`
	}

	next := fmt.Sprintf("%s/workspaces/%s/members?pagelen=100", baseURL, url.PathEscape(workspace))
	members := []Account{}

	for next != "" {
		var page membershipsResponse
		if err := c.doJSON(http.MethodGet, next, nil, &page, http.StatusOK); err != nil {
			return nil, err
		}

		for _, value := range page.Values {
			members = append(members, value.User)
		}

		next = page.Next
	}

	return members, nil
}

// parsePullRequestID accepts the numeric ID Bitbucket uses for pull requests. The
// value usually arrives from an expression, so it can be a string or a number.
func parsePullRequestID(value string) (int, error) {
	id, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("pull request ID %q is not a number", value)
	}

	if id <= 0 {
		return 0, fmt.Errorf("pull request ID must be greater than zero")
	}

	return id, nil
}
