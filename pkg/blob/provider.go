package blob

import (
	"context"
	"errors"
	"io"
	"time"
)

const (
	ProviderGCS        = "gcs"
	ProviderFilesystem = "filesystem"
)

var (
	ErrNotFound              = errors.New("blob not found")
	ErrSignedURLUnsupported  = errors.New("blob provider cannot mint object signed URLs")
	ErrProviderNotConfigured = errors.New("blob storage provider is not configured")
)

type PutOptions struct {
	ContentType string
}

type ObjectInfo struct {
	Size        int64
	ContentType string
}

// Provider stores opaque object bytes. Catalog metadata lives in Postgres.
type Provider interface {
	Name() string
	Put(ctx context.Context, key string, r io.Reader, opts PutOptions) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Head(ctx context.Context, key string) (*ObjectInfo, error)
	Delete(ctx context.Context, key string) error
	// SignedGetURL returns a time-limited HTTPS GET for the object.
	// Filesystem returns ErrSignedURLUnsupported; callers mint an HMAC file URL.
	SignedGetURL(ctx context.Context, key string, ttl time.Duration) (string, error)
}
