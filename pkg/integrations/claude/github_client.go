package claude

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const githubAPIBase = "https://api.github.com"

type GitHubClient struct {
	Token string
	http  core.HTTPContext
}

func NewGitHubClient(token string, httpCtx core.HTTPContext) *GitHubClient {
	return &GitHubClient{Token: token, http: httpCtx}
}

func (g *GitHubClient) execRequest(method, url string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+g.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res, err := g.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub API error (%d): %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// GetBranchSHA returns the commit SHA of the tip of the given branch.
func (g *GitHubClient) GetBranchSHA(owner, repo, branch string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/git/ref/heads/%s", githubAPIBase, owner, repo, branch)
	body, err := g.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	var ref struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.Unmarshal(body, &ref); err != nil {
		return "", fmt.Errorf("failed to parse ref response: %v", err)
	}

	return ref.Object.SHA, nil
}

// CreateBranch creates a new branch from the given base SHA.
func (g *GitHubClient) CreateBranch(owner, repo, branchName, baseSHA string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/git/refs", githubAPIBase, owner, repo)
	payload := map[string]string{
		"ref": "refs/heads/" + branchName,
		"sha": baseSHA,
	}
	_, err := g.execRequest(http.MethodPost, url, payload)
	return err
}

// CommitFile creates or updates a single file on the given branch.
func (g *GitHubClient) CommitFile(owner, repo, branch, path, message, content string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", githubAPIBase, owner, repo, path)
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	payload := map[string]string{
		"message": message,
		"content": encoded,
		"branch":  branch,
	}
	_, err := g.execRequest(http.MethodPut, url, payload)
	return err
}

// PRResult holds the result of creating a pull request.
type PRResult struct {
	URL    string
	Number int
}

// CreatePRComment posts a comment on a pull request.
func (g *GitHubClient) CreatePRComment(owner, repo string, prNumber int, body string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", githubAPIBase, owner, repo, prNumber)
	_, err := g.execRequest(http.MethodPost, url, map[string]string{"body": body})
	return err
}

// PRFile represents a file changed in a pull request.
type PRFile struct {
	Filename string `json:"filename"`
	Status   string `json:"status"`
	Patch    string `json:"patch"`
}

// GetPRFiles returns the list of files changed in a pull request.
func (g *GitHubClient) GetPRFiles(owner, repo string, prNumber int) ([]PRFile, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/files", githubAPIBase, owner, repo, prNumber)
	body, err := g.execRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var files []PRFile
	if err := json.Unmarshal(body, &files); err != nil {
		return nil, fmt.Errorf("failed to parse PR files: %v", err)
	}

	return files, nil
}

// SubmitPRReview posts an APPROVE or REQUEST_CHANGES review on a pull request.
func (g *GitHubClient) SubmitPRReview(owner, repo string, prNumber int, event, body string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/reviews", githubAPIBase, owner, repo, prNumber)
	_, err := g.execRequest(http.MethodPost, url, map[string]string{
		"body":  body,
		"event": event,
	})
	return err
}

// MergePR merges a pull request into its base branch.
func (g *GitHubClient) MergePR(owner, repo string, prNumber int, commitMessage string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/merge", githubAPIBase, owner, repo, prNumber)
	_, err := g.execRequest(http.MethodPut, url, map[string]string{
		"commit_message": commitMessage,
		"merge_method":   "squash",
	})
	return err
}

// CreatePR opens a pull request and returns its URL and number.
func (g *GitHubClient) CreatePR(owner, repo, title, body, head, base string) (*PRResult, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls", githubAPIBase, owner, repo)
	payload := map[string]string{
		"title": title,
		"body":  body,
		"head":  head,
		"base":  base,
	}

	responseBody, err := g.execRequest(http.MethodPost, url, payload)
	if err != nil {
		return nil, err
	}

	var pr struct {
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
	}
	if err := json.Unmarshal(responseBody, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse PR response: %v", err)
	}

	return &PRResult{URL: pr.HTMLURL, Number: pr.Number}, nil
}
