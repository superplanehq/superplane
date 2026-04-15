package blobs

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	gcsstorage "cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

type GCSStorage struct {
	bucket string
	client *gcsstorage.Client
}

func NewGCSStorage(bucket string, client *gcsstorage.Client) *GCSStorage {
	return &GCSStorage{
		bucket: bucket,
		client: client,
	}
}

func (s *GCSStorage) Put(ctx context.Context, scope Scope, path string, body io.Reader, opts PutOptions) error {
	key, err := objectKey(scope, path)
	if err != nil {
		return err
	}
	obj := s.client.Bucket(s.bucket).Object(key)
	writer := obj.NewWriter(ctx)

	if opts.ContentType != "" {
		writer.ContentType = opts.ContentType
	}

	if _, err := io.Copy(writer, body); err != nil {
		writer.Close()
		return err
	}

	return writer.Close()
}

func (s *GCSStorage) Get(ctx context.Context, scope Scope, path string) (io.ReadCloser, error) {
	key, err := objectKey(scope, path)
	if err != nil {
		return nil, err
	}
	obj := s.client.Bucket(s.bucket).Object(key)

	reader, err := obj.NewReader(ctx)
	if err != nil {
		if errors.Is(err, gcsstorage.ErrObjectNotExist) {
			return nil, ErrBlobNotFound
		}
		return nil, err
	}

	return reader, nil
}

func (s *GCSStorage) Delete(ctx context.Context, scope Scope, path string) error {
	key, err := objectKey(scope, path)
	if err != nil {
		return err
	}
	obj := s.client.Bucket(s.bucket).Object(key)

	if err := obj.Delete(ctx); err != nil {
		if errors.Is(err, gcsstorage.ErrObjectNotExist) {
			return ErrBlobNotFound
		}
		return err
	}

	return nil
}

func (s *GCSStorage) List(ctx context.Context, scope Scope, input ListInput) (*ListOutput, error) {
	prefix, err := scopePrefix(scope)
	if err != nil {
		return nil, err
	}

	maxResults := 100
	if input.MaxResults > 0 && input.MaxResults <= 1000 {
		maxResults = input.MaxResults
	}

	query := &gcsstorage.Query{Prefix: prefix}
	it := s.client.Bucket(s.bucket).Objects(ctx, query)

	if input.ContinuationToken != "" {
		it.PageInfo().Token = input.ContinuationToken
	}

	it.PageInfo().MaxSize = maxResults

	var blobs []BlobInfo
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		blobs = append(blobs, BlobInfo{
			Path:        strings.TrimPrefix(attrs.Name, prefix),
			Size:        attrs.Size,
			ContentType: attrs.ContentType,
			UpdatedAt:   attrs.Updated,
		})

		if len(blobs) >= maxResults {
			break
		}
	}

	return &ListOutput{
		Blobs:     blobs,
		NextToken: it.PageInfo().Token,
	}, nil
}

func (s *GCSStorage) PresignPut(ctx context.Context, scope Scope, path string, opts PutOptions, expiry time.Duration) (*PresignedURL, error) {
	key, err := objectKey(scope, path)
	if err != nil {
		return nil, err
	}

	signOpts := &gcsstorage.SignedURLOptions{
		Method:  "PUT",
		Expires: time.Now().Add(expiry),
	}

	if opts.ContentType != "" {
		signOpts.ContentType = opts.ContentType
	}

	url, err := s.client.Bucket(s.bucket).SignedURL(key, signOpts)
	if err != nil {
		return nil, err
	}

	return &PresignedURL{
		URL:       url,
		ExpiresAt: time.Now().Add(expiry),
	}, nil
}

func (s *GCSStorage) PresignGet(_ context.Context, scope Scope, path string, expiry time.Duration) (*PresignedURL, error) {
	key, err := objectKey(scope, path)
	if err != nil {
		return nil, err
	}

	url, err := s.client.Bucket(s.bucket).SignedURL(key, &gcsstorage.SignedURLOptions{
		Method:  "GET",
		Expires: time.Now().Add(expiry),
	})
	if err != nil {
		return nil, err
	}

	return &PresignedURL{
		URL:       url,
		ExpiresAt: time.Now().Add(expiry),
	}, nil
}
