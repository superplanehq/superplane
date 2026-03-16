package storage

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalFolderStorage struct {
	rootDir string
}

func NewLocalFolderStorage(rootDir string) *LocalFolderStorage {
	return &LocalFolderStorage{rootDir: rootDir}
}

func (s *LocalFolderStorage) Write(path string, reader io.Reader) error {
	resolvedPath, err := s.resolvePath(path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return fmt.Errorf("create storage directory: %w", err)
	}

	file, err := os.Create(resolvedPath)
	if err != nil {
		return fmt.Errorf("create storage file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("write storage file: %w", err)
	}

	return nil
}

func (s *LocalFolderStorage) Read(path string) (io.Reader, error) {
	resolvedPath, err := s.resolvePath(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(data), nil
}

func (s *LocalFolderStorage) resolvePath(path string) (string, error) {
	if path == "" {
		return "", errors.New("storage path is required")
	}

	cleanPath := filepath.Clean(filepath.FromSlash(path))
	if cleanPath == "." || cleanPath == ".." {
		return "", errors.New("storage path is invalid")
	}
	if filepath.IsAbs(cleanPath) {
		return "", errors.New("storage path must be relative")
	}
	if strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", errors.New("storage path must stay within root dir")
	}

	return filepath.Join(s.rootDir, cleanPath), nil
}
