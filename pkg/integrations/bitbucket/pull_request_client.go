package bitbucket

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
)

const (
	PullRequestStateOpen     = "OPEN"
	PullRequestStateMerged   = "MERGED"
	PullRequestStateDeclined = "DECLINED"
)

type PullRequestListOptions struct {
	SourceBranch      string
	DestinationBranch string
	States            []string
}

type CreatePullRequestRequest struct {
	Title       string
	Description string
	Source      string
	Destination string
	Draft       bool
}

type UpdatePullRequestRequest struct {
	Title       string
	Description string
}

// APIError carries the Bitbucket HTTP status with the response message so
// callers can distinguish conflicts and rate limits without parsing text.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return e.Message
}

// repositoryPath turns "workspace/repo" into an escaped API path segment.
func repositoryPath(repository string) (string, error) {
	workspace, slug, ok := strings.Cut(strings.TrimSpace(repository), "/")
	if !ok || workspace == "" || slug == "" || strings.Contains(slug, "/") {
		return "", fmt.Errorf("repository must be in workspace/repository format: %q", repository)
	}
	return url.PathEscape(workspace) + "/" + url.PathEscape(strings.TrimSuffix(slug, ".git")), nil
}

// GetMainBranch returns the main branch name of a "workspace/repo" repository.
func (c *Client) GetMainBranch(repository string) (string, error) {
	path, err := repositoryPath(repository)
	if err != nil {
		return "", err
	}

	var response struct {
		MainBranch struct {
			Name string `json:"name"`
		} `json:"mainbranch"`
	}
	endpoint := fmt.Sprintf("%s/repositories/%s", baseURL, path)
	if err := c.doJSON(http.MethodGet, endpoint, nil, http.StatusOK, &response); err != nil {
		return "", err
	}
	return response.MainBranch.Name, nil
}

func (c *Client) ListPullRequests(repository string, opts PullRequestListOptions) ([]map[string]any, error) {
	path, err := repositoryPath(repository)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	filters := []string{}
	if opts.SourceBranch != "" {
		filters = append(filters, fmt.Sprintf("source.branch.name=%q", opts.SourceBranch))
	}
	if opts.DestinationBranch != "" {
		filters = append(filters, fmt.Sprintf("destination.branch.name=%q", opts.DestinationBranch))
	}
	if len(filters) > 0 {
		query.Set("q", strings.Join(filters, " AND "))
	}
	for _, state := range opts.States {
		query.Add("state", state)
	}

	var response struct {
		Values []map[string]any `json:"values"`
	}
	endpoint := fmt.Sprintf("%s/repositories/%s/pullrequests?%s", baseURL, path, query.Encode())
	if err := c.doJSON(http.MethodGet, endpoint, nil, http.StatusOK, &response); err != nil {
		return nil, err
	}
	return response.Values, nil
}

func (c *Client) CreatePullRequest(repository string, request CreatePullRequestRequest) (map[string]any, error) {
	path, err := repositoryPath(repository)
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"title":       request.Title,
		"description": request.Description,
		"source":      map[string]any{"branch": map[string]any{"name": request.Source}},
		"destination": map[string]any{"branch": map[string]any{"name": request.Destination}},
		"draft":       request.Draft,
	}

	var pullRequest map[string]any
	endpoint := fmt.Sprintf("%s/repositories/%s/pullrequests", baseURL, path)
	if err := c.doJSON(http.MethodPost, endpoint, body, http.StatusCreated, &pullRequest); err != nil {
		return nil, err
	}
	return pullRequest, nil
}

func (c *Client) UpdatePullRequest(repository string, id int64, request UpdatePullRequestRequest) (map[string]any, error) {
	path, err := repositoryPath(repository)
	if err != nil {
		return nil, err
	}

	body := map[string]any{}
	if request.Title != "" {
		body["title"] = request.Title
	}
	if request.Description != "" {
		body["description"] = request.Description
	}

	var pullRequest map[string]any
	endpoint := fmt.Sprintf("%s/repositories/%s/pullrequests/%d", baseURL, path, id)
	if err := c.doJSON(http.MethodPut, endpoint, body, http.StatusOK, &pullRequest); err != nil {
		return nil, err
	}
	return pullRequest, nil
}

func (c *Client) CreatePullRequestComment(repository string, id int64, content string) (map[string]any, error) {
	return c.CreatePullRequestCommentWithParent(repository, id, content, "")
}

// CreatePullRequestCommentWithParent posts a comment, or a threaded reply
// when parentID names an existing comment.
func (c *Client) CreatePullRequestCommentWithParent(repository string, id int64, content, parentID string) (map[string]any, error) {

	path, err := repositoryPath(repository)
	if err != nil {
		return nil, err
	}

	body := map[string]any{"content": map[string]any{"raw": content}}
	if parent, ok := parseCommentID(parentID); ok {
		body["parent"] = map[string]any{"id": parent}
	}

	var comment map[string]any
	endpoint := fmt.Sprintf("%s/repositories/%s/pullrequests/%d/comments", baseURL, path, id)
	if err := c.doJSON(http.MethodPost, endpoint, body, http.StatusCreated, &comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func parseCommentID(raw string) (int64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// PullRequestMergeStrategy maps a merge method name to the Bitbucket merge
// strategy. Unknown methods fall back to a merge commit.
func PullRequestMergeStrategy(method string) string {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "squash":
		return "squash"
	case "rebase", "fast_forward", "fast-forward":
		return "fast_forward"
	default:
		return "merge_commit"
	}
}

// BitbucketPullRequest is the merge-relevant subset of a pull request.
type BitbucketPullRequest struct {
	ID          int64  `json:"id"`
	State       string `json:"state"`
	Draft       bool   `json:"draft"`
	SourceHash  string
	SourceBr    string
	DestBranch  string
	MergeCommit string
}

// GetPullRequest reads one pull request by number.
func (c *Client) GetPullRequest(repository string, id int64) (*BitbucketPullRequest, error) {
	path, err := repositoryPath(repository)
	if err != nil {
		return nil, err
	}

	var response struct {
		ID     int64  `json:"id"`
		State  string `json:"state"`
		Draft  bool   `json:"draft"`
		Source struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
			Commit struct {
				Hash string `json:"hash"`
			} `json:"commit"`
		} `json:"source"`
		Destination struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
			Commit struct {
				Hash string `json:"hash"`
			} `json:"commit"`
		} `json:"destination"`
		MergeCommit struct {
			Hash string `json:"hash"`
		} `json:"merge_commit"`
	}
	endpoint := fmt.Sprintf("%s/repositories/%s/pullrequests/%d", baseURL, path, id)
	if err := c.doJSON(http.MethodGet, endpoint, nil, http.StatusOK, &response); err != nil {
		return nil, err
	}
	return &BitbucketPullRequest{
		ID:          response.ID,
		State:       response.State,
		Draft:       response.Draft,
		SourceHash:  response.Source.Commit.Hash,
		SourceBr:    response.Source.Branch.Name,
		DestBranch:  response.Destination.Branch.Name,
		MergeCommit: response.MergeCommit.Hash,
	}, nil
}

// MergePullRequest merges one open pull request with the given strategy.
// Bitbucket has no head precondition, so callers compare the source hash first.
func (c *Client) MergePullRequest(repository string, id int64, strategy string) (*BitbucketPullRequest, error) {
	path, err := repositoryPath(repository)
	if err != nil {
		return nil, err
	}

	body := map[string]any{"type": PullRequestMergeStrategy(strategy)}

	var response struct {
		ID    int64  `json:"id"`
		State string `json:"state"`
	}
	endpoint := fmt.Sprintf("%s/repositories/%s/pullrequests/%d/merge", baseURL, path, id)
	if err := c.doJSON(http.MethodPost, endpoint, body, http.StatusOK, &response); err != nil {
		return nil, err
	}
	return &BitbucketPullRequest{ID: response.ID, State: response.State}, nil
}

// DeclinePullRequest declines one open pull request.
func (c *Client) DeclinePullRequest(repository string, id int64) (*BitbucketPullRequest, error) {
	path, err := repositoryPath(repository)
	if err != nil {
		return nil, err
	}

	var response struct {
		ID    int64  `json:"id"`
		State string `json:"state"`
	}
	endpoint := fmt.Sprintf("%s/repositories/%s/pullrequests/%d/decline", baseURL, path, id)
	if err := c.doJSON(http.MethodPost, endpoint, nil, http.StatusOK, &response); err != nil {
		return nil, err
	}
	return &BitbucketPullRequest{ID: response.ID, State: response.State}, nil
}

// Commit build states reported by Bitbucket commit statuses.
const (
	CommitStatusSuccessful = "SUCCESSFUL"
	CommitStatusFailed     = "FAILED"
	CommitStatusInProgress = "INPROGRESS"
	CommitStatusStopped    = "STOPPED"
)

// CommitStatus is one Bitbucket build status on a commit.
type CommitStatus struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	State       string `json:"state"`
	URL         string `json:"url"`
	Description string `json:"description"`
	CreatedOn   string `json:"created_on"`
	UpdatedOn   string `json:"updated_on"`
}

// ListCommitStatuses returns every build status on a commit, following
// pagination. An empty list is not an error; callers wait it out.
func (c *Client) ListCommitStatuses(repository, sha string) ([]CommitStatus, error) {
	path, err := repositoryPath(repository)
	if err != nil {
		return nil, err
	}
	sha = strings.TrimSpace(sha)
	if sha == "" {
		return nil, fmt.Errorf("commit sha is required")
	}

	statuses := []CommitStatus{}
	endpoint := fmt.Sprintf("%s/repositories/%s/commit/%s/statuses?pagelen=100", baseURL, path, url.PathEscape(sha))
	for endpoint != "" {
		var response struct {
			Values []CommitStatus `json:"values"`
			Next   string         `json:"next"`
		}
		if err := c.doJSON(http.MethodGet, endpoint, nil, http.StatusOK, &response); err != nil {
			return nil, err
		}
		statuses = append(statuses, response.Values...)
		endpoint = response.Next
	}
	return statuses, nil
}

func (c *Client) doJSON(method, endpoint string, payload any, expectedStatus int, out any) error {
	var reader io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("error encoding request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	c.setAuthHeaders(req)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != expectedStatus {
		return &APIError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("unexpected status code %d: %s", resp.StatusCode, bitbucketErrorMessage(body))}
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}
	return nil
}

// bitbucketErrorMessage returns error.message from a Bitbucket error body,
// or the raw body when the shape is unknown.
func bitbucketErrorMessage(body []byte) string {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error.Message != "" {
		return parsed.Error.Message
	}
	return string(body)
}

// MergedBitbucketPullRequest is the velocity-relevant subset of a merged
// pull request.
type MergedBitbucketPullRequest struct {
	ID           int64
	State        string
	Title        string
	AuthorUUID   string
	AuthorNick   string
	AuthorName   string
	AuthorAvatar string
	SourceHash   string
	MergeHash    string
	UpdatedOn    time.Time
}

// ListMergedPullRequests returns merged pull requests, newest first,
// stopping once updated_on predates from. At most maxPages pages are read;
// truncated reports whether the page cap cut the walk short.
func (c *Client) ListMergedPullRequests(repository string, from time.Time, maxPages int) ([]MergedBitbucketPullRequest, bool, error) {
	path, err := repositoryPath(repository)
	if err != nil {
		return nil, false, err
	}
	if maxPages <= 0 {
		maxPages = 1
	}

	merged := []MergedBitbucketPullRequest{}
	endpoint := fmt.Sprintf("%s/repositories/%s/pullrequests?state=MERGED&sort=-updated_on&pagelen=50", baseURL, path)
	for page := 0; page < maxPages && endpoint != ""; page++ {
		var response struct {
			Values []struct {
				ID        int64  `json:"id"`
				State     string `json:"state"`
				Title     string `json:"title"`
				UpdatedOn string `json:"updated_on"`
				Author    struct {
					UUID        string `json:"uuid"`
					Nickname    string `json:"nickname"`
					DisplayName string `json:"display_name"`
					Links       struct {
						Avatar struct {
							Href string `json:"href"`
						} `json:"avatar"`
					} `json:"links"`
				} `json:"author"`
				Source struct {
					Commit struct {
						Hash string `json:"hash"`
					} `json:"commit"`
				} `json:"source"`
				MergeCommit struct {
					Hash string `json:"hash"`
				} `json:"merge_commit"`
			} `json:"values"`
			Next string `json:"next"`
		}
		if err := c.doJSON(http.MethodGet, endpoint, nil, http.StatusOK, &response); err != nil {
			return nil, false, err
		}
		for _, pr := range response.Values {
			updated, err := parseBitbucketTime(pr.UpdatedOn)
			if err != nil {
				continue
			}
			if updated.Before(from) {
				return merged, false, nil
			}
			merged = append(merged, MergedBitbucketPullRequest{
				ID:           pr.ID,
				State:        pr.State,
				Title:        pr.Title,
				AuthorUUID:   bitbucketUUID(pr.Author.UUID),
				AuthorNick:   pr.Author.Nickname,
				AuthorName:   pr.Author.DisplayName,
				AuthorAvatar: pr.Author.Links.Avatar.Href,
				SourceHash:   pr.Source.Commit.Hash,
				MergeHash:    pr.MergeCommit.Hash,
				UpdatedOn:    updated,
			})
		}
		endpoint = response.Next
	}
	return merged, endpoint != "", nil
}

// MergeCommit carries the merge timestamp and message of one commit.
type MergeCommit struct {
	Hash    string
	Date    time.Time
	Message string
}

// GetMergeCommit reads one commit's date and message for merge attribution.
func (c *Client) GetMergeCommit(repository, sha string) (*MergeCommit, error) {
	path, err := repositoryPath(repository)
	if err != nil {
		return nil, err
	}
	sha = strings.TrimSpace(sha)
	if sha == "" {
		return nil, fmt.Errorf("commit sha is required")
	}

	var response struct {
		Hash    string `json:"hash"`
		Date    string `json:"date"`
		Message string `json:"message"`
	}
	endpoint := fmt.Sprintf("%s/repositories/%s/commit/%s", baseURL, path, url.PathEscape(sha))
	if err := c.doJSON(http.MethodGet, endpoint, nil, http.StatusOK, &response); err != nil {
		return nil, err
	}
	date, err := parseBitbucketTime(response.Date)
	if err != nil {
		return nil, err
	}
	return &MergeCommit{Hash: response.Hash, Date: date, Message: response.Message}, nil
}

// GetCommitDate reads one commit's author date.
func (c *Client) GetCommitDate(repository, sha string) (time.Time, error) {
	path, err := repositoryPath(repository)
	if err != nil {
		return time.Time{}, err
	}
	sha = strings.TrimSpace(sha)
	if sha == "" {
		return time.Time{}, fmt.Errorf("commit sha is required")
	}

	var response struct {
		Date string `json:"date"`
	}
	endpoint := fmt.Sprintf("%s/repositories/%s/commit/%s", baseURL, path, url.PathEscape(sha))
	if err := c.doJSON(http.MethodGet, endpoint, nil, http.StatusOK, &response); err != nil {
		return time.Time{}, err
	}
	parsed, err := parseBitbucketTime(response.Date)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

func parseBitbucketTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("timestamp is required")
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed, nil
	}
	return time.Parse(time.RFC3339, value)
}

// bitbucketUUID normalizes a "{uuid}" account identifier for identity
// matching. Nicknames are never identity.
func bitbucketUUID(value string) string {
	normalized, err := NormalizeAccountID(value)
	if err != nil {
		return ""
	}
	return normalized
}
