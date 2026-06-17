package supergit

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/git/provider"
)

type Provider struct {
	client        *Client
	defaultBranch string
}

func NewProvider() (*Provider, error) {
	baseURL := strings.TrimSpace(os.Getenv("GIT_STORAGE_SUPERGIT_BASE_URL"))
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	return &Provider{
		client:        NewClient(baseURL),
		defaultBranch: "main",
	}, nil
}

func (p *Provider) Name() string {
	return provider.SuperGitProvider
}

func (p *Provider) GetRepositoryID(options provider.RepositoryOptions) string {
	return fmt.Sprintf("orgs/%s/canvases/%s", options.OrganizationID.String(), options.CanvasID.String())
}

func (p *Provider) CreateRepository(ctx context.Context, repoID string) (*provider.Repository, error) {
	repo, err := p.client.createRepo(ctx, RepoRequest{
		ID:            repoID,
		DefaultBranch: "main",
	})

	if err != nil {
		return nil, err
	}

	_, err = p.Commit(ctx, repo.ID, provider.InitialRepositoryCommitOptions(p.defaultBranch))
	if err != nil {
		return nil, err
	}

	return &provider.Repository{
		ID: repo.ID,
	}, nil
}

func (p *Provider) DeleteRepository(ctx context.Context, repoID string) error {
	return p.client.deleteRepo(ctx, repoID)
}

func (p *Provider) ListFiles(ctx context.Context, repoID, ref string) ([]string, error) {
	return p.client.listFiles(ctx, repoID, provider.RefOrDefault(ref, p.defaultBranch))
}

func (p *Provider) GetFile(ctx context.Context, repoID, path, ref string) (io.ReadCloser, error) {
	filePath, err := provider.NormalizePath(path)
	if err != nil {
		return nil, err
	}

	return p.client.getFile(ctx, repoID, filePath, provider.RefOrDefault(ref, p.defaultBranch))
}

func (p *Provider) Commit(ctx context.Context, repoID string, options provider.CommitOptions) (string, error) {
	if err := provider.ValidateCommitMetadata(options.Message, options.Author); err != nil {
		return "", err
	}

	operations, err := provider.ValidateCommitOperations(options.Operations)
	if err != nil {
		return "", err
	}

	body, err := buildCommitNDJSON(operations, provider.CommitOptions{
		Branch:          provider.RefOrDefault(options.Branch, p.defaultBranch),
		BaseBranch:      strings.TrimSpace(options.BaseBranch),
		ExpectedHeadSHA: strings.TrimSpace(options.ExpectedHeadSHA),
		Message:         strings.TrimSpace(options.Message),
		Author:          options.Author,
	})

	if err != nil {
		return "", err
	}

	result, err := p.client.createCommit(ctx, repoID, body)
	if err != nil {
		return "", err
	}

	return result.Commit.CommitSHA, nil
}

func (p *Provider) Head(ctx context.Context, repoID, ref string) (string, error) {
	commit, err := p.client.getCommit(ctx, repoID, provider.RefOrDefault(ref, p.defaultBranch))
	if err != nil {
		return "", err
	}

	return commit.CommitSHA, nil
}

func (p *Provider) ListBranches(ctx context.Context, repoID, prefix string) ([]string, error) {
	return p.client.listBranches(ctx, repoID, prefix)
}

func (p *Provider) CreateBranch(ctx context.Context, repoID, branch, fromRef string) error {
	return p.client.createBranch(ctx, repoID, CreateBranchRequest{
		Branch:  strings.TrimSpace(branch),
		FromRef: provider.RefOrDefault(fromRef, p.defaultBranch),
	})
}

func (p *Provider) MergeBranch(ctx context.Context, repoID, sourceBranch, targetBranch, message string, author provider.CommitAuthor) (string, error) {
	if err := provider.ValidateCommitMetadata(message, author); err != nil {
		return "", err
	}

	req := MergeBranchRequest{
		SourceBranch: strings.TrimSpace(sourceBranch),
		TargetBranch: provider.RefOrDefault(targetBranch, p.defaultBranch),
		Message:      strings.TrimSpace(message),
	}
	req.Author.Name = strings.TrimSpace(author.Name)
	req.Author.Email = strings.TrimSpace(author.Email)

	return p.client.mergeBranch(ctx, repoID, req)
}

func (p *Provider) DeleteBranch(ctx context.Context, repoID, branch string) error {
	return p.client.deleteBranch(ctx, repoID, strings.TrimSpace(branch))
}
