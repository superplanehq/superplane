package blobstorage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FilesystemBlobStorage struct {
	basePath string
}

func NewFilesystemBlobStorage(basePath string) *FilesystemBlobStorage {
	return &FilesystemBlobStorage{basePath: basePath}
}

func (s *FilesystemBlobStorage) Put(_ context.Context, input PutInput) (*PutOutput, error) {
	fullPath, err := s.resolvePath(input.Key)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return nil, err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)
	if _, err := io.Copy(writer, input.Body); err != nil {
		return nil, err
	}

	return &PutOutput{ETag: hex.EncodeToString(hasher.Sum(nil))}, nil
}

func (s *FilesystemBlobStorage) Get(_ context.Context, key string) (*GetOutput, error) {
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

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	return &GetOutput{
		Body: file,
		Size: info.Size(),
	}, nil
}

func (s *FilesystemBlobStorage) Delete(_ context.Context, key string) error {
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

func (s *FilesystemBlobStorage) resolvePath(key string) (string, error) {
	cleanKey := strings.TrimSpace(key)
	cleanKey = strings.TrimPrefix(cleanKey, "/")
	if cleanKey == "" {
		return "", errors.New("blob key is required")
	}

	base := filepath.Clean(s.basePath)
	fullPath := filepath.Join(base, cleanKey)
	cleanFullPath := filepath.Clean(fullPath)

	rel, err := filepath.Rel(base, cleanFullPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("blob key points outside base path")
	}

	return cleanFullPath, nil
}
