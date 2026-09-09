package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/superplanehq/superplane/pkg/blob"
	"google.golang.org/api/option"
)

type Store struct {
	client *storage.Client
	bucket string
}

func NewProvider() (*Store, error) {
	bucket := strings.TrimSpace(os.Getenv("BLOB_STORAGE_BUCKET"))
	if bucket == "" {
		return nil, fmt.Errorf("BLOB_STORAGE_BUCKET is not set")
	}

	ctx := context.Background()
	var opts []option.ClientOption
	if credentialsFile := strings.TrimSpace(os.Getenv("BLOB_STORAGE_CREDENTIALS_FILE")); credentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsFile))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create GCS client: %w", err)
	}

	return &Store{client: client, bucket: bucket}, nil
}

func (s *Store) Name() string {
	return blob.ProviderGCS
}

func (s *Store) Put(ctx context.Context, key string, r io.Reader, opts blob.PutOptions) error {
	writer := s.client.Bucket(s.bucket).Object(key).NewWriter(ctx)
	if strings.TrimSpace(opts.ContentType) != "" {
		writer.ContentType = opts.ContentType
	}
	if _, err := io.Copy(writer, r); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write GCS object: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close GCS object: %w", err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	reader, err := s.client.Bucket(s.bucket).Object(key).NewReader(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, blob.ErrNotFound
		}
		return nil, fmt.Errorf("read GCS object: %w", err)
	}
	return reader, nil
}

func (s *Store) Head(ctx context.Context, key string) (*blob.ObjectInfo, error) {
	attrs, err := s.client.Bucket(s.bucket).Object(key).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, blob.ErrNotFound
		}
		return nil, fmt.Errorf("head GCS object: %w", err)
	}
	return &blob.ObjectInfo{
		Size:        attrs.Size,
		ContentType: attrs.ContentType,
	}, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	err := s.client.Bucket(s.bucket).Object(key).Delete(ctx)
	if err != nil && !errors.Is(err, storage.ErrObjectNotExist) {
		return fmt.Errorf("delete GCS object: %w", err)
	}
	return nil
}

func (s *Store) SignedGetURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return s.client.Bucket(s.bucket).SignedURL(key, &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(ttl),
		QueryParameters: url.Values{
			blob.SignedURLMarkerParam: {blob.SignedURLMarkerValue},
		},
	})
}
