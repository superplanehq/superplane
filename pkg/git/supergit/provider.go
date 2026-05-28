package supergit

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/uuid"
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

func (p *Provider) CreateRepository(ctx context.Context, options provider.CreateRepositoryOptions) (*provider.Repository, error) {
	repoID := p.getRepositoryID(options.OrganizationID, options.CanvasID)
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

func (p *Provider) ListFiles(ctx context.Context, repoID string) (*provider.ListFilesResult, error) {
	return p.client.listFiles(ctx, repoID, p.defaultBranch)
}

func (p *Provider) GetFile(ctx context.Context, repoID string, path string) (io.ReadCloser, error) {
	filePath, err := provider.NormalizePath(path)
	if err != nil {
		return nil, err
	}

	return p.client.getFile(ctx, repoID, filePath, p.defaultBranch)
}

func (p *Provider) Commit(ctx context.Context, repoID string, options provider.CommitOptions) (*provider.CommitResult, error) {
	if err := provider.ValidateCommitMetadata(options.Message, options.Author); err != nil {
		return nil, err
	}

	operations, err := provider.ValidateCommitOperations(options.Operations)
	if err != nil {
		return nil, err
	}

	body, err := buildCommitNDJSON(operations, provider.CommitOptions{
		Branch:          provider.RefOrDefault(options.Branch, p.defaultBranch),
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

	return &provider.CommitResult{
		CommitSHA: result.Commit.CommitSHA,
	}, nil
}

func (p *Provider) Head(ctx context.Context, repoID string) (string, error) {
	commit, err := p.client.getCommit(ctx, repoID, p.defaultBranch)
	if err != nil {
		return "", err
	}

	return commit.CommitSHA, nil
}

func (p *Provider) getRepositoryID(organizationID, canvasID uuid.UUID) string {
	return fmt.Sprintf("orgs/%s/canvases/%s", organizationID.String(), canvasID.String())
}
