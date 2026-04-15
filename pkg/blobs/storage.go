package blobs

import (
	"context"
	"io"
	"time"
)

const (
	BackendMemory     = "memory"
	BackendFilesystem = "filesystem"
	BackendS3         = "s3"
	BackendGCS        = "gcs"
)

type ScopeType string

const (
	ScopeOrganization ScopeType = "organization"
	ScopeCanvas       ScopeType = "canvas"
	ScopeNode         ScopeType = "node"
	ScopeExecution    ScopeType = "execution"
)

type Scope struct {
	Type           ScopeType
	OrganizationID string
	CanvasID       string
	NodeID         string
	ExecutionID    string
}

type PutOptions struct {
	ContentType string
}

type BlobInfo struct {
	Path        string
	Size        int64
	ContentType string
	UpdatedAt   time.Time
}

type ListInput struct {
	MaxResults int
	// ContinuationToken is opaque and backend-specific.
	// Callers should only pass tokens previously returned by List.
	ContinuationToken string
}

type ListOutput struct {
	Blobs []BlobInfo
	// NextToken is opaque and backend-specific.
	NextToken string
}

type PresignedURL struct {
	URL       string
	ExpiresAt time.Time
}

type Storage interface {
	Put(ctx context.Context, scope Scope, path string, body io.Reader, opts PutOptions) error
	Get(ctx context.Context, scope Scope, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, scope Scope, path string) error
	List(ctx context.Context, scope Scope, input ListInput) (*ListOutput, error)
	PresignPut(ctx context.Context, scope Scope, path string, opts PutOptions, expiry time.Duration) (*PresignedURL, error)
	PresignGet(ctx context.Context, scope Scope, path string, expiry time.Duration) (*PresignedURL, error)
}
