package provider

import (
	"context"
	"errors"
	"io"

	"github.com/google/uuid"
)

/*
 * Two available git storage providers:
 * - codestorage - https://code.storage
 * - supergit - https://github.com/superplanehq/supergit
 */
const (
	CodeStorageProvider = "codestorage"
	SuperGitProvider    = "supergit"
)

/*
 * Users should not be able to create files in this directory.
 * I am not really sure we need this protection, but it's better
 * to start being restrictive and open up later on, than the other way around.
 */
const ReservedSuperPlanePath = ".superplane"

var (
	ErrInvalidPath          = errors.New("invalid file path")
	ErrInvalidRepositoryID  = errors.New("invalid repository path")
	ErrReservedPath         = errors.New("path is reserved for SuperPlane")
	ErrInvalidCommit        = errors.New("invalid commit")
	ErrExpectedHeadMismatch = errors.New("expected head sha does not match current branch head")
)

type Provider interface {
	CreateRepository(ctx context.Context, options CreateRepositoryOptions) (*Repository, error)
	DeleteRepository(ctx context.Context, repoID string) error
	ListFiles(ctx context.Context, repoID string) (*ListFilesResult, error)
	GetFile(ctx context.Context, repoID string, path string) (io.ReadCloser, error)
	Commit(ctx context.Context, repoID string, options CommitOptions) (*CommitResult, error)
	Head(ctx context.Context, repoID string, branch string) (string, error)
}

type CreateRepositoryOptions struct {
	OrganizationID uuid.UUID
	CanvasID       uuid.UUID
}

type Repository struct {
	ID string
}

type ListFilesResult struct {
	Paths []string
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
