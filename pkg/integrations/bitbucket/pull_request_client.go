package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
	path, err := repositoryPath(repository)
	if err != nil {
		return nil, err
	}

	body := map[string]any{"content": map[string]any{"raw": content}}

	var comment map[string]any
	endpoint := fmt.Sprintf("%s/repositories/%s/pullrequests/%d/comments", baseURL, path, id)
	if err := c.doJSON(http.MethodPost, endpoint, body, http.StatusCreated, &comment); err != nil {
		return nil, err
	}
	return comment, nil
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
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, bitbucketErrorMessage(body))
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
