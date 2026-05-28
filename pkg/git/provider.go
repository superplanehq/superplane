package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/config"
)

var (
	ReservedSuperPlanePath = ".superplane"

	ErrInvalidPath          = errors.New("invalid file path")
	ErrInvalidRepositoryID  = errors.New("invalid repository path")
	ErrReservedPath         = errors.New("path is reserved for SuperPlane")
	ErrFileTooLarge         = errors.New("file exceeds configured size limit")
	ErrCommitTooLarge       = errors.New("commit exceeds configured size limit")
	ErrInvalidCommit        = errors.New("invalid commit")
	ErrRemoteURLUnsupported = errors.New("git remote URLs are not supported by this canvas storage provider")
	ErrExpectedHeadMismatch = errors.New("expected head sha does not match current branch head")
)

type Provider interface {
	CreateRepository(ctx context.Context, spec RepositorySpec) (*Repository, error)
	DeleteRepository(ctx context.Context, ref RepositoryRef) error
	ListFiles(ctx context.Context, ref RepositoryRef, options ListFilesOptions) (*ListFilesResult, error)
	GetFile(ctx context.Context, ref RepositoryRef, options GetFileOptions) (io.ReadCloser, error)
	Commit(ctx context.Context, ref RepositoryRef, options CommitOptions) (*CommitResult, error)
	Head(ctx context.Context, ref RepositoryRef, branch string) (string, error)
}

type Limits struct {
	MaxFileBytes   int64
	MaxCommitBytes int64
}

type RepositorySpec struct {
	OrganizationID uuid.UUID
	CanvasID       uuid.UUID
	RepoID         string
	DefaultBranch  string
}

type Repository struct {
	RepoID        string
	DefaultBranch string
}

type RepositoryRef struct {
	RepoID        string
	DefaultBranch string
}

type ListFilesOptions struct {
	Ref string
}

type ListFilesResult struct {
	Paths []string
	Ref   string
}

type GetFileOptions struct {
	Path string
	Ref  string
}

type FileOperation struct {
	Path      string
	Content   io.Reader
	SizeBytes int64
	Delete    bool
}

type CommitAuthor struct {
	Name  string
	Email string
}

type CommitOptions struct {
	Branch          string
	BaseBranch      string
	ExpectedHeadSHA string
	Message         string
	Author          CommitAuthor
	Operations      []FileOperation
}

type CommitResult struct {
	CommitSHA string
}

type GitCredentialsOptions struct {
	ReadOnly       bool
	TTL            time.Duration
	AllowForcePush bool
}

type GitCredentials struct {
	Username string
	Password string
}

type validatedOperation struct {
	Path      string
	Content   io.Reader
	SizeBytes int64
	Delete    bool
}

func NewProvider(cfg config.CanvasStorageConfig) (Provider, error) {
	switch cfg.Driver {
	case config.CanvasStorageDriverCodeStorage:
		return NewCodeStorageProvider(cfg)
	case config.CanvasStorageDriverSupergit:
		return NewSupergitProvider(cfg)
	case config.CanvasStorageDriverDisabled:
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported canvas storage driver %q", cfg.Driver)
	}
}

func CanvasRepoID(organizationID, canvasID uuid.UUID) string {
	return fmt.Sprintf("orgs/%s/canvases/%s", organizationID.String(), canvasID.String())
}

func ValidateRepositoryID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ReplaceAll(value, "\\", "/"), "/") {
		return "", ErrInvalidRepositoryID
	}

	normalized, err := NormalizePath(value)
	if err != nil {
		return "", ErrInvalidRepositoryID
	}

	segments := strings.Split(normalized, "/")
	if len(segments) != 4 || segments[0] != "orgs" || segments[2] != "canvases" {
		return "", ErrInvalidRepositoryID
	}
	if _, err := uuid.Parse(segments[1]); err != nil {
		return "", ErrInvalidRepositoryID
	}
	if _, err := uuid.Parse(segments[3]); err != nil {
		return "", ErrInvalidRepositoryID
	}

	return normalized, nil
}

func defaultBranch(branch string) string {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return "main"
	}
	return branch
}

func refOrDefault(ref, branch string) string {
	ref = strings.TrimSpace(ref)
	if ref != "" {
		return ref
	}
	return defaultBranch(branch)
}

func ValidateUserPath(value string) (string, error) {
	normalized, err := NormalizePath(value)
	if err != nil {
		return "", err
	}

	if normalized == ReservedSuperPlanePath || strings.HasPrefix(normalized, ReservedSuperPlanePath+"/") {
		return "", ErrReservedPath
	}

	return normalized, nil
}

func NormalizePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsRune(value, '\x00') {
		return "", ErrInvalidPath
	}

	value = strings.ReplaceAll(value, "\\", "/")
	value = strings.TrimLeft(value, "/")
	if value == "" {
		return "", ErrInvalidPath
	}

	normalized := path.Clean(value)
	if normalized == "." || normalized == ".." || strings.HasPrefix(normalized, "../") {
		return "", ErrInvalidPath
	}

	for _, segment := range strings.Split(normalized, "/") {
		if segment == "" || segment == "." || segment == ".." || segment == ".git" {
			return "", ErrInvalidPath
		}
	}

	return normalized, nil
}

func validateCommitOperations(operations []FileOperation, limits Limits) ([]validatedOperation, error) {
	if len(operations) == 0 {
		return nil, fmt.Errorf("%w: at least one file operation is required", ErrInvalidCommit)
	}

	validated := make([]validatedOperation, 0, len(operations))
	var totalBytes int64

	for _, operation := range operations {
		path, err := ValidateUserPath(operation.Path)
		if err != nil {
			return nil, err
		}

		if !operation.Delete {
			if operation.Content == nil {
				return nil, fmt.Errorf("%w: content is required for %q", ErrInvalidPath, path)
			}
			if operation.SizeBytes < 0 {
				return nil, fmt.Errorf("%w: size is required for %q", ErrInvalidPath, path)
			}
			if limits.MaxFileBytes > 0 && operation.SizeBytes > limits.MaxFileBytes {
				return nil, fmt.Errorf("%w: %q", ErrFileTooLarge, path)
			}
			totalBytes += operation.SizeBytes
		}

		validated = append(validated, validatedOperation{
			Path:      path,
			Content:   operation.Content,
			SizeBytes: operation.SizeBytes,
			Delete:    operation.Delete,
		})
	}

	if limits.MaxCommitBytes > 0 && totalBytes > limits.MaxCommitBytes {
		return nil, ErrCommitTooLarge
	}

	return validated, nil
}

func validateCommitMetadata(message string, author CommitAuthor) error {
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("%w: commit message is required", ErrInvalidCommit)
	}
	if strings.TrimSpace(author.Name) == "" {
		return fmt.Errorf("%w: author name is required", ErrInvalidCommit)
	}
	if strings.TrimSpace(author.Email) == "" {
		return fmt.Errorf("%w: author email is required", ErrInvalidCommit)
	}
	return nil
}
