package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/blob"
)

type Store struct {
	root string
}

func NewProvider() (*Store, error) {
	root := strings.TrimSpace(os.Getenv("BLOB_STORAGE_LOCAL_PATH"))
	if root == "" {
		return nil, fmt.Errorf("BLOB_STORAGE_LOCAL_PATH is not set")
	}
	return New(root)
}

func New(root string) (*Store, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("blob filesystem root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create blob filesystem root: %w", err)
	}
	return &Store{root: root}, nil
}

func (s *Store) Name() string {
	return blob.ProviderFilesystem
}

func (s *Store) Put(_ context.Context, key string, r io.Reader, _ blob.PutOptions) error {
	path, err := s.pathFor(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create blob directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create blob file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, r); err != nil {
		return fmt.Errorf("write blob file: %w", err)
	}
	return nil
}

func (s *Store) Get(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := s.pathFor(key)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, blob.ErrNotFound
		}
		return nil, fmt.Errorf("open blob file: %w", err)
	}
	return file, nil
}

func (s *Store) Head(_ context.Context, key string) (*blob.ObjectInfo, error) {
	path, err := s.pathFor(key)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, blob.ErrNotFound
		}
		return nil, fmt.Errorf("stat blob file: %w", err)
	}
	return &blob.ObjectInfo{Size: info.Size()}, nil
}

func (s *Store) Delete(_ context.Context, key string) error {
	path, err := s.pathFor(key)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete blob file: %w", err)
	}
	return nil
}

func (s *Store) SignedGetURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", blob.ErrSignedURLUnsupported
}

func (s *Store) pathFor(key string) (string, error) {
	if strings.TrimSpace(key) == "" || strings.Contains(key, "..") || strings.HasPrefix(key, "/") {
		return "", fmt.Errorf("invalid blob key")
	}
	cleaned := filepath.Clean("/" + key)
	relative := strings.TrimPrefix(cleaned, "/")
	if relative == "" {
		return "", fmt.Errorf("invalid blob key")
	}
	full := filepath.Join(s.root, filepath.FromSlash(relative))
	root := filepath.Clean(s.root) + string(os.PathSeparator)
	if full != filepath.Clean(s.root) && !strings.HasPrefix(full, root) {
		return "", fmt.Errorf("invalid blob key")
	}
	return full, nil
}
