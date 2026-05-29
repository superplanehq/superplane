package inmemory

import (
	"context"
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/git/provider"
)

/*
 * In-memory git provider.
 *
 * This provider is used for testing purposes.
 * It does not store any data in a real git repository.
 * It is used to test the git provider interface and implementation.
 * It is not meant to be used in production.
 */
type Provider struct {
	repositories map[string]provider.Repository
}

func NewProvider() *Provider {
	return &Provider{
		repositories: map[string]provider.Repository{},
	}
}

func (p *Provider) Name() string {
	return "memory"
}

func (p *Provider) GetRepositoryID(options provider.RepositoryOptions) string {
	return fmt.Sprintf("orgs/%s/canvases/%s", options.OrganizationID.String(), options.CanvasID.String())
}

func (p *Provider) CreateRepository(ctx context.Context, repoID string) (*provider.Repository, error) {
	repo := provider.Repository{ID: repoID}
	p.repositories[repoID] = repo
	return &repo, nil
}

func (p *Provider) DeleteRepository(ctx context.Context, repoID string) error {
	delete(p.repositories, repoID)
	return nil
}

func (p *Provider) ListFiles(ctx context.Context, repoID string) ([]string, error) {
	return nil, nil
}

func (p *Provider) GetFile(ctx context.Context, repoID string, path string) (io.ReadCloser, error) {
	return nil, nil
}

func (p *Provider) Commit(ctx context.Context, repoID string, options provider.CommitOptions) (string, error) {
	return "", nil
}

func (p *Provider) Head(ctx context.Context, repoID string) (string, error) {
	return "", nil
}
