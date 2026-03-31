package blobs

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FilesystemStorage struct {
	basePath string
}

func NewFilesystemStorage(basePath string) *FilesystemStorage {
	return &FilesystemStorage{basePath: basePath}
}

func (s *FilesystemStorage) Put(_ context.Context, scope Scope, blobPath string, body io.Reader, _ PutOptions) error {
	key, err := objectKey(scope, blobPath)
	if err != nil {
		return err
	}

	fullPath, err := s.resolvePath(key)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, body)
	return err
}

func (s *FilesystemStorage) Get(_ context.Context, scope Scope, blobPath string) (io.ReadCloser, error) {
	key, err := objectKey(scope, blobPath)
	if err != nil {
		return nil, err
	}

	fullPath, err := s.resolvePath(key)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrBlobNotFound
		}
		return nil, err
	}

	return file, nil
}

func (s *FilesystemStorage) Delete(_ context.Context, scope Scope, blobPath string) error {
	key, err := objectKey(scope, blobPath)
	if err != nil {
		return err
	}

	fullPath, err := s.resolvePath(key)
	if err != nil {
		return err
	}

	if err := os.Remove(fullPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrBlobNotFound
		}
		return err
	}

	return nil
}

func (s *FilesystemStorage) List(_ context.Context, scope Scope, input ListInput) (*ListOutput, error) {
	prefix, err := scopePrefix(scope)
	if err != nil {
		return nil, err
	}

	baseDir, err := s.resolvePath(prefix)
	if err != nil {
		return &ListOutput{}, nil
	}

	var keys []string
	_ = filepath.Walk(baseDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		rel, relErr := filepath.Rel(filepath.Clean(s.basePath), path)
		if relErr != nil {
			return nil
		}

		keys = append(keys, filepath.ToSlash(rel))
		return nil
	})

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

	blobs := make([]BlobInfo, 0, endIdx-startIdx)
	for _, k := range keys[startIdx:endIdx] {
		resolved, resolveErr := s.resolvePath(k)
		if resolveErr != nil {
			continue
		}

		info, statErr := os.Stat(resolved)
		if statErr != nil {
			continue
		}

		blobs = append(blobs, BlobInfo{
			Path:      strings.TrimPrefix(k, prefix),
			Size:      info.Size(),
			UpdatedAt: info.ModTime(),
		})
	}

	var nextToken string
	if endIdx < len(keys) {
		nextToken = keys[endIdx-1]
	}

	return &ListOutput{
		Blobs:     blobs,
		NextToken: nextToken,
	}, nil
}

func (s *FilesystemStorage) PresignPut(_ context.Context, _ Scope, _ string, _ PutOptions, _ time.Duration) (*PresignedURL, error) {
	return nil, ErrPresignedURLNotSupported
}

func (s *FilesystemStorage) PresignGet(_ context.Context, _ Scope, _ string, _ time.Duration) (*PresignedURL, error) {
	return nil, ErrPresignedURLNotSupported
}

func (s *FilesystemStorage) resolvePath(key string) (string, error) {
	base := filepath.Clean(s.basePath)
	fullPath := filepath.Clean(filepath.Join(base, key))

	rel, err := filepath.Rel(base, fullPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("blob key points outside base path")
	}

	return fullPath, nil
}
