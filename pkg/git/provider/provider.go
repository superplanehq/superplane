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
	ErrInvalidRef           = errors.New("invalid git ref")
	ErrReservedPath         = errors.New("path is reserved for SuperPlane")
	ErrInvalidCommit        = errors.New("invalid commit")
	ErrExpectedHeadMismatch = errors.New("expected head sha does not match current branch head")
)

type Provider interface {

	//
	// The unique identifier of the provider.
	//
	Name() string

	//
	// Get the provider specific repository identifier,
	// Allows providers to define how they map metadata about a repository,
	// to its identifier.
	//
	GetRepositoryID(options RepositoryOptions) string

	//
	// Repository management methods.
	//
	CreateRepository(ctx context.Context, repoID string) (*Repository, error)
	DeleteRepository(ctx context.Context, repoID string) error

	//
	// File management methods
	//
	ListFiles(ctx context.Context, repoID, ref string) ([]string, error)
	GetFile(ctx context.Context, repoID, path, ref string) (io.ReadCloser, error)
	Commit(ctx context.Context, repoID string, options CommitOptions) (string, error)
	Head(ctx context.Context, repoID, ref string) (string, error)

	//
	// Branch management methods
	//
	ListBranches(ctx context.Context, repoID, prefix string) ([]string, error)
	CreateBranch(ctx context.Context, repoID, branch, fromRef string) error
	MergeBranch(ctx context.Context, repoID, sourceBranch, targetBranch, message string, author CommitAuthor) (string, error)
	DeleteBranch(ctx context.Context, repoID, branch string) error
}

type RepositoryOptions struct {
	OrganizationID uuid.UUID
	CanvasID       uuid.UUID
}

type Repository struct {
	ID string
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
