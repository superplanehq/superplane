package git

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

	"github.com/superplanehq/superplane/pkg/config"
)

type SupergitProvider struct {
	client        *supergitClient
	defaultBranch string
	limits        Limits
}

func NewSupergitProvider(cfg config.CanvasStorageConfig) (*SupergitProvider, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.SupergitBaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("CANVAS_STORAGE_SUPERGIT_BASE_URL is required")
	}

	return &SupergitProvider{
		client:        newSupergitClient(baseURL),
		defaultBranch: defaultBranch(cfg.DefaultBranch),
		limits: Limits{
			MaxFileBytes:   cfg.MaxFileBytes,
			MaxCommitBytes: cfg.MaxCommitBytes,
		},
	}, nil
}

func (p *SupergitProvider) CreateRepository(ctx context.Context, spec RepositorySpec) (*Repository, error) {
	repoID := strings.TrimSpace(spec.RepoID)
	if repoID == "" {
		repoID = CanvasRepoID(spec.OrganizationID, spec.CanvasID)
	}

	branch := defaultBranch(spec.DefaultBranch)
	if branch == "main" {
		branch = p.defaultBranch
	}

	repo, err := p.client.createRepo(ctx, supergitRepoRequest{
		ID:            repoID,
		DefaultBranch: branch,
	})
	if err != nil {
		return nil, err
	}

	repoBranch := defaultBranch(repo.DefaultBranch)
	if _, err := p.initializeRepository(ctx, repoID, repoBranch); err != nil {
		return nil, err
	}

	return &Repository{
		RepoID:        repoID,
		DefaultBranch: repoBranch,
	}, nil
}

func (p *SupergitProvider) initializeRepository(ctx context.Context, repoID, branch string) (*CommitResult, error) {
	return p.Commit(ctx, RepositoryRef{RepoID: repoID, DefaultBranch: branch}, CommitOptions{
		Message: initialRepositoryCommitMessage,
		Author: CommitAuthor{
			Name:  initialRepositoryAuthorName,
			Email: initialRepositoryAuthorEmail,
		},
		Operations: []FileOperation{
			{
				Path:      initialRepositoryFilePath,
				Content:   strings.NewReader(""),
				SizeBytes: 0,
			},
		},
	})
}

func (p *SupergitProvider) DeleteRepository(ctx context.Context, ref RepositoryRef) error {
	repoID, err := ValidateRepositoryID(ref.RepoID)
	if err != nil {
		return err
	}

	return p.client.deleteRepo(ctx, repoID)
}

func (p *SupergitProvider) ListFiles(ctx context.Context, ref RepositoryRef, options ListFilesOptions) (*ListFilesResult, error) {
	repoID, err := ValidateRepositoryID(ref.RepoID)
	if err != nil {
		return nil, err
	}

	return p.client.listFiles(ctx, repoID, refOrDefault(options.Ref, ref.DefaultBranch))
}

func (p *SupergitProvider) GetFile(ctx context.Context, ref RepositoryRef, options GetFileOptions) (io.ReadCloser, error) {
	filePath, err := NormalizePath(options.Path)
	if err != nil {
		return nil, err
	}

	repoID, err := ValidateRepositoryID(ref.RepoID)
	if err != nil {
		return nil, err
	}

	return p.client.getFile(ctx, repoID, filePath, refOrDefault(options.Ref, ref.DefaultBranch))
}

func (p *SupergitProvider) Commit(ctx context.Context, ref RepositoryRef, options CommitOptions) (*CommitResult, error) {
	if err := validateCommitMetadata(options.Message, options.Author); err != nil {
		return nil, err
	}

	operations, err := validateCommitOperations(options.Operations, p.limits)
	if err != nil {
		return nil, err
	}

	repoID, err := ValidateRepositoryID(ref.RepoID)
	if err != nil {
		return nil, err
	}

	body, err := buildCommitNDJSON(operations, CommitOptions{
		Branch:          refOrDefault(options.Branch, ref.DefaultBranch),
		BaseBranch:      strings.TrimSpace(options.BaseBranch),
		ExpectedHeadSHA: strings.TrimSpace(options.ExpectedHeadSHA),
		Message:         strings.TrimSpace(options.Message),
		Author:          options.Author,
	})
	if err != nil {
		return nil, err
	}

	result, err := p.client.createCommit(ctx, repoID, body)
	if err != nil {
		return nil, err
	}

	return &CommitResult{
		CommitSHA: result.Commit.CommitSHA,
	}, nil
}

func (p *SupergitProvider) Head(ctx context.Context, ref RepositoryRef, branch string) (string, error) {
	repoID, err := ValidateRepositoryID(ref.RepoID)
	if err != nil {
		return "", err
	}

	target := refOrDefault(branch, ref.DefaultBranch)
	commit, err := p.client.getCommit(ctx, repoID, target)
	if err != nil {
		return "", err
	}

	return commit.CommitSHA, nil
}

type supergitClient struct {
	baseURL    string
	httpClient *http.Client
}

func newSupergitClient(baseURL string) *supergitClient {
	return &supergitClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

type supergitRepoRequest struct {
	ID            string `json:"id"`
	DefaultBranch string `json:"default_branch"`
}

type supergitRepoResponse struct {
	ID            string `json:"id"`
	DefaultBranch string `json:"default_branch"`
}

type supergitListFilesResponse struct {
	Paths []string `json:"paths"`
	Ref   string   `json:"ref"`
}

type supergitCommitResponse struct {
	Commit struct {
		CommitSHA string `json:"commit_sha"`
	} `json:"commit"`
	Result struct {
		NewSHA string `json:"new_sha"`
	} `json:"result"`
}

type supergitGetCommitResponse struct {
	CommitSHA string `json:"commit_sha"`
}

func (c *supergitClient) createRepo(ctx context.Context, req supergitRepoRequest) (*supergitRepoResponse, error) {
	var resp supergitRepoResponse
	if err := c.doJSON(ctx, http.MethodPost, "/repos", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *supergitClient) deleteRepo(ctx context.Context, repoID string) error {
	return c.doJSON(ctx, http.MethodDelete, repoPath(repoID), nil, nil)
}

func (c *supergitClient) listFiles(ctx context.Context, repoID, ref string) (*ListFilesResult, error) {
	query := url.Values{}
	if ref != "" {
		query.Set("ref", ref)
	}

	var resp supergitListFilesResponse
	if err := c.doJSON(ctx, http.MethodGet, repoPath(repoID)+"/files?"+query.Encode(), nil, &resp); err != nil {
		return nil, err
	}

	return &ListFilesResult{Paths: resp.Paths, Ref: resp.Ref}, nil
}

func (c *supergitClient) getFile(ctx context.Context, repoID, filePath, ref string) (io.ReadCloser, error) {
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
		return nil, c.decodeAPIError(resp)
	}

	return resp.Body, nil
}

func (c *supergitClient) createCommit(ctx context.Context, repoID string, body io.Reader) (*supergitCommitResponse, error) {
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
		return nil, c.decodeAPIError(resp)
	}

	var result supergitCommitResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *supergitClient) getCommit(ctx context.Context, repoID, sha string) (*supergitGetCommitResponse, error) {
	query := url.Values{}
	query.Set("sha", sha)

	var resp supergitGetCommitResponse
	if err := c.doJSON(ctx, http.MethodGet, repoPath(repoID)+"/commit?"+query.Encode(), nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *supergitClient) doJSON(ctx context.Context, method, endpoint string, payload any, out any) error {
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
		return c.decodeAPIError(resp)
	}

	if out == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *supergitClient) newRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, body)
}

func (c *supergitClient) decodeAPIError(resp *http.Response) error {
	var payload struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&payload)

	message := strings.TrimSpace(payload.Error)
	if message == "" {
		message = resp.Status
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
		if strings.Contains(message, "expected head sha") {
			return ErrExpectedHeadMismatch
		}
		if strings.Contains(message, "invalid repository path") {
			return ErrInvalidRepositoryID
		}
		return fmt.Errorf("%w: %s", ErrInvalidCommit, message)
	case http.StatusConflict:
		return ErrExpectedHeadMismatch
	case http.StatusRequestEntityTooLarge:
		if strings.Contains(strings.ToLower(message), "file") {
			return ErrFileTooLarge
		}
		return ErrCommitTooLarge
	default:
		return fmt.Errorf("supergit request failed: %s", message)
	}
}

func repoPath(repoID string) string {
	return "/repos/" + url.PathEscape(repoID)
}

func buildCommitNDJSON(operations []validatedOperation, options CommitOptions) (io.Reader, error) {
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
