package blobstorage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sync"
)

type memoryBlob struct {
	data        []byte
	contentType string
}

type InMemoryBlobStorage struct {
	mu    sync.RWMutex
	blobs map[string]memoryBlob
}

func NewInMemoryBlobStorage() *InMemoryBlobStorage {
	return &InMemoryBlobStorage{
		blobs: map[string]memoryBlob{},
	}
}

func (s *InMemoryBlobStorage) Put(_ context.Context, input PutInput) (*PutOutput, error) {
	content, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256(content)
	etag := hex.EncodeToString(sum[:])

	s.mu.Lock()
	s.blobs[input.Key] = memoryBlob{
		data:        content,
		contentType: input.ContentType,
	}
	s.mu.Unlock()

	return &PutOutput{ETag: etag}, nil
}

func (s *InMemoryBlobStorage) Get(_ context.Context, key string) (*GetOutput, error) {
	s.mu.RLock()
	blob, ok := s.blobs[key]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrBlobNotFound
	}

	return &GetOutput{
		Body:        io.NopCloser(bytes.NewReader(blob.data)),
		Size:        int64(len(blob.data)),
		ContentType: blob.contentType,
	}, nil
}

func (s *InMemoryBlobStorage) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.blobs[key]; !ok {
		return ErrBlobNotFound
	}

	delete(s.blobs, key)
	return nil
}
