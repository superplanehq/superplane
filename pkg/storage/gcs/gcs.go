package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	gcs "cloud.google.com/go/storage"
)

/*
 * GCSStorage is a storage implementation that uses Google Cloud Storage.
 */
type GCSStorage struct {
	bucket    string
	projectID string
	client    *gcs.Client
}

func NewGCSStorage(ctx context.Context, bucket, projectID string) (*GCSStorage, error) {
	client, err := gcs.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error client gcs client: %w", err)
	}

	if bucket == "" {
		return nil, fmt.Errorf("missing bucket name")
	}

	if projectID == "" {
		return nil, fmt.Errorf("missing Google Cloud project ID")
	}

	return &GCSStorage{
		bucket:    bucket,
		projectID: projectID,
		client:    client,
	}, nil
}

func (s *GCSStorage) Class() string {
	return "gcs"
}

func (s *GCSStorage) Provision(ctx context.Context) error {
	bucket := s.client.Bucket(s.bucket)

	_, err := bucket.Attrs(ctx)
	if err == nil {
		return nil
	}

	if !errors.Is(err, gcs.ErrBucketNotExist) {
		return fmt.Errorf("get gcs bucket %q: %w", s.bucket, err)
	}

	err = bucket.Create(ctx, s.projectID, nil)
	if err != nil {
		return fmt.Errorf("create gcs bucket %q: %w", s.bucket, err)
	}

	return nil
}

func (s *GCSStorage) Write(ctx context.Context, path string, body io.Reader) error {
	writer := s.client.Bucket(s.bucket).Object(path).NewWriter(ctx)
	_, err := io.Copy(writer, body)
	if err != nil {
		_ = writer.Close()
		return fmt.Errorf("write gcs object %q: %w", path, err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("write gcs object %q: %w", path, err)
	}

	return nil
}

func (s *GCSStorage) Read(ctx context.Context, path string) (io.Reader, error) {
	reader, err := s.client.Bucket(s.bucket).Object(path).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("read gcs object %q: %w", path, err)
	}

	return reader, nil
}

func (s *GCSStorage) Delete(ctx context.Context, path string) error {
	err := s.client.Bucket(s.bucket).Object(path).Delete(ctx)
	if err != nil {
		return fmt.Errorf("delete gcs object %q: %w", path, err)
	}

	return nil
}
