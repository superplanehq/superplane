package blobstorage

import (
	"context"
	"io"
)

const (
	BackendMemory     = "memory"
	BackendFilesystem = "filesystem"
	BackendS3         = "s3"
	BackendGCS        = "gcs"
)

type PutInput struct {
	Key         string
	Body        io.Reader
	Size        int64
	ContentType string
}

type PutOutput struct {
	ETag string
}

type GetOutput struct {
	Body        io.ReadCloser
	Size        int64
	ContentType string
}

type BlobStorage interface {
	Put(ctx context.Context, input PutInput) (*PutOutput, error)
	Get(ctx context.Context, key string) (*GetOutput, error)
	Delete(ctx context.Context, key string) error
}
