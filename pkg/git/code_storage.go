package git

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/config"

	codestorage "github.com/pierrecomputer/sdk/packages/code-storage-go"
)

type CodeStorageProvider struct {
	client        *codestorage.Client
	defaultBranch string
	limits        Limits
}

func NewCodeStorageProvider(cfg config.CanvasStorageConfig) (*CodeStorageProvider, error) {
	if cfg.CodeStoragePrivateKeyPath == "" {
		return nil, fmt.Errorf("CODE_STORAGE_PRIVATE_KEY_PATH is required")
	}

	key, err := os.ReadFile(cfg.CodeStoragePrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read Code Storage private key: %w", err)
	}

	client, err := codestorage.NewClient(codestorage.Options{
		Name: cfg.CodeStorageName,
		Key:  string(key),
	})

	if err != nil {
		return nil, err
	}

	return &CodeStorageProvider{
		client:        client,
		defaultBranch: defaultBranch(cfg.DefaultBranch),
		limits: Limits{
			MaxFileBytes:   cfg.MaxFileBytes,
			MaxCommitBytes: cfg.MaxCommitBytes,
		},
	}, nil
}

func (p *CodeStorageProvider) CreateRepository(ctx context.Context, spec RepositorySpec) (*Repository, error) {
	repoID := strings.TrimSpace(spec.RepoID)
	if repoID == "" {
		repoID = CanvasRepoID(spec.OrganizationID, spec.CanvasID)
	}

	branch := defaultBranch(spec.DefaultBranch)
	if branch == "main" {
		branch = p.defaultBranch
	}

	repo, err := p.client.CreateRepo(ctx, codestorage.CreateRepoOptions{
		ID:            repoID,
		DefaultBranch: branch,
	})

	if err != nil {
		return nil, err
	}

	repoBranch := defaultBranch(repo.DefaultBranch)

	return &Repository{
		RepoID:        repo.ID,
		DefaultBranch: repoBranch,
	}, nil
}

func (p *CodeStorageProvider) InitRepository(ctx context.Context, ref RepositoryRef, branch string) error {
	repo, err := p.repo(ref)
	if err != nil {
		return err
	}

	branch = defaultBranch(branch)
	options := initialRepositoryCommitOptions(branch)

	builder, err := repo.CreateCommit(codestorage.CommitOptions{
		TargetBranch:  branch,
		CommitMessage: options.Message,
		Author: codestorage.CommitSignature{
			Name:  options.Author.Name,
			Email: options.Author.Email,
		},
	})
	if err != nil {
		return err
	}

	for _, operation := range options.Operations {
		builder.AddFile(operation.Path, operation.Content, nil)
	}

	if err := builder.Err(); err != nil {
		return err
	}

	_, err = builder.Send(ctx)
	return err
}

func (p *CodeStorageProvider) DeleteRepository(ctx context.Context, ref RepositoryRef) error {
	repoID, err := ValidateRepositoryID(ref.RepoID)
	if err != nil {
		return err
	}

	repo, err := p.client.FindOne(ctx, codestorage.FindOneOptions{ID: repoID})
	if err != nil {
		return err
	}

	//
	// If repo does not exist, do not do anything.
	//
	if repo == nil {
		return nil
	}

	_, err = p.client.DeleteRepo(ctx, codestorage.DeleteRepoOptions{ID: repoID})
	if err != nil {
		return err
	}

	return nil
}

func (p *CodeStorageProvider) ListFiles(ctx context.Context, ref RepositoryRef, options ListFilesOptions) (*ListFilesResult, error) {
	repo, err := p.repo(ref)
	if err != nil {
		return nil, err
	}

	result, err := repo.ListFiles(ctx, codestorage.ListFilesOptions{
		Ref: refOrDefault(options.Ref, ref.DefaultBranch),
	})
	if err != nil {
		return nil, err
	}

	return &ListFilesResult{Paths: result.Paths, Ref: result.Ref}, nil
}

func (p *CodeStorageProvider) GetFile(ctx context.Context, ref RepositoryRef, options GetFileOptions) (io.ReadCloser, error) {
	filePath, err := NormalizePath(options.Path)
	if err != nil {
		return nil, err
	}

	repo, err := p.repo(ref)
	if err != nil {
		return nil, err
	}

	resp, err := repo.FileStream(ctx, codestorage.GetFileOptions{
		Path: filePath,
		Ref:  refOrDefault(options.Ref, ref.DefaultBranch),
	})
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (p *CodeStorageProvider) Commit(ctx context.Context, ref RepositoryRef, options CommitOptions) (*CommitResult, error) {
	if err := validateCommitMetadata(options.Message, options.Author); err != nil {
		return nil, err
	}

	operations, err := validateCommitOperations(options.Operations, p.limits)
	if err != nil {
		return nil, err
	}

	repo, err := p.repo(ref)
	if err != nil {
		return nil, err
	}

	builder, err := repo.CreateCommit(codestorage.CommitOptions{
		TargetBranch:    refOrDefault(options.Branch, ref.DefaultBranch),
		BaseBranch:      strings.TrimSpace(options.BaseBranch),
		ExpectedHeadSHA: strings.TrimSpace(options.ExpectedHeadSHA),
		CommitMessage:   strings.TrimSpace(options.Message),
		Author: codestorage.CommitSignature{
			Name:  strings.TrimSpace(options.Author.Name),
			Email: strings.TrimSpace(options.Author.Email),
		},
	})
	if err != nil {
		return nil, err
	}

	for _, operation := range operations {
		if operation.Delete {
			builder.DeletePath(operation.Path)
			continue
		}
		builder.AddFile(operation.Path, operation.Content, nil)
	}

	result, err := builder.Send(ctx)
	if err != nil {
		return nil, err
	}

	return &CommitResult{
		CommitSHA: result.CommitSHA,
	}, nil
}

func (p *CodeStorageProvider) Head(ctx context.Context, ref RepositoryRef, branch string) (string, error) {
	repo, err := p.repo(ref)
	if err != nil {
		return "", err
	}

	//
	// I don't like that I have to list all branches to find the head SHA,
	// but right now, there's
	//
	target := refOrDefault(branch, ref.DefaultBranch)
	cursor := ""
	for {
		result, err := repo.ListBranches(ctx, codestorage.ListBranchesOptions{
			Cursor: cursor,
			Limit:  100,
		})

		if err != nil {
			return "", err
		}

		for _, candidate := range result.Branches {
			if candidate.Name == target {
				return candidate.HeadSHA, nil
			}
		}

		if !result.HasMore || result.NextCursor == "" {
			return "", nil
		}

		cursor = result.NextCursor
	}
}

func (p *CodeStorageProvider) repo(ref RepositoryRef) (*codestorage.Repo, error) {
	return p.client.Repo(codestorage.RepoOptions{
		ID:            strings.TrimSpace(ref.RepoID),
		DefaultBranch: defaultBranch(ref.DefaultBranch),
	})
}
