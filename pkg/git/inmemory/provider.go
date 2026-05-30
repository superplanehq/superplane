package inmemory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"maps"
	"slices"
	"sort"

	"github.com/superplanehq/superplane/pkg/git/provider"
)

/*
 * In-memory git provider.
 *
 * This provider is used for testing purposes.
 * It does not store data in a real git repository.
 * It is not meant to be used in production.
 */
type Provider struct {
	repositories map[string]*repositoryState
}

type repositoryState struct {
	files   map[string][]byte
	headSHA string
}

func NewProvider() *Provider {
	return &Provider{
		repositories: map[string]*repositoryState{},
	}
}

func (p *Provider) Name() string {
	return "memory"
}

func (p *Provider) GetRepositoryID(options provider.RepositoryOptions) string {
	return fmt.Sprintf("orgs/%s/canvases/%s", options.OrganizationID.String(), options.CanvasID.String())
}

func (p *Provider) CreateRepository(_ context.Context, repoID string) (*provider.Repository, error) {
	if _, ok := p.repositories[repoID]; ok {
		return nil, provider.ErrInvalidRepositoryID
	}

	p.repositories[repoID] = &repositoryState{
		files: map[string][]byte{
			"README.md": {},
		},
		headSHA: initialHeadSHA(repoID),
	}

	return &provider.Repository{ID: repoID}, nil
}

func (p *Provider) DeleteRepository(_ context.Context, repoID string) error {
	delete(p.repositories, repoID)
	return nil
}

func (p *Provider) ListFiles(_ context.Context, repoID string) ([]string, error) {
	repository, err := p.repository(repoID)
	if err != nil {
		return nil, err
	}

	paths := slices.Collect(maps.Keys(repository.files))
	sort.Strings(paths)
	return paths, nil
}

func (p *Provider) GetFile(_ context.Context, repoID string, path string) (io.ReadCloser, error) {
	repository, err := p.repository(repoID)
	if err != nil {
		return nil, err
	}

	content, ok := repository.files[path]
	if !ok {
		return nil, provider.ErrInvalidPath
	}

	return io.NopCloser(bytes.NewReader(content)), nil
}

func (p *Provider) Commit(_ context.Context, repoID string, options provider.CommitOptions) (string, error) {
	repository, err := p.repository(repoID)
	if err != nil {
		return "", err
	}

	if err := provider.ValidateCommitMetadata(options.Message, options.Author); err != nil {
		return "", err
	}

	operations, err := provider.ValidateCommitOperations(options.Operations)
	if err != nil {
		return "", err
	}
	options.Operations = operations

	if options.ExpectedHeadSHA != repository.headSHA {
		return "", provider.ErrExpectedHeadMismatch
	}

	for _, operation := range options.Operations {
		if operation.Delete {
			delete(repository.files, operation.Path)
			continue
		}

		content, err := io.ReadAll(operation.Content)
		if err != nil {
			return "", fmt.Errorf("read file content: %w", err)
		}

		repository.files[operation.Path] = content
	}

	repository.headSHA = nextHeadSHA(repository.headSHA, options.Message)
	return repository.headSHA, nil
}

func (p *Provider) Head(_ context.Context, repoID string) (string, error) {
	repository, err := p.repository(repoID)
	if err != nil {
		return "", err
	}

	return repository.headSHA, nil
}

func (p *Provider) repository(repoID string) (*repositoryState, error) {
	repository, ok := p.repositories[repoID]
	if !ok {
		return nil, provider.ErrInvalidRepositoryID
	}

	return repository, nil
}

func initialHeadSHA(repoID string) string {
	sum := sha256.Sum256([]byte("initial:" + repoID))
	return hex.EncodeToString(sum[:8])
}

func nextHeadSHA(currentHeadSHA, message string) string {
	sum := sha256.Sum256([]byte(currentHeadSHA + ":" + message))
	return hex.EncodeToString(sum[:8])
}
