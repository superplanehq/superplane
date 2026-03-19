package storage

import (
	"context"
	"io"
)

type Storage interface {
	Class() string
	Provision(ctx context.Context) error
	Write(ctx context.Context, path string, body io.Reader) error
	Read(ctx context.Context, path string) (io.Reader, error)
	Delete(ctx context.Context, path string) error
}
