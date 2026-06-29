package supergit

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/git/provider"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

type RepoRequest struct {
	ID            string `json:"id"`
	DefaultBranch string `json:"default_branch"`
}

type RepoResponse struct {
	ID            string `json:"id"`
	DefaultBranch string `json:"default_branch"`
}

type ListFilesResponse struct {
	Paths []string `json:"paths"`
	Ref   string   `json:"ref"`
}

type CommitResponse struct {
	Commit struct {
		CommitSHA string `json:"commit_sha"`
	} `json:"commit"`
	Result struct {
		NewSHA string `json:"new_sha"`
	} `json:"result"`
}

type GetCommitResponse struct {
	CommitSHA string `json:"commit_sha"`
}

type ListBranchesResponse struct {
	Branches []string `json:"branches"`
}

type CreateBranchRequest struct {
	Branch  string `json:"branch"`
	FromRef string `json:"from_ref"`
}

type MergeBranchRequest struct {
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	Message      string `json:"message"`
	Author       struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"author"`
}

type MergeBranchResponse struct {
	CommitSHA string `json:"commit_sha"`
}

func (c *Client) createRepo(ctx context.Context, req RepoRequest) (*RepoResponse, error) {
	var resp RepoResponse
	if err := c.doJSON(ctx, http.MethodPost, "/repos", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) deleteRepo(ctx context.Context, repoID string) error {
	return c.doJSON(ctx, http.MethodDelete, repoPath(repoID), nil, nil)
}

func (c *Client) listFiles(ctx context.Context, repoID, ref string) ([]string, error) {
	query := url.Values{}
	if ref != "" {
		query.Set("ref", ref)
	}

	var resp ListFilesResponse
	if err := c.doJSON(ctx, http.MethodGet, repoPath(repoID)+"/files?"+query.Encode(), nil, &resp); err != nil {
		return nil, err
	}

	return resp.Paths, nil
}

func (c *Client) getFile(ctx context.Context, repoID, filePath, ref string) (io.ReadCloser, error) {
	query := url.Values{}
	query.Set("path", filePath)
	if ref != "" {
		query.Set("ref", ref)
	}

	req, err := c.newRequest(ctx, http.MethodGet, repoPath(repoID)+"/files?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, fmt.Errorf("supergit request failed: %s", resp.Status)
	}

	return resp.Body, nil
}

func (c *Client) createCommit(ctx context.Context, repoID string, body io.Reader) (*CommitResponse, error) {
	req, err := c.newRequest(ctx, http.MethodPost, repoPath(repoID)+"/commits", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("supergit request failed: %s", resp.Status)
	}

	var result CommitResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) getCommit(ctx context.Context, repoID, sha string) (*GetCommitResponse, error) {
	query := url.Values{}
	query.Set("sha", sha)

	var resp GetCommitResponse
	if err := c.doJSON(ctx, http.MethodGet, repoPath(repoID)+"/commit?"+query.Encode(), nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) listBranches(ctx context.Context, repoID, prefix string) ([]string, error) {
	query := url.Values{}
	if prefix != "" {
		query.Set("prefix", prefix)
	}

	var resp ListBranchesResponse
	endpoint := repoPath(repoID) + "/branches"
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Branches, nil
}

func (c *Client) createBranch(ctx context.Context, repoID string, req CreateBranchRequest) error {
	return c.doJSON(ctx, http.MethodPost, repoPath(repoID)+"/branches", req, nil)
}

func (c *Client) mergeBranch(ctx context.Context, repoID string, req MergeBranchRequest) (string, error) {
	var resp MergeBranchResponse
	if err := c.doJSON(ctx, http.MethodPost, repoPath(repoID)+"/merge", req, &resp); err != nil {
		return "", err
	}

	return resp.CommitSHA, nil
}

func (c *Client) deleteBranch(ctx context.Context, repoID, branch string) error {
	return c.doJSON(ctx, http.MethodDelete, repoPath(repoID)+"/branches/"+url.PathEscape(branch), nil, nil)
}

func (c *Client) doJSON(ctx context.Context, method, endpoint string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}

	req, err := c.newRequest(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("supergit request failed: %s", resp.Status)
	}

	if out == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) newRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, body)
}

func repoPath(repoID string) string {
	return "/repos/" + url.PathEscape(repoID)
}

func buildCommitNDJSON(operations []provider.FileOperation, options provider.CommitOptions) (io.Reader, error) {
	type fileEntry struct {
		Path      string `json:"path"`
		Operation string `json:"operation"`
		ContentID string `json:"content_id,omitempty"`
		Mode      string `json:"mode,omitempty"`
	}

	files := make([]fileEntry, 0, len(operations))
	var lines []string

	metadata := map[string]any{
		"metadata": map[string]any{
			"target_branch":     options.Branch,
			"base_branch":       options.BaseBranch,
			"expected_head_sha": options.ExpectedHeadSHA,
			"commit_message":    options.Message,
			"author": map[string]string{
				"name":  options.Author.Name,
				"email": options.Author.Email,
			},
			"files": files,
		},
	}

	for index, operation := range operations {
		contentID := fmt.Sprintf("blob-%d", index+1)
		if operation.Delete {
			files = append(files, fileEntry{
				Path:      operation.Path,
				Operation: "delete",
				ContentID: contentID,
			})
			chunk := map[string]any{
				"blob_chunk": map[string]any{
					"content_id": contentID,
					"data":       "",
					"eof":        true,
				},
			}
			encoded, err := json.Marshal(chunk)
			if err != nil {
				return nil, err
			}
			lines = append(lines, string(encoded))
			continue
		}

		content, err := io.ReadAll(operation.Content)
		if err != nil {
			return nil, err
		}

		files = append(files, fileEntry{
			Path:      operation.Path,
			Operation: "upsert",
			ContentID: contentID,
			Mode:      "100644",
		})

		chunk := map[string]any{
			"blob_chunk": map[string]any{
				"content_id": contentID,
				"data":       base64.StdEncoding.EncodeToString(content),
				"eof":        true,
			},
		}
		encoded, err := json.Marshal(chunk)
		if err != nil {
			return nil, err
		}
		lines = append(lines, string(encoded))
	}

	metadata["metadata"].(map[string]any)["files"] = files
	firstLine, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	body := string(firstLine) + "\n" + strings.Join(lines, "\n") + "\n"
	return strings.NewReader(body), nil
}
