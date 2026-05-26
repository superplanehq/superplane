package canvasstorage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/config"

	codestorage "github.com/pierrecomputer/sdk/packages/code-storage-go"
)

const (
	initialRepositoryFilePath      = "README.md"
	initialRepositoryCommitMessage = "Initialize repository"
	initialRepositoryAuthorName    = "SuperPlane"
	initialRepositoryAuthorEmail   = "bot@superplane.local"
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
	created := err == nil
	if err != nil {
		if !isCodeStorageRepositoryAlreadyExists(err) {
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

	repoBranch := defaultBranch(repo.DefaultBranch)
	if created {
		if _, err := p.initializeRepository(ctx, repo, repoBranch); err != nil {
			return nil, err
		}
	}

	return &Repository{
		RepoID:        repo.ID,
		DefaultBranch: repoBranch,
	}, nil
}

func (p *CodeStorageProvider) initializeRepository(ctx context.Context, repo *codestorage.Repo, branch string) (*CommitResult, error) {
	builder, err := repo.CreateCommit(codestorage.CommitOptions{
		TargetBranch:  defaultBranch(branch),
		CommitMessage: initialRepositoryCommitMessage,
		Author: codestorage.CommitSignature{
			Name:  initialRepositoryAuthorName,
			Email: initialRepositoryAuthorEmail,
		},
	})
	if err != nil {
		return nil, err
	}

	builder.AddFile(initialRepositoryFilePath, strings.NewReader(""), nil)
	if err := builder.Err(); err != nil {
		return nil, err
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

func (p *CodeStorageProvider) DeleteRepository(ctx context.Context, ref RepositoryRef) error {
	repoID, err := ValidateRepositoryID(ref.RepoID)
	if err != nil {
		return err
	}

	_, err = p.client.DeleteRepo(ctx, codestorage.DeleteRepoOptions{ID: repoID})
	if err != nil {
		if isCodeStorageRepositoryAlreadyDeleted(err) {
			return nil
		}

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

func (p *CodeStorageProvider) CurrentHead(ctx context.Context, ref RepositoryRef, branch string) (string, error) {
	repo, err := p.repo(ref)
	if err != nil {
		return "", err
	}

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

func (p *CodeStorageProvider) GitURL(ctx context.Context, ref RepositoryRef) (string, error) {
	remote, err := p.authenticatedRemoteURL(ctx, ref, GitCredentialsOptions{ReadOnly: true})
	if err != nil {
		return "", err
	}

	u, err := url.Parse(remote)
	if err != nil {
		return "", err
	}

	u.User = nil
	return u.String(), nil
}

func (p *CodeStorageProvider) GenerateGitCredentials(ctx context.Context, ref RepositoryRef, options GitCredentialsOptions) (*GitCredentials, error) {
	remote, err := p.authenticatedRemoteURL(ctx, ref, options)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(remote)
	if err != nil {
		return nil, err
	}

	password, _ := u.User.Password()
	return &GitCredentials{
		Username: u.User.Username(),
		Password: password,
	}, nil
}

func (p *CodeStorageProvider) authenticatedRemoteURL(ctx context.Context, ref RepositoryRef, options GitCredentialsOptions) (string, error) {
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

func isCodeStorageRepositoryAlreadyExists(err error) bool {
	var apiErr *codestorage.APIError
	if errors.As(err, &apiErr) && apiErr.Status == 409 {
		return true
	}

	return strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "repository already exists")
}

func isCodeStorageRepositoryAlreadyDeleted(err error) bool {
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "repository not found") ||
		strings.Contains(message, "repository already deleted")
}
