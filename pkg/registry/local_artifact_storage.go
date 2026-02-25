package registry

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type LocalArtifactStorage struct {
	RootDirectory string
	ResourceType  string
	ResourceID    string
}

func (s *LocalArtifactStorage) Create(name string) (core.Artifact, error) {
	path, err := s.artifactPath(name)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("error creating directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("error creating file: %w", err)
	}

	return &LocalArtifact{file: file}, nil
}

func (s *LocalArtifactStorage) Get(name string) (core.Artifact, error) {
	path, err := s.artifactPath(name)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	return &LocalArtifact{file: file}, nil
}

func (s *LocalArtifactStorage) List() ([]string, error) {
	resourcePath, err := s.resourcePath()
	if err != nil {
		return nil, err
	}

	items, err := os.ReadDir(resourcePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []string{}, nil
		}

		return nil, fmt.Errorf("error listing artifacts: %w", err)
	}

	artifactNames := make([]string, 0, len(items))
	for _, item := range items {
		if item.IsDir() {
			continue
		}

		artifactNames = append(artifactNames, item.Name())
	}

	sort.Strings(artifactNames)

	return artifactNames, nil
}

func (s *LocalArtifactStorage) resourcePath() (string, error) {
	if strings.TrimSpace(s.RootDirectory) == "" {
		return "", fmt.Errorf("root directory is required")
	}

	if !isSafePathSegment(s.ResourceType) {
		return "", fmt.Errorf("invalid resource type")
	}

	if !isSafePathSegment(s.ResourceID) {
		return "", fmt.Errorf("invalid resource id")
	}

	return filepath.Join(s.RootDirectory, s.ResourceType, s.ResourceID), nil
}

func (s *LocalArtifactStorage) artifactPath(name string) (string, error) {
	if !isSafePathSegment(name) {
		return "", fmt.Errorf("invalid artifact name")
	}

	resourcePath, err := s.resourcePath()
	if err != nil {
		return "", err
	}

	return filepath.Join(resourcePath, name), nil
}

func isSafePathSegment(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}

	if value == "." || value == ".." {
		return false
	}

	return !strings.Contains(value, "/") && !strings.Contains(value, "\\")
}

type LocalArtifact struct {
	file *os.File
}

func (a *LocalArtifact) Write(data []byte) (int, error) {
	return a.file.Write(data)
}

func (a *LocalArtifact) Read(p []byte) (int, error) {
	return a.file.Read(p)
}

func (a *LocalArtifact) Close() error {
	return a.file.Close()
}
