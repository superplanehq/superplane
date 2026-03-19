package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

/*
 * LocalStorage is a storage implementation that uses the local filesystem.
 * Useful for testing and development environments.
 */
type LocalStorage struct {
	basePath       string
	organizationID string
}

func NewLocalStorage(basePath, organizationID string) *LocalStorage {
	return &LocalStorage{
		basePath:       basePath,
		organizationID: organizationID,
	}
}

func (s *LocalStorage) Class() string {
	return "local"
}

func (s *LocalStorage) Provision(ctx context.Context) error {
	return os.MkdirAll(s.base(), 0755)
}

func (s *LocalStorage) Write(ctx context.Context, path string, body io.Reader) error {
	file, err := os.Create(filepath.Join(s.base(), path))
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, body)
	if err != nil {
		return err
	}

	return nil
}

func (s *LocalStorage) Read(ctx context.Context, path string) (io.Reader, error) {
	file, err := os.Open(filepath.Join(s.base(), path))
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	return os.Remove(filepath.Join(s.base(), path))
}

func (s *LocalStorage) base() string {
	return filepath.Join(s.basePath, s.organizationID)
}
