package blobs

import (
	"bytes"
	"context"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

type memoryBlob struct {
	data        []byte
	contentType string
	updatedAt   time.Time
}

type MemoryStorage struct {
	mu    sync.RWMutex
	blobs map[string]memoryBlob
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		blobs: make(map[string]memoryBlob),
	}
}

func (s *MemoryStorage) Put(_ context.Context, scope Scope, path string, body io.Reader, opts PutOptions) error {
	key, err := objectKey(scope, path)
	if err != nil {
		return err
	}

	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.blobs[key] = memoryBlob{
		data:        data,
		contentType: opts.ContentType,
		updatedAt:   time.Now(),
	}
	s.mu.Unlock()

	return nil
}

func (s *MemoryStorage) Get(_ context.Context, scope Scope, path string) (io.ReadCloser, error) {
	key, err := objectKey(scope, path)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	blob, ok := s.blobs[key]
	s.mu.RUnlock()

	if !ok {
		return nil, ErrBlobNotFound
	}

	return io.NopCloser(bytes.NewReader(blob.data)), nil
}

func (s *MemoryStorage) Delete(_ context.Context, scope Scope, path string) error {
	key, err := objectKey(scope, path)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.blobs[key]; !ok {
		return ErrBlobNotFound
	}

	delete(s.blobs, key)
	return nil
}

func (s *MemoryStorage) List(_ context.Context, scope Scope, input ListInput) (*ListOutput, error) {
	prefix, err := scopePrefix(scope)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	var keys []string
	for k := range s.blobs {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	s.mu.RUnlock()

	sort.Strings(keys)

	startIdx := 0
	if input.ContinuationToken != "" {
		for i, k := range keys {
			if k > input.ContinuationToken {
				startIdx = i
				break
			}
			if i == len(keys)-1 {
				return &ListOutput{}, nil
			}
		}
	}

	maxResults := input.MaxResults
	if maxResults <= 0 {
		maxResults = 100
	}

	endIdx := startIdx + maxResults
	if endIdx > len(keys) {
		endIdx = len(keys)
	}

	s.mu.RLock()
	blobs := make([]BlobInfo, 0, endIdx-startIdx)
	for _, k := range keys[startIdx:endIdx] {
		blob := s.blobs[k]
		blobs = append(blobs, BlobInfo{
			Path:        strings.TrimPrefix(k, prefix),
			Size:        int64(len(blob.data)),
			ContentType: blob.contentType,
			UpdatedAt:   blob.updatedAt,
		})
	}
	s.mu.RUnlock()

	var nextToken string
	if endIdx < len(keys) {
		nextToken = keys[endIdx-1]
	}

	return &ListOutput{
		Blobs:     blobs,
		NextToken: nextToken,
	}, nil
}

func (s *MemoryStorage) PresignPut(_ context.Context, _ Scope, _ string, _ PutOptions, _ time.Duration) (*PresignedURL, error) {
	return nil, ErrPresignedURLNotSupported
}

func (s *MemoryStorage) PresignGet(_ context.Context, _ Scope, _ string, _ time.Duration) (*PresignedURL, error) {
	return nil, ErrPresignedURLNotSupported
}
