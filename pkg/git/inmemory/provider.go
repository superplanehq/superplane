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
	"strings"

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
	defaultBranch string
	branches      map[string]string
	snapshots     map[string]map[string][]byte
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

	headSHA := initialHeadSHA(repoID)
	p.repositories[repoID] = &repositoryState{
		defaultBranch: "main",
		branches: map[string]string{
			"main": headSHA,
		},
		snapshots: map[string]map[string][]byte{
			headSHA: cloneFiles(map[string][]byte{
				"README.md": {},
			}),
		},
	}

	return &provider.Repository{ID: repoID}, nil
}

func (p *Provider) DeleteRepository(_ context.Context, repoID string) error {
	delete(p.repositories, repoID)
	return nil
}

func (p *Provider) ListFiles(_ context.Context, repoID, ref string) ([]string, error) {
	repository, err := p.repository(repoID)
	if err != nil {
		return nil, err
	}

	files, err := repository.filesAtRef(ref)
	if err != nil {
		return nil, err
	}

	paths := slices.Collect(maps.Keys(files))
	sort.Strings(paths)
	return paths, nil
}

func (p *Provider) GetFile(_ context.Context, repoID, path, ref string) (io.ReadCloser, error) {
	repository, err := p.repository(repoID)
	if err != nil {
		return nil, err
	}

	files, err := repository.filesAtRef(ref)
	if err != nil {
		return nil, err
	}

	content, ok := files[path]
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

	branch := provider.RefOrDefault(options.Branch, repository.defaultBranch)
	currentHead, ok := repository.branches[branch]
	if !ok {
		return "", provider.ErrInvalidRef
	}

	if options.ExpectedHeadSHA != "" && options.ExpectedHeadSHA != currentHead {
		return "", provider.ErrExpectedHeadMismatch
	}

	files, err := repository.filesAtRef(currentHead)
	if err != nil {
		return "", err
	}

	for _, operation := range options.Operations {
		if operation.Delete {
			delete(files, operation.Path)
			continue
		}

		content, err := io.ReadAll(operation.Content)
		if err != nil {
			return "", fmt.Errorf("read file content: %w", err)
		}

		files[operation.Path] = content
	}

	newSHA := nextHeadSHA(currentHead, options.Message)
	repository.snapshots[newSHA] = files
	repository.branches[branch] = newSHA
	return newSHA, nil
}

func (p *Provider) Head(_ context.Context, repoID, ref string) (string, error) {
	repository, err := p.repository(repoID)
	if err != nil {
		return "", err
	}

	return repository.resolveRef(ref)
}

func (p *Provider) ListBranches(_ context.Context, repoID, prefix string) ([]string, error) {
	repository, err := p.repository(repoID)
	if err != nil {
		return nil, err
	}

	branches := slices.Collect(maps.Keys(repository.branches))
	sort.Strings(branches)
	if prefix == "" {
		return branches, nil
	}

	filtered := make([]string, 0, len(branches))
	for _, branch := range branches {
		if strings.HasPrefix(branch, prefix) {
			filtered = append(filtered, branch)
		}
	}

	return filtered, nil
}

func (p *Provider) CreateBranch(_ context.Context, repoID, branch, fromRef string) error {
	repository, err := p.repository(repoID)
	if err != nil {
		return err
	}

	branch = strings.TrimSpace(branch)
	if branch == "" {
		return provider.ErrInvalidRef
	}

	if _, ok := repository.branches[branch]; ok {
		return fmt.Errorf("%w: branch %q already exists", provider.ErrInvalidRef, branch)
	}

	fromSHA, err := repository.resolveRef(fromRef)
	if err != nil {
		return err
	}

	repository.branches[branch] = fromSHA
	return nil
}

func (p *Provider) MergeBranch(_ context.Context, repoID, sourceBranch, targetBranch, message string, _ provider.CommitAuthor) (string, error) {
	repository, err := p.repository(repoID)
	if err != nil {
		return "", err
	}

	sourceSHA, err := repository.resolveRef(sourceBranch)
	if err != nil {
		return "", err
	}

	targetSHA, err := repository.resolveRef(targetBranch)
	if err != nil {
		return "", err
	}

	targetFiles, err := repository.filesAtRef(targetSHA)
	if err != nil {
		return "", err
	}

	sourceFiles, err := repository.filesAtRef(sourceSHA)
	if err != nil {
		return "", err
	}

	for path, content := range sourceFiles {
		targetFiles[path] = append([]byte(nil), content...)
	}

	newSHA := nextHeadSHA(targetSHA, message)
	repository.snapshots[newSHA] = targetFiles
	repository.branches[targetBranch] = newSHA
	return newSHA, nil
}

func (p *Provider) DeleteBranch(_ context.Context, repoID, branch string) error {
	repository, err := p.repository(repoID)
	if err != nil {
		return err
	}

	branch = strings.TrimSpace(branch)
	if branch == "" {
		return provider.ErrInvalidRef
	}

	if branch == repository.defaultBranch {
		return fmt.Errorf("%w: cannot delete default branch", provider.ErrInvalidRef)
	}

	if _, ok := repository.branches[branch]; !ok {
		return provider.ErrInvalidRef
	}

	delete(repository.branches, branch)
	return nil
}

func (p *Provider) repository(repoID string) (*repositoryState, error) {
	repository, ok := p.repositories[repoID]
	if !ok {
		return nil, provider.ErrInvalidRepositoryID
	}

	return repository, nil
}

func (r *repositoryState) resolveRef(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		ref = r.defaultBranch
	}

	if sha, ok := r.branches[ref]; ok {
		return sha, nil
	}

	if _, ok := r.snapshots[ref]; ok {
		return ref, nil
	}

	return "", provider.ErrInvalidRef
}

func (r *repositoryState) filesAtRef(ref string) (map[string][]byte, error) {
	sha, err := r.resolveRef(ref)
	if err != nil {
		return nil, err
	}

	files, ok := r.snapshots[sha]
	if !ok {
		return nil, provider.ErrInvalidRef
	}

	return cloneFiles(files), nil
}

func cloneFiles(files map[string][]byte) map[string][]byte {
	cloned := make(map[string][]byte, len(files))
	for path, content := range files {
		cloned[path] = append([]byte(nil), content...)
	}
	return cloned
}

func initialHeadSHA(repoID string) string {
	sum := sha256.Sum256([]byte("initial:" + repoID))
	return hex.EncodeToString(sum[:20])
}

func nextHeadSHA(currentHeadSHA, message string) string {
	sum := sha256.Sum256([]byte(currentHeadSHA + ":" + message))
	return hex.EncodeToString(sum[:20])
}
