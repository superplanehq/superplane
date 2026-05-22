package canvasstorage

import (
	"context"
	"errors"
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
	key := cfg.CodeStoragePrivateKey
	if key == "" && cfg.CodeStoragePrivateKeyPath != "" {
		bytes, err := os.ReadFile(cfg.CodeStoragePrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read Code Storage private key: %w", err)
		}
		key = string(bytes)
	}

	client, err := codestorage.NewClient(codestorage.Options{
		Name:           cfg.CodeStorageName,
		Key:            key,
		APIBaseURL:     cfg.CodeStorageAPIBaseURL,
		StorageBaseURL: cfg.CodeStorageStorageBaseURL,
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

func (p *CodeStorageProvider) EnsureRepository(ctx context.Context, spec RepositorySpec) (*Repository, error) {
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
		var apiErr *codestorage.APIError
		if !errors.As(err, &apiErr) || apiErr.Status != 409 {
			return nil, err
		}

		repo, err = p.client.FindOne(ctx, codestorage.FindOneOptions{ID: repoID})
		if err != nil {
			return nil, err
		}
		if repo == nil {
			return nil, fmt.Errorf("Code Storage repository %q already exists but could not be loaded", repoID)
		}
	}

	return &Repository{
		RepoID:        repo.ID,
		DefaultBranch: defaultBranch(repo.DefaultBranch),
	}, nil
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

func (p *CodeStorageProvider) CommitFiles(ctx context.Context, ref RepositoryRef, options CommitFilesOptions) (*CommitResult, error) {
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
		OldSHA:    result.RefUpdate.OldSHA,
		NewSHA:    result.RefUpdate.NewSHA,
		Branch:    result.RefUpdate.Branch,
	}, nil
}

func (p *CodeStorageProvider) RemoteURL(ctx context.Context, ref RepositoryRef, options RemoteURLOptions) (string, error) {
	permissions := []codestorage.Permission{codestorage.PermissionGitRead}
	var ops codestorage.Ops
	if !options.ReadOnly {
		permissions = append(permissions, codestorage.PermissionGitWrite)
		if !options.AllowForcePush {
			ops = append(ops, codestorage.OpNoForcePush)
		}
	}

	repo, err := p.repo(ref)
	if err != nil {
		return "", err
	}

	return repo.RemoteURL(ctx, codestorage.RemoteURLOptions{
		Permissions: permissions,
		TTL:         options.TTL,
		Ops:         ops,
	})
}

func (p *CodeStorageProvider) repo(ref RepositoryRef) (*codestorage.Repo, error) {
	return p.client.Repo(codestorage.RepoOptions{
		ID:            strings.TrimSpace(ref.RepoID),
		DefaultBranch: defaultBranch(ref.DefaultBranch),
	})
}
